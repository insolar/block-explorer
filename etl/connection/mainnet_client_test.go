// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package connection

import (
	"testing"

	"github.com/insolar/block-explorer/etl"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/connectivity"
)

func TestNewClient_readyToConnect(t *testing.T) {
	config := testConfig()
	client, err := NewMainNetClient(config)
	require.NoError(t, err)
	require.Equal(t, connectivity.Idle.String(), client.GetGRPCConn().GetState().String(), "MainnetClient does not ready to connect")
}

func testConfig() etl.GRPCConfig {
	return etl.GRPCConfig{
		Addr:            "127.0.0.1:5678",
		MaxTransportMsg: 1073741824,
	}
}
