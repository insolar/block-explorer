// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package connection

import (
	"context"
	"io"
	"testing"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar/record"
	pb "github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

type recExpServer struct{}

var expectedRecord = &pb.Record{
	Polymorph:         1,
	RecordNumber:      100,
	Record:            record.Material{},
	ShouldIterateFrom: nil,
}

func (r *recExpServer) Export(records *pb.GetRecords, stream pb.RecordExporter_ExportServer) error {
	if err := stream.Send(expectedRecord); err != nil {
		return err
	}
	return nil
}

func TestClient_GetGRPCConnIsWorking(t *testing.T) {
	server := testutils.CreateTestGRPCServer(t)
	pb.RegisterRecordExporterServer(server.Server, &recExpServer{})
	server.Serve(t)
	defer server.Server.Stop()

	// prepare config with listening address
	cfg := configuration.Replicator{
		Addr:            server.GetAddress(),
		MaxTransportMsg: 100500,
	}

	// initialization MainNet connection
	client, err := NewMainNetClient(context.Background(), cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()

	greeterClient := pb.NewRecordExporterClient(client.GetGRPCConn())
	// send record to stream
	request := &pb.GetRecords{}
	stream, err := greeterClient.Export(context.Background(), request)
	require.NoError(t, err, "Error when sending client request")

	for {
		t.Log("listening...")
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%v.Export(_) = _, %v", client, err)
		}
		require.NoError(t, err, "Err listening stream")
		require.True(t, expectedRecord.Equal(record), "Incorrect response message")
	}
}
