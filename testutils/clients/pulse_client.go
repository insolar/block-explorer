// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package clients

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"google.golang.org/grpc"
)

type TestPulseClient struct {
	export             func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error)
	topSyncPulse       func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error)
	nextFinalizedPulse func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error)
}

func (c *TestPulseClient) NextFinalizedPulse(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
	return c.nextFinalizedPulse(ctx, in, opts...)
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
	client.nextFinalizedPulse = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		return getFullPulse(), nil
	}
	return client
}

func getFullPulse() *exporter.FullPulse {
	pulseNumber := gen.PulseNumber()
	res := &exporter.FullPulse{
		PulseNumber:      pulseNumber,
		PrevPulseNumber:  pulseNumber,
		NextPulseNumber:  pulseNumber,
		Entropy:          insolar.Entropy{},
		PulseTimestamp:   0,
		EpochPulseNumber: 0,
		Jets:             nil,
	}
	return res
}
