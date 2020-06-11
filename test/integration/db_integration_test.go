// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package integration

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	betest "github.com/insolar/block-explorer/testutils/betestsetup"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
)

type dbIntegrationSuite struct {
	suite.Suite
	c  connectionmanager.ConnectionManager
	be betest.BlockExplorerTestSetUp
}

func (a *dbIntegrationSuite) SetupTest() {
	a.c.Start(a.T())
	a.c.StartDB(a.T())

	a.be = betest.NewBlockExplorer(a.c.ExporterClient, a.c.DB)
	err := a.be.Start()
	require.NoError(a.T(), err)
}

func (a *dbIntegrationSuite) TearDownTest() {
	err := a.be.Stop()
	require.NoError(a.T(), err)
	// TODO remove sleep after resolving https://insolar.atlassian.net/browse/PENV-343
	time.Sleep(time.Second * 1)
	a.c.Stop()
}

func (a *dbIntegrationSuite) TestIntegrationWithDb_GetRecords() {
	a.T().Log("C4991 Process records and get saved records by pulse number from database")
	pulsesNumber := 10
	recordsInPulse := 1
	recordsWithDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(pulsesNumber, recordsInPulse)
	stream, err := a.c.ImporterClient.Import(context.Background())
	require.NoError(a.T(), err)

	records := make([]*exporter.Record, 0)
	for i := 0; i < pulsesNumber; i++ {
		record, _ := recordsWithDifferencePulses()
		records = append(records, record)
		if err := stream.Send(record); err != nil {
			if err == io.EOF {
				break
			}
			a.T().Fatal("Error sending to stream", err)
		}
	}
	reply, err := stream.CloseAndRecv()
	require.NoError(a.T(), err)
	require.True(a.T(), reply.Ok)
	require.Len(a.T(), records, pulsesNumber)

	jetDrops := make([]types.PlatformJetDrops, 0)
	for _, r := range records {
		jetDrop := types.PlatformJetDrops{Records: []*exporter.Record{r}}
		jetDrops = append(jetDrops, jetDrop)
	}
	require.Len(a.T(), jetDrops, pulsesNumber)

	refs := make([]types.Reference, 0)
	ctx := context.Background()
	for _, jd := range jetDrops {
		transform, err := transformer.Transform(ctx, &jd)
		if err != nil {
			a.T().Logf("error transforming record: %v", err)
			return
		}
		for _, t := range transform {
			r := t.MainSection.Records
			require.NotEmpty(a.T(), r)
			ref := r[0].Ref
			require.NotEmpty(a.T(), ref)
			refs = append(refs, ref)
		}
	}
	require.Len(a.T(), refs, pulsesNumber)

	// last record with the biggest pulse number won't be processed, so we do not expect this record in DB
	expRecordsCount := recordsInPulse * (pulsesNumber - 1)
	a.waitRecordsCount(expRecordsCount)

	for _, ref := range refs[:expRecordsCount] {
		modelRef := models.ReferenceFromTypes(ref)
		record, err := a.be.Storage().GetRecord(modelRef)
		require.NoError(a.T(), err, "Error executing GetRecord from db")
		require.NotEmpty(a.T(), record, "Record is empty")
		require.Equal(a.T(), modelRef, record.Reference, "Reference not equal")
	}
}

func (a *dbIntegrationSuite) TestIntegrationWithDb_GetJetDrops() {
	a.T().Log("C4992 Process records and get saved jetDrops by pulse number from database")
	recordsCount := 2
	pulses := 2
	expRecordsJet1 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecordsJet2 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecords := make([]*exporter.Record, 0)
	expRecords = append(expRecords, expRecordsJet1...)
	expRecords = append(expRecords, expRecordsJet2...)

	pulseNumbers := map[int]bool{}
	for _, r := range expRecords {
		pulseNumbers[int(r.Record.ID.Pulse())] = true
	}

	err := heavymock.ImportRecords(a.c.ImporterClient, expRecords)
	require.NoError(a.T(), err)

	// last records with the biggest pulse number won't be processed, so we do not expect this record in DB
	a.waitRecordsCount(len(expRecords) - recordsCount)

	var jetDropsDB []models.JetDrop
	for pulse, _ := range pulseNumbers {
		jd, err := a.be.Storage().GetJetDrops(models.Pulse{PulseNumber: pulse})
		require.NoError(a.T(), err)
		jetDropsDB = append(jetDropsDB, jd...)
	}

	require.Len(a.T(), jetDropsDB, 3, "jetDrops count in db not as expected")

	prefixFirst := expRecordsJet1[0].Record.JetID.Prefix()
	prefixSecond := expRecordsJet1[1].Record.JetID.Prefix()
	prefixThird := expRecordsJet2[0].Record.JetID.Prefix()
	jds := [][]byte{jetDropsDB[0].JetID, jetDropsDB[1].JetID, jetDropsDB[2].JetID}
	require.Contains(a.T(), jds, prefixFirst)
	require.Contains(a.T(), jds, prefixSecond)
	require.Contains(a.T(), jds, prefixThird)
	require.Equal(a.T(), recordsCount, jetDropsDB[0].RecordAmount)
	require.Equal(a.T(), recordsCount, jetDropsDB[1].RecordAmount)
	require.Equal(a.T(), recordsCount, jetDropsDB[2].RecordAmount)
}

func (a *dbIntegrationSuite) waitRecordsCount(expCount int) {
	var c int
	for i := 0; i < 60; i++ {
		record := models.Record{}
		a.c.DB.Model(&record).Count(&c)
		a.T().Logf("Select from record, expected rows count=%v, actual=%v, attempt: %v", expCount, c, i)
		if c >= expCount {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	a.T().Logf("Found %v rows", c)
	require.Equal(a.T(), expCount, c, "Records count in DB not as expected")
}

func TestAll(t *testing.T) {
	suite.Run(t, new(dbIntegrationSuite))
}
