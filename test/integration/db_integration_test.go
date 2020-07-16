// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package integration

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/clients"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestIntegrationWithDb_GetRecords(t *testing.T) {
	t.Log("C4991 Process records and get saved records by pulse number from database")
	ts := NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)
	records := make([]*exporter.Record, 0)

	pulsesNumber := 10
	recordsInPulse := 1
	recordsWithDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(pulsesNumber, recordsInPulse)
	stream, err := ts.ConMngr.ImporterClient.Import(context.Background())
	require.NoError(t, err)

	ts.BE.PulseClient.NextFinalizedPulseFunc = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		p := uint32(ts.ConMngr.Importer.GetLowestUnsentPulse())
		if p == 1<<32-1 {
			return nil, errors.New("unready yet")
		}
		return clients.GetFullPulse(p), nil
	}

	for i := 0; i < pulsesNumber; i++ {
		record, _ := recordsWithDifferencePulses()
		records = append(records, record)
		if err := stream.Send(record); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal("Error sending to stream", err)
		}
	}
	reply, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.True(t, reply.Ok)
	require.Len(t, records, pulsesNumber)

	jetDrops := make([]types.PlatformJetDrops, 0)
	for _, r := range records {
		jetDrop := types.PlatformJetDrops{Pulse: clients.GetFullPulse(uint32(r.Record.ID.Pulse())),
			Records: []*exporter.Record{r}}
		jetDrops = append(jetDrops, jetDrop)
	}
	require.Len(t, jetDrops, pulsesNumber)

	ts.StartBE(t)
	defer ts.StopBE(t)

	refs := make([]types.Reference, 0)
	ctx := context.Background()
	for _, jd := range jetDrops {
		transform, err := transformer.Transform(ctx, &jd)
		if err != nil {
			t.Logf("error transforming record: %v", err)
			return
		}
		for _, tr := range transform {
			r := tr.MainSection.Records
			require.NotEmpty(t, r)
			ref := r[0].Ref
			require.NotEmpty(t, ref)
			refs = append(refs, ref)
		}
	}
	require.Len(t, refs, pulsesNumber)

	// last record with the biggest pulse number won't be processed, so we do not expect this record in DB
	expRecordsCount := recordsInPulse * (pulsesNumber - 1)
	ts.WaitRecordsCount(t, expRecordsCount, 6000)

	for _, ref := range refs[:expRecordsCount] {
		modelRef := models.ReferenceFromTypes(ref)
		record, err := ts.BE.Storage().GetRecord(modelRef)
		require.NoError(t, err, "Error executing GetRecord from db")
		require.NotEmpty(t, record, "Record is empty")
		require.Equal(t, modelRef, record.Reference, "Reference not equal")
	}
}

func TestIntegrationWithDb_GetJetDrops(t *testing.T) {
	t.Log("C4992 Process records and get saved jetDrops by pulse number from database")

	ts := NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)

	recordsCount := 2
	pulses := 2
	expRecordsJet1 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecordsJet2 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecords := make([]*exporter.Record, 0)
	expRecords = append(expRecords, expRecordsJet1...)
	expRecords = append(expRecords, expRecordsJet2...)

	pulseNumbers := map[int64]bool{}
	for _, r := range expRecords {
		pulseNumbers[int64(r.Record.ID.Pulse())] = true
	}

	ts.BE.PulseClient.NextFinalizedPulseFunc = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		p := uint32(ts.ConMngr.Importer.GetLowestUnsentPulse())
		if p == 1<<32-1 {
			return nil, errors.New("unready yet")
		}
		return clients.GetFullPulse(p), nil
	}

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, expRecords)
	require.NoError(t, err)

	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(expRecords), 6000)

	var jetDropsDB []models.JetDrop
	for pulse, _ := range pulseNumbers {
		jd, err := ts.BE.Storage().GetJetDrops(models.Pulse{PulseNumber: pulse})
		require.NoError(t, err)
		jetDropsDB = append(jetDropsDB, jd...)
	}

	require.Len(t, jetDropsDB, recordsCount*pulses, "jetDrops count in db not as expected")

	prefixFirst := converter.JetIDToString(expRecordsJet1[0].Record.JetID)
	prefixSecond := converter.JetIDToString(expRecordsJet1[1].Record.JetID)
	prefixThird := converter.JetIDToString(expRecordsJet2[0].Record.JetID)
	jds := []string{jetDropsDB[0].JetID, jetDropsDB[1].JetID, jetDropsDB[2].JetID}
	require.Contains(t, jds, prefixFirst)
	require.Contains(t, jds, prefixSecond)
	require.Contains(t, jds, prefixThird)
	require.Equal(t, recordsCount, jetDropsDB[0].RecordAmount)
	require.Equal(t, recordsCount, jetDropsDB[1].RecordAmount)
	require.Equal(t, recordsCount, jetDropsDB[2].RecordAmount)
}
