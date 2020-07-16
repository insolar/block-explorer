// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package heavymock

import (
	"context"
	"io"
	"testing"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	server := testutils.CreateTestGRPCServer(t)
	exporter.RegisterRecordExporterServer(server.Server, NewRecordExporter(&ImporterServer{}))
	server.Serve(t)
	defer server.Server.Stop()

	// prepare config with listening address
	cfg := configuration.Replicator{
		Addr:            server.Address,
		MaxTransportMsg: 100500,
	}

	// initialization Platform connection
	ctx := context.Background()
	client, err := connection.NewGRPCClientConnection(ctx, cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()

	greeterClient := exporter.NewRecordExporterClient(client.GetGRPCConn())
	// send record to stream
	request := &exporter.GetRecords{}
	stream, err := greeterClient.Export(context.Background(), request)
	require.NoError(t, err, "Error when sending client request")

	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%v.Export(_) = _, %v", client, err)
		}
		require.NoError(t, err, "Err listening stream")
		require.Equal(t, SimpleRecord.Record.ID.Pulse()+10, record.Record.ID.Pulse(), "Incorrect response message")
	}
}

func TestHeavymockImporter_storeAndSend(t *testing.T) {
	server := testutils.CreateTestGRPCServer(t)
	importer := NewHeavymockImporter()
	RegisterHeavymockImporterServer(server.Server, importer)
	exporter.RegisterRecordExporterServer(server.Server, NewRecordExporter(importer))
	server.Serve(t)
	defer server.Server.Stop()

	cfg := connection.GetClientConfiguration(server.Address)
	importerConn, err := connection.NewGRPCClientConnection(context.Background(), cfg)
	require.NoError(t, err)

	defer importerConn.GetGRPCConn().Close()

	importerCli := NewHeavymockImporterClient(importerConn.GetGRPCConn())
	exporterCli := exporter.NewRecordExporterClient(importerConn.GetGRPCConn())

	recordsPtOne := testutils.GenerateRecordsSilence(5)
	recordsPtTwo := testutils.GenerateRecordsSilence(10)

	pu := gen.PulseNumber()
	for i, _ := range recordsPtOne {
		recordsPtOne[i].Record.ID = *insolar.NewID(pu, nil)
	}
	for i, _ := range recordsPtTwo {
		recordsPtTwo[i].Record.ID = *insolar.NewID(pu, nil)
	}

	err = ImportRecords(importerCli, recordsPtOne)
	require.NoError(t, err)
	require.Len(t, importer.GetUnsentRecords(), len(recordsPtOne))

	err = ImportRecords(importerCli, recordsPtTwo)
	require.NoError(t, err)
	require.Len(t, importer.GetUnsentRecords(), len(recordsPtOne)+len(recordsPtTwo))

	received, err := ReceiveRecords(exporterCli, &exporter.GetRecords{PulseNumber: pu})
	require.NoError(t, err)
	require.Len(t, received, len(recordsPtOne)+len(recordsPtTwo)+1) // one nextpulse record
	require.Empty(t, importer.GetUnsentRecords())

	// send same records once again, then receive
	err = ImportRecords(importerCli, recordsPtOne)
	require.NoError(t, err)
	require.Len(t, importer.GetUnsentRecords(), len(recordsPtOne))
	received, err = ReceiveRecords(exporterCli, &exporter.GetRecords{PulseNumber: pu})
	require.NoError(t, err)
	require.Len(t, received, len(recordsPtOne)+1)
	require.Empty(t, importer.GetUnsentRecords())
}
