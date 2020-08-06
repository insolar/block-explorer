// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var testHeavyGRPCVersion = exporter.AllowedOnHeavyVersion

type TestGRPCServerConfig struct {
	VersionChecker bool
	HeavyVersion   *int
}

type TestGRPCServer struct {
	Listener net.Listener
	Server   *grpc.Server
	Network  string
	Address  string
}

func CreateTestGRPCServer(t testing.TB, config *TestGRPCServerConfig) *TestGRPCServer {
	if config == nil {
		config = &TestGRPCServerConfig{}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")
	var grpcServer *grpc.Server
	if config.VersionChecker {
		if config.HeavyVersion != nil {
			testHeavyGRPCVersion = *config.HeavyVersion
		}
		grpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(versionCheckUnary)),
			grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(versionCheckStream)),
		)
	} else {
		grpcServer = grpc.NewServer()
	}

	return &TestGRPCServer{
		Listener: listener,
		Server:   grpcServer,
		Network:  listener.Addr().Network(),
		Address:  listener.Addr().String(),
	}
}

// Serve starts to read gRPC requests
func (s *TestGRPCServer) Serve(t testing.TB) {
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
	}()
}

func versionCheckUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve metadata")
	}
	err := validateClientVersion(md)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func versionCheckStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, "failed to retrieve metadata")
	}
	err := validateClientVersion(md)
	if err != nil {
		return err
	}
	return handler(srv, stream)
}

func validateClientVersion(metaDataFromRequest metadata.MD) error {
	typeClient, ok := metaDataFromRequest[exporter.KeyClientType]
	if !ok || len(typeClient) == 0 || typeClient[0] == exporter.Unknown.String() {
		return status.Error(codes.InvalidArgument, "unknown type client")
	}

	switch typeClient[0] {
	case exporter.ValidateHeavyVersion.String():
	case exporter.ValidateContractVersion.String():
		return status.Error(codes.InvalidArgument, "block explorer should send client type 1")
	default:
		return status.Error(codes.InvalidArgument, "unknown type client")
	}
	// validate protocol version from client
	err := compareAllowedVersion(exporter.KeyClientVersionHeavy, int64(testHeavyGRPCVersion), metaDataFromRequest)
	if err != nil {
		return err
	}
	return nil
}

func compareAllowedVersion(nameVersion string, allowedVersion int64, metaDataFromRequest metadata.MD) error {
	versionClientMD, ok := metaDataFromRequest[nameVersion]
	if !ok || len(versionClientMD) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("unknown %s", nameVersion))
	}
	versionClient, err := strconv.ParseInt(versionClientMD[0], 10, 64)
	if err != nil || versionClient < 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("incorrect format of the %s", nameVersion))
	}
	if versionClient == 0 || versionClient < allowedVersion {
		return exporter.ErrDeprecatedClientVersion
	}
	return nil
}
