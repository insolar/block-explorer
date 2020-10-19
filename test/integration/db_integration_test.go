// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package integration

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/clients"
)

func TestIntegrationWithDb_GetRecords(t *testing.T) {
	t.Log("C4991 Process records and get saved records by pulse number from database")
	ts := NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)
	records := make([]*exporter.Record, 0)

	pulsesNumber := 10
	recordsInPulse := 1
	recordsWithDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(pulsesNumber, recordsInPulse, int64(pulse.MinTimePulse))
	stream, err := ts.ConMngr.ImporterClient.Import(context.Background())
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

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

	jetDrops := make([]types.PlatformPulseData, 0)
	jetsInPulse := map[insolar.PulseNumber][]exporter.JetDropContinue{}
	for _, r := range records {
		p, err := clients.GetFullPulse(uint32(r.Record.ID.Pulse()), nil)
		require.NoError(t, err)
		jetDrop := types.PlatformPulseData{Pulse: p,
			Records: []*exporter.Record{r}}
		jetDrops = append(jetDrops, jetDrop)
		jetsInPulse[r.Record.ID.Pulse()] = append(jetsInPulse[r.Record.ID.Pulse()], exporter.JetDropContinue{JetID: r.Record.JetID, Hash: testutils.GenerateRandBytes()})
	}
	require.Len(t, jetDrops, pulsesNumber)

	for _, jetDrop := range jetDrops {
		jetDrop.Pulse.Jets = jetsInPulse[jetDrop.Pulse.PulseNumber]
	}

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
			ref := r[0].Reference()
			require.NotEmpty(t, ref)
			refs = append(refs, ref)
		}
	}
	require.Len(t, refs, pulsesNumber)

	// last record with the biggest pulse number won't be processed, so we do not expect this record in DB
	expRecordsCount := recordsInPulse * (pulsesNumber - 1)
	ts.WaitRecordsCount(t, recordsInPulse*pulsesNumber, 6000)

	for _, ref := range refs[:expRecordsCount] {
		modelRef := models.ReferenceFromTypes(ref)
		record, err := ts.BE.Storage().GetRecord(modelRef)
		require.NoError(t, err, "Error executing GetRecord from db")
		require.NotEmpty(t, record, "Record is empty")
		require.Equal(t, modelRef, record.Reference, "Reference not equal")
	}
}

func TestIntegrationWithDb_GetRecords_ErrorSameRecords(t *testing.T) {
	t.Log("C5498 Process same records; duplicated records not saved in database")
	ts := NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)
	records := make([]*exporter.Record, 0)

	pulsesNumber := 2
	recordsInPulse := 5
	recordsWithDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(pulsesNumber, recordsInPulse, int64(pulse.MinTimePulse))
	stream, err := ts.ConMngr.ImporterClient.Import(context.Background())
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

	for i := 0; i < pulsesNumber*recordsInPulse; i++ {
		record, _ := recordsWithDifferencePulses()
		records = append(records, record)
		if err := stream.Send(record); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal("Error sending to stream", err)
		}
		// send record second times for first pulse
		if i < recordsInPulse {
			if err := stream.Send(record); err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal("Error sending to stream", err)
			}
		}
	}
	reply, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.True(t, reply.Ok)
	require.Len(t, records, pulsesNumber*recordsInPulse)

	jetDrops := make([]types.PlatformPulseData, 0)
	var notSavedRefs [][]byte
	var jetsInPulse []exporter.JetDropContinue
	var recs []*exporter.Record
	for i, r := range records {
		p, err := clients.GetFullPulse(uint32(r.Record.ID.Pulse()), nil)
		require.NoError(t, err)
		if p.PulseNumber == pulse.MinTimePulse {
			notSavedRefs = append(notSavedRefs, r.Record.ID.Bytes())
			continue
		}
		recs = append(recs, r)
		jetsInPulse = append(jetsInPulse, exporter.JetDropContinue{JetID: r.Record.JetID, Hash: testutils.GenerateRandBytes()})
		if i%recordsInPulse == (recordsInPulse - 1) {
			jetDrop := types.PlatformPulseData{Pulse: p,
				Records: recs}
			jetDrops = append(jetDrops, jetDrop)
			recs = []*exporter.Record{}
		}
	}

	require.Len(t, jetDrops, pulsesNumber-1)

	for _, jetDrop := range jetDrops {
		jetDrop.Pulse.Jets = jetsInPulse
	}

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
			records := tr.MainSection.Records
			require.NotEmpty(t, records)
			for _, r := range records {
				ref := r.Reference()
				require.NotEmpty(t, ref)
				refs = append(refs, ref)
			}
		}
	}
	require.Len(t, refs, recordsInPulse)

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
	for _, ref := range notSavedRefs {
		modelRef := models.ReferenceFromTypes(ref)
		_, err := ts.BE.Storage().GetRecord(modelRef)
		require.Error(t, err, "Record must be not saved")
		require.Equal(t, err.Error(), "record not found", "Wrong error message")
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

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

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

func TestIntegrationWithDb_GetPulse(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PENV-802")
	t.Log("C5648 Process records and get saved pulses by pulse number from database")

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

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, expRecords)
	require.NoError(t, err)

	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(expRecords), 6000)

	var pulsesDB []models.Pulse
	for pulse, _ := range pulseNumbers {
		p, err := ts.BE.Storage().GetPulse(pulse)
		require.NoError(t, err)
		pulsesDB = append(pulsesDB, p)
	}

	require.Len(t, pulsesDB, pulses, "pulses count in db not as expected")

	for i := 0; i < pulses; i++ {
		require.Contains(t, pulseNumbers, pulsesDB[i].PulseNumber)
		// there is two jetDrops in every pulse
		require.EqualValues(t, 2, pulsesDB[i].JetDropAmount)
		require.EqualValues(t, recordsCount*2, pulsesDB[i].RecordAmount)
	}
}

// save data for two pulses
// save data for first pulse again, change PrevPulseNumber and jetDrops hashes to new values
// check pulse in db have new value at PrevPulseNumber
func TestIntegrationWithDb_GetPulse_ReloadData(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PENV-802")
	t.Log("C5649 Process records twice (as if reloading data happened) and get saved pulse by pulse number from database")

	ts := NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)

	updatedPrevPulseNumber := insolar.PulseNumber(100000000)
	updatedHash := testutils.GenerateRandBytes()
	recordsCount := 2
	pulses := 2
	expRecordsJet1 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecordsJet2 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecords := make([]*exporter.Record, 0)
	expRecords = append(expRecords, expRecordsJet1...)
	expRecords = append(expRecords, expRecordsJet2...)

	pulseNumber := int64(expRecords[0].Record.ID.Pulse())

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

	var sendSamePulse int32
	var nextFinalizedPulseFirst *exporter.FullPulse
	var p uint32
	ts.BE.PulseClient.NextFinalizedPulseFunc = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		if atomic.LoadInt32(&sendSamePulse) == 1 {
			if p == uint32(nextFinalizedPulseFirst.PulseNumber) {
				return nil, errors.New("unready yet")
			}
			p = uint32(nextFinalizedPulseFirst.PulseNumber)
			nextFinalizedPulseFirst.PrevPulseNumber = updatedPrevPulseNumber
			for i := 0; i < len(nextFinalizedPulseFirst.Jets); i++ {
				nextFinalizedPulseFirst.Jets[i].Hash = updatedHash
			}
			return nextFinalizedPulseFirst, nil
		}
		pulse, jetDropContinue := ts.ConMngr.Importer.GetLowestUnsentPulse()
		if p == uint32(pulse) {
			return nil, errors.New("unready yet")
		}
		p = uint32(pulse)
		fullPulse, err := clients.GetFullPulse(p, jetDropContinue)
		if nextFinalizedPulseFirst == nil {
			nextFinalizedPulseFirst = fullPulse
		}
		return fullPulse, err
	}

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, expRecords)
	require.NoError(t, err)

	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(expRecords), 6000)

	pulse, err := ts.BE.Storage().GetPulse(pulseNumber)
	require.NoError(t, err)

	require.Equal(t, pulseNumber, pulse.PulseNumber)
	require.EqualValues(t, 2, pulse.JetDropAmount)
	require.EqualValues(t, 2*recordsCount, pulse.RecordAmount)

	atomic.AddInt32(&sendSamePulse, 1)

	expectedPulse := pulse
	expectedPulse.PrevPulseNumber = int64(updatedPrevPulseNumber)
	for i := 0; i < len(nextFinalizedPulseFirst.Jets); i++ {
		jdID := models.NewJetDropID(converter.JetIDToString(nextFinalizedPulseFirst.Jets[i].JetID), pulse.PulseNumber)
		ts.WaitJetDropHash(t, *jdID, updatedHash, 6000)
	}

	pulse, err = ts.BE.Storage().GetPulse(pulseNumber)
	require.NoError(t, err)

	require.EqualValues(t, 2, pulse.JetDropAmount)
	require.EqualValues(t, 2*recordsCount, pulse.RecordAmount)
}
