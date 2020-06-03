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

	"github.com/insolar/block-explorer/etl/controller"
	"github.com/insolar/block-explorer/etl/extractor"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/processor"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/connection_manager"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type dbIntegrationSuite struct {
	suite.Suite
	c connection_manager.ConnectionManager
}

func (a *dbIntegrationSuite) SetupSuite() {
	a.c.Start(a.T())
	a.c.StartDB(a.T())
}

func (a *dbIntegrationSuite) TearDownSuite() {
	a.c.Stop()
}

func (a *dbIntegrationSuite) TestGetRecordsFromDb() {
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
	extractorMn := extractor.NewPlatformExtractor(100, a.c.ExporterClient)
	err = extractorMn.Start(ctx)
	require.NoError(a.T(), err)
	defer extractorMn.Stop(ctx)

	transformerJdChan := make(chan *types.PlatformJetDrops)
	transformerMn := transformer.NewMainNetTransformer(transformerJdChan)
	err = transformerMn.Start(ctx)
	require.NoError(a.T(), err)
	defer transformerMn.Stop(ctx)

	jetDrops := extractorMn.GetJetDrops(ctx)
	refs := make([]types.Reference, 0)
	for i := 0; i < recordsCount; i++ {
		jd := <-jetDrops
		transformerJdChan <- jd
		transform, err := transformer.Transform(ctx, jd)
		if err != nil {
			a.T().Logf("error transforming record: %v", err)
			return
		}
		for _, t := range transform {
			refs = append(refs, t.MainSection.Records[0].Ref)
		}
	}

	s := storage.NewStorage(a.c.DB)
	contr, err := controller.NewController(extractorMn, s)
	require.NoError(a.T(), err)
	proc := processor.NewProcessor(transformerMn, s, contr, 1)
	proc.Start(ctx)
	defer proc.Stop(ctx)

	a.waitRecordsCount(recordsCount - 1)

	for _, ref := range refs {
		modelRef := models.ReferenceFromTypes(ref)
		record, err := s.GetRecord(modelRef)
		require.NoError(a.T(), err, "Error executing GetRecord from db")
		require.NotEmpty(a.T(), record, "Record is empty")
		require.Equal(a.T(), modelRef, record.Reference, "Reference not equal")
	}
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
