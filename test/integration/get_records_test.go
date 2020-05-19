// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.
// +build mock_integration

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

type integrationTest struct {
	suite.Suite
	s      *testutils.TestGRPCServer
	c      *connection.MainnetClient
	i      *heavymock.ImporterClient
	t      *testing.T
	cfg    configuration.Replicator
	expcli exporter.RecordExporterClient
	impcli heavymock.HeavymockImporterClient
}

func (a *integrationTest) SetupSuite() {
	a.t = a.T()
	a.s = testutils.CreateTestGRPCServer(a.t)
	importer := heavymock.NewHeavymockImporter()
	heavymock.RegisterHeavymockImporterServer(a.s.Server, importer)
	exporter.RegisterRecordExporterServer(a.s.Server, heavymock.NewRecordExporter(importer))
	a.s.Serve(a.t)

	ctx := context.Background()
	a.cfg = configuration.Replicator{
		Addr:            a.s.GetPort(),
		MaxTransportMsg: 100500,
	}
	c, err := connection.NewMainNetClient(ctx, a.cfg)
	require.NoError(a.t, err)
	a.c = c

	i, err := heavymock.NewImporterClient(a.s.GetPort())
	require.NoError(a.t, err)
	a.i = i

	a.expcli = exporter.NewRecordExporterClient(a.c.GetGRPCConn())
	a.impcli = heavymock.NewHeavymockImporterClient(a.i.GetGRPCConn())
}

func (a *integrationTest) TearDownSuite() {
	a.s.Server.Stop()
	a.c.GetGRPCConn().Close()
	a.i.GetGRPCConn().Close()
}

func (a *integrationTest) TestGetRecords_simpleRecord() {
	request := &exporter.GetRecords{
		Count: uint32(5),
	}

	stream, err := a.expcli.Export(context.Background(), request)
	require.NoError(a.t, err, "Error when sending client request")

	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(a.t, err, "Err listening stream")
		require.Equal(a.t, heavymock.SimpleRecord, record, "Incorrect response message")
		a.t.Logf("received record: %v", record)
	}
}

func (a *integrationTest) TestGetRecords_pulseRecords() {
	expPulse := gen.PulseNumber()
	request := &exporter.GetRecords{
		Count:       uint32(5),
		PulseNumber: expPulse,
	}

	stream, err := a.expcli.Export(context.Background(), request)
	require.NoError(a.t, err, "Error when sending client request")

	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(a.t, err, "Err listening stream")
		a.t.Logf("received record: %v", record)
		require.Equal(a.t, &expPulse, record.ShouldIterateFrom, "Incorrect record pulse number")
	}
}

func (a *integrationTest) TestGetRecords_sendAndReceiveWithImporter() {
	var expectedRecords []exporter.Record
	recordsCount := 10
	for i := 0; i < recordsCount; i++ {
		expectedRecords = append(expectedRecords, *heavymock.SimpleRecord)
	}

	stream, err := a.impcli.Import(context.Background())
	require.NoError(a.t, err)
	for _, record := range expectedRecords {
		if err := stream.Send(&record); err != nil {
			if err == io.EOF {
				break
			}
			a.t.Fatal("Error sending to stream", err)
		}
	}
	reply, err := stream.CloseAndRecv()
	require.NoError(a.t, err)
	require.True(a.t, reply.Ok)

	request := &exporter.GetRecords{
		Polymorph: heavymock.MagicPolymorphExport,
	}

	expStream, err := a.expcli.Export(context.Background(), request)
	require.NoError(a.t, err, "Error when sending export request")

	var c int
	for {
		record, err := expStream.Recv()
		if err == io.EOF {
			break
		}
		c++
		require.NoError(a.t, err, "Err listening stream")
		a.t.Logf("received record: %v", record)
		require.True(a.t, heavymock.SimpleRecord.Equal(record), "Incorrect record pulse number")
	}
	require.Equal(a.t, recordsCount, c)
}

func TestAllTests(t *testing.T) {
	suite.Run(t, new(integrationTest))
}
