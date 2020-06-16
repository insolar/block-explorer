// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package clients

import (
	"context"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"google.golang.org/grpc"
)

type TestPulseClient struct {
	export       func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error)
	topSyncPulse func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error)
}

func (c *TestPulseClient) Export(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
	return c.export(ctx, in, opts...)
}

func (c *TestPulseClient) TopSyncPulse(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error) {
	return c.topSyncPulse(ctx, in, opts...)
}

func getTestTopSyncPulseResponse(pn uint32) *exporter.TopSyncPulseResponse {
	return &exporter.TopSyncPulseResponse{
		Polymorph:   0,
		PulseNumber: pn,
	}
}

func GetTestPulseClient(pn uint32, err error) *TestPulseClient {
	client := &TestPulseClient{}
	client.topSyncPulse = func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (response *exporter.TopSyncPulseResponse, e error) {
		return getTestTopSyncPulseResponse(pn), err
	}
	return client
}
