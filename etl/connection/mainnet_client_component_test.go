// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package connection

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/insolar/insolar/record"
	pb "github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
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
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "failed to listen")
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	pb.RegisterRecordExporterServer(grpcServer, &recExpServer{})

	// need to run grpcServer.Serve in different goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			require.Error(t, err, "server exited with error")
			return
		}
	}()

	// prepare config with listening address
	cfg := configuration.Replicator{
		Addr:            listener.Addr().String(),
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
		require.Equal(t, expectedRecord, record, "Incorrect response message")
	}
}
