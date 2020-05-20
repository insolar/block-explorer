// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type TestGRPCServer struct {
	Listener net.Listener
	Server   *grpc.Server
	Network  string
	Address  string
}

func CreateTestGRPCServer(t *testing.T) *TestGRPCServer {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "failed to listen")
	grpcServer := grpc.NewServer()

	return &TestGRPCServer{
		Listener: listener,
		Server:   grpcServer,
		Network:  listener.Addr().Network(),
		Address:  listener.Addr().String(),
	}
}

// Serve starts to read gRPC requests
func (s *TestGRPCServer) Serve(t *testing.T) {
	// need to run grpcServer.Serve in different goroutine
	go func() {
		var err error
		// for needs to fix the flaky behavior of grpcServer.Serve
		for i := 0; i < 100; i++ {
			if err = s.Server.Serve(s.Listener); err != nil {
				time.Sleep(time.Millisecond * 100)
				continue
			}
		}
		require.Error(t, err, "server exited with error")
		return
	}()
}
