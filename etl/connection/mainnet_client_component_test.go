// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package connection

import (
	"context"
	"net"
	"testing"

	"github.com/insolar/block-explorer/etl"
	pb "github.com/insolar/block-explorer/etl/connection/testdata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type gserver struct{}

// SayHello implements of pb.GreeterServer
func (s *gserver) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func TestClient_GetGRPCConnIsWorking(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "failed to listen")
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	pb.RegisterGreeterServer(grpcServer, &gserver{})

	// need to run grpcServer.Serve in different goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			require.Error(t, err, "server exited with error")
			return
		}
	}()

	// prepare config with listening address
	cfg := etl.GRPCConfig{
		Addr:            listener.Addr().String(),
		MaxTransportMsg: 100500,
	}

	// initialization MainNet connection
	client, err := NewMainNetClient(cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()

	greeterClient := pb.NewGreeterClient(client.GetGRPCConn())
	resp, err := greeterClient.SayHello(context.Background(), &pb.HelloRequest{Name: "Insolar"})
	require.NoError(t, err, "SayHello failed")
	require.Equal(t, "Hello Insolar", resp.Message, "Incorrect response message")
}
