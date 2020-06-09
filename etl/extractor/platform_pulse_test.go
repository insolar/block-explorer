// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package extractor

import (
	"context"
	"errors"
	"testing"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestPulse_getCurrentPulse_Success(t *testing.T) {
	ctx := context.Background()
	expectedPulse := uint32(0)
	client := &pulseClient{}
	client.topSyncPulse = func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (response *exporter.TopSyncPulseResponse, e error) {
		return &exporter.TopSyncPulseResponse{
			Polymorph:   0,
			PulseNumber: expectedPulse,
		}, nil
	}

	pe := NewPlatformPulseExtractor(client)
	currentPulse, err := pe.GetCurrentPulse(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedPulse, currentPulse)
}

func TestPulse_getCurrentPulse_Fail(t *testing.T) {
	ctx := context.Background()
	client := &pulseClient{}
	client.topSyncPulse = func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (response *exporter.TopSyncPulseResponse, e error) {
		return &exporter.TopSyncPulseResponse{
			Polymorph:   0,
			PulseNumber: 1,
		}, errors.New("test error")
	}

	pe := NewPlatformPulseExtractor(client)
	_, err := pe.GetCurrentPulse(ctx)
	require.Error(t, err)
}

type pulseClient struct {
	export       func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error)
	topSyncPulse func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error)
}

func (c *pulseClient) Export(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
	return c.export(ctx, in, opts...)
}

func (c *pulseClient) TopSyncPulse(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error) {
	return c.topSyncPulse(ctx, in, opts...)
}
