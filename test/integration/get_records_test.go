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

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type integrationSuite struct {
	suite.Suite
	s              *testutils.TestGRPCServer
	c              *connection.MainnetClient
	i              *heavymock.ImporterClient
	cfg            configuration.Replicator
	exporterClient exporter.RecordExporterClient
	importerClient heavymock.HeavymockImporterClient
}

func (a *integrationSuite) SetupSuite() {
	a.s = testutils.CreateTestGRPCServer(a.T())
	importer := heavymock.NewHeavymockImporter()
	heavymock.RegisterHeavymockImporterServer(a.s.Server, importer)
	exporter.RegisterRecordExporterServer(a.s.Server, heavymock.NewRecordExporter(importer))
	a.s.Serve(a.T())

	ctx := context.Background()
	a.cfg = configuration.Replicator{
		Addr:            a.s.GetAddress(),
		MaxTransportMsg: 100500,
	}
	c, err := connection.NewMainNetClient(ctx, a.cfg)
	require.NoError(a.T(), err)
	a.c = c

	i, err := heavymock.NewImporterClient(a.s.GetAddress())
	require.NoError(a.T(), err)
	a.i = i

	a.exporterClient = exporter.NewRecordExporterClient(a.c.GetGRPCConn())
	a.importerClient = heavymock.NewHeavymockImporterClient(a.i.GetGRPCConn())
}

func (a *integrationSuite) TearDownSuite() {
	a.s.Server.Stop()
	a.c.GetGRPCConn().Close()
	a.i.GetGRPCConn().Close()
}

func (a *integrationSuite) TestGetRecords_simpleRecord() {
	request := &exporter.GetRecords{
		Count: uint32(5),
	}

	stream, err := a.exporterClient.Export(context.Background(), request)
	require.NoError(a.T(), err, "Error when sending client request")

	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(a.T(), err, "Err listening stream")
		require.True(a.T(), heavymock.SimpleRecord.Equal(record), "Incorrect response message")
		a.T().Logf("received record: %v", record)
	}
}

func (a *integrationSuite) TestGetRecords_pulseRecords() {
	expPulse := gen.PulseNumber()
	request := &exporter.GetRecords{
		Count:       uint32(5),
		PulseNumber: expPulse,
	}

	stream, err := a.exporterClient.Export(context.Background(), request)
	require.NoError(a.T(), err, "Error when sending client request")

	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(a.T(), err, "Err listening stream")
		a.T().Logf("received record: %v", record)
		require.Equal(a.T(), &expPulse, record.ShouldIterateFrom, "Incorrect record pulse number")
	}
}

func (a *integrationSuite) TestGetRecords_sendAndReceiveWithImporter() {
	var expectedRecords []exporter.Record
	recordsCount := 10
	for i := 0; i < recordsCount; i++ {
		expectedRecords = append(expectedRecords, *heavymock.SimpleRecord)
	}

	stream, err := a.importerClient.Import(context.Background())
	require.NoError(a.T(), err)
	for _, record := range expectedRecords {
		if err := stream.Send(&record); err != nil {
			if err == io.EOF {
				break
			}
			a.T().Fatal("Error sending to stream", err)
		}
	}
	reply, err := stream.CloseAndRecv()
	require.NoError(a.T(), err)
	require.True(a.T(), reply.Ok)

	request := &exporter.GetRecords{
		Polymorph: heavymock.MagicPolymorphExport,
	}

	expStream, err := a.exporterClient.Export(context.Background(), request)
	require.NoError(a.T(), err, "Error when sending export request")

	var c int
	for {
		record, err := expStream.Recv()
		if err == io.EOF {
			break
		}
		c++
		require.NoError(a.T(), err, "Err listening stream")
		a.T().Logf("received record: %v", record)
		require.True(a.T(), heavymock.SimpleRecord.Equal(record), "Incorrect record pulse number")
	}
	require.Equal(a.T(), recordsCount, c)
}

func TestAllTests(t *testing.T) {
	suite.Run(t, new(integrationSuite))
}
