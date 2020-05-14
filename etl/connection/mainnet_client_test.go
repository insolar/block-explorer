// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package connection

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/insolar/block-explorer/etl"
	"github.com/insolar/block-explorer/etl/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func TestNewClient_readyToConnect(t *testing.T) {
	config := testConfig()
	client, err := NewMainNetClient(config)
	require.NoError(t, err)
	require.Equal(t, connectivity.Idle.String(), client.GetGRPCConn().GetState().String(), "client does not ready to connect")
}

func TestNewClient_mockConnection(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockClient := mock.NewMockClient(mc)
	mockClient.
		EXPECT().
		GetGRPCConn().
		AnyTimes().
		Return(&grpc.ClientConn{})

	clientConn := mockClient.GetGRPCConn()
	require.NotNil(t, clientConn)
}

func testConfig() etl.GRPCConfig {
	return etl.GRPCConfig{
		Addr:            "127.0.0.1:5678",
		MaxTransportMsg: 1073741824,
	}
}
