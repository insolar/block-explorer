// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package connection

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/stretchr/testify/require"
)

func TestNewClient_readyToConnect(t *testing.T) {
	config := testConfig()
	client, err := NewGRPCClientConnection(belogger.TestContext(t), config)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()
	require.NotNil(t, client.GetGRPCConn())
}

func testConfig() configuration.Replicator {
	return configuration.Replicator{
		Addr:            "127.0.0.1:0",
		MaxTransportMsg: 1073741824,
	}
}

func TestConnection_GRPC_auth_with_tls(t *testing.T) {
	cfg := testConfig()
	cfg.Auth.Required = true
	cfg.Auth.InsecureTLS = false
	client, err := NewGRPCClientConnection(belogger.TestContext(t), cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()
	require.NotNil(t, client.GetGRPCConn())
}

func TestConnection_GRPC_auth_insecure(t *testing.T) {
	cfg := testConfig()
	cfg.Auth.Required = true
	cfg.Auth.InsecureTLS = true
	client, err := NewGRPCClientConnection(belogger.TestContext(t), cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()
	require.NotNil(t, client.GetGRPCConn())
}

func TestConnection_GRPC_no_auth(t *testing.T) {
	cfg := testConfig()
	cfg.Auth.Required = false
	client, err := NewGRPCClientConnection(belogger.TestContext(t), cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()
	require.NotNil(t, client.GetGRPCConn())
}

func TestConnection_NotifyShutdown(t *testing.T) {
	ctx := belogger.TestContext(t)
	stopChannel := make(chan struct{})
	isSend := false

	client := mock.NewClientMock(t)
	client.NotifyShutdownMock.Set(func(ctx context.Context, stopChannel chan<- struct{}, waitForStateChange time.Duration) {
		isSend = true
		go func() { stopChannel <- struct{}{} }()
	})
	client.NotifyShutdown(ctx, stopChannel, time.Millisecond)

	select {
	case <-stopChannel:
		// error happened
		require.True(t, isSend)
	case <-time.After(time.Millisecond * 100):
		t.Fatal("chan receive timeout. Stop signal was not received")
	}
}
