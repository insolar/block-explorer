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

	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestHeavymockImporter_import(t *testing.T) {
	server := testutils.CreateTestGRPCServer(t)
	importer := NewHeavymockImporter()
	RegisterHeavymockImporterServer(server.Server, importer)
	server.Serve(t)
	defer server.Server.Stop()

	cfg := connection.GetClientConfiguration(server.Address)
	importerConn, err := connection.NewGRPCClientConnection(context.Background(), cfg)
	require.NoError(t, err)

	defer importerConn.GetGRPCConn().Close()

	client := NewHeavymockImporterClient(importerConn.GetGRPCConn())

	stream, err := client.Import(context.Background())
	require.NoError(t, err)

	records := testutils.GenerateRecordsSilence(5)
	var expectedRecords []*exporter.Record
	for _, record := range records {
		expectedRecords = append(expectedRecords, record)
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
	unsentRecords := importer.GetUnsentRecords()
	require.Len(t, unsentRecords, len(records))
	var c int
	for _, u := range unsentRecords {
		for _, e := range expectedRecords {
			if e.Equal(u) {
				c++
				break
			}
		}
	}
	require.Equal(t, len(expectedRecords), c)
}
