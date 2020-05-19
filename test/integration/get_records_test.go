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
	s   *testutils.TestGRPCServer
	c   *connection.MainnetClient
	t   *testing.T
	cfg configuration.Replicator
	cli exporter.RecordExporterClient
}

func (a *integrationTest) SetupSuite() {
	a.t = a.T()
	a.s = testutils.CreateTestGRPCServer(a.t)
	exporter.RegisterRecordExporterServer(a.s.Server, heavymock.NewRecordExporter())
	a.s.Serve(a.t)

	ctx := context.Background()
	a.cfg = configuration.Replicator{
		Addr:            a.s.GetPort(),
		MaxTransportMsg: 100500,
	}
	c, err := connection.NewMainNetClient(ctx, a.cfg)
	require.NoError(a.t, err)
	a.c = c

	a.cli = exporter.NewRecordExporterClient(a.c.GetGRPCConn())
}

func (a *integrationTest) TearDownSuite() {
	a.s.Server.Stop()
	a.c.GetGRPCConn().Close()
}

func (a *integrationTest) TestGetRecords_simpleRecord() {
	request := &exporter.GetRecords{
		Count: uint32(5),
	}

	stream, err := a.cli.Export(context.Background(), request)
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

	stream, err := a.cli.Export(context.Background(), request)
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

func TestAllTests(t *testing.T) {
	suite.Run(t, new(integrationTest))
}
