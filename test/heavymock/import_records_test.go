// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package heavymock

import (
	"context"
	"io"
	"testing"

	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestHeavymockImporter_import(t *testing.T) {
	server := testutils.CreateTestGRPCServer(t)
	RegisterHeavymockImporterServer(server.Server, NewHeavymockImporter())
	server.Serve(t)
	defer server.Server.Stop()

	importerConn, err := NewImporterClient(server.GetAddress())
	require.NoError(t, err)

	defer importerConn.GetGRPCConn().Close()

	client := NewHeavymockImporterClient(importerConn.GetGRPCConn())

	stream, err := client.Import(context.Background())
	require.NoError(t, err)

	var expectedRecords []exporter.Record
	for i := 0; i < 5; i++ {
		expectedRecords = append(expectedRecords, *SimpleRecord)
	}
	for _, record := range expectedRecords {
		if err := stream.Send(&record); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal("Error sending to stream", err)
		}
	}

	reply, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.True(t, reply.Ok)
}
