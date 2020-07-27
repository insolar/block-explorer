// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package clients

import (
	"context"
	"errors"

	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"

	"google.golang.org/grpc"
)

type TestPulseClient struct {
	export                 func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error)
	topSyncPulse           func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error)
	NextFinalizedPulseFunc func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error)
}

func (c *TestPulseClient) NextFinalizedPulse(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
	return c.NextFinalizedPulseFunc(ctx, in, opts...)
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
	client.NextFinalizedPulseFunc = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		return GetFullPulse(pn, nil)
	}
	return client
}

func GetFullPulse(pn uint32, jetDropContinue []exporter.JetDropContinue) (*exporter.FullPulse, error) {
	time, err := insolar.PulseNumber(pn).AsApproximateTime()
	if err != nil {
		return nil, err
	}
	res := &exporter.FullPulse{
		PulseNumber:      insolar.PulseNumber(pn),
		PrevPulseNumber:  insolar.PulseNumber(pn - 10),
		NextPulseNumber:  insolar.PulseNumber(pn + 10),
		Entropy:          insolar.Entropy{},
		PulseTimestamp:   time.Unix(),
		EpochPulseNumber: 0,
		Jets:             jetDropContinue,
	}
	return res, nil
}

func (c *TestPulseClient) SetNextFinalizedPulseFunc(importer *heavymock.ImporterServer) {
	c.NextFinalizedPulseFunc = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		pulse, jetDropContinue := importer.GetLowestUnsentPulse()
		p := uint32(pulse)
		if p == 1<<32-1 {
			return nil, errors.New("unready yet")
		}
		return GetFullPulse(p, jetDropContinue)
	}
}
