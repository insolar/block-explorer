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

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	betest "github.com/insolar/block-explorer/testutils/betestsetup"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	a.c.Stop()
}

func (a *dbIntegrationSuite) TestIntegrationWithDb_GetRecords() {
	recordsCount := 10
	recordsWithDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(recordsCount, 1)
	stream, err := a.c.ImporterClient.Import(context.Background())
	require.NoError(a.T(), err)

	for i := 0; i < recordsCount; i++ {
		record, _ := recordsWithDifferencePulses()
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
	require.Len(a.T(), a.c.Importer.GetSavedRecords(), recordsCount) // because recordsWithDifferencePulses generates 3 records

	ctx := context.Background()
	jetDrops := a.be.Extractor().GetJetDrops(ctx)
	refs := make([]types.Reference, 0)
	counter := 0
	for counter < recordsCount {
		select {
		case jd := <-jetDrops:
			transform, err := transformer.Transform(ctx, jd)
			if err != nil {
				a.T().Logf("error transforming record: %v", err)
				return
			}
			for _, t := range transform {
				refs = append(refs, t.MainSection.Records[0].Ref)
			}
			counter++
		case <-time.After(1000 * time.Millisecond):
			a.T().Fatalf("Timeout waiting for records: expected %v, got %v, saved in importer %v",
				recordsCount, counter, len(a.c.Importer.GetSavedRecords()))
		}
	}

	a.waitRecordsCount(recordsCount)

	for _, ref := range refs {
		modelRef := models.ReferenceFromTypes(ref)
		record, err := a.be.Storage().GetRecord(modelRef)
		require.NoError(a.T(), err, "Error executing GetRecord from db")
		require.NotEmpty(a.T(), record, "Record is empty")
		require.Equal(a.T(), modelRef, record.Reference, "Reference not equal")
	}
}

func (a *dbIntegrationSuite) TestIntegrationWithDb_GetJetDrops() {
	recordsCount := 2
	pulses := 2
	expRecordsJet1 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecordsJet2 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
	expRecords := make([]*exporter.Record, 0)
	expRecords = append(expRecords, expRecordsJet1...)
	expRecords = append(expRecords, expRecordsJet2...)

	err := heavymock.ImportRecords(a.c.ImporterClient, expRecords)
	require.NoError(a.T(), err)

	a.waitRecordsCount(len(expRecords))

	// TODO: change it to '{PulseNumber: int(pulse)}' at PENV-212
	jetDropsDB, err := a.be.Storage().GetJetDrops(models.Pulse{PulseNumber: 1})
	require.NoError(a.T(), err)
	require.Len(a.T(), jetDropsDB, 2, "jetDrops count in db not as expected")

	prefixFirst := expRecordsJet1[0].Record.JetID.Prefix()
	prefixSecond := expRecordsJet1[0].Record.JetID.Prefix()
	jds := [][]byte{jetDropsDB[0].JetID, jetDropsDB[1].JetID}
	require.Contains(a.T(), jds, prefixFirst)
	require.Contains(a.T(), jds, prefixSecond)
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
