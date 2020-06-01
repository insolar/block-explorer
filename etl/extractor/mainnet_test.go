// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package extractor

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetJetDrops(t *testing.T) {
	ctx := context.Background()
	batchSize := 1
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	f := testutils.GenerateRecords(batchSize)
	expectedRecord, err := f()
	fn := func() (record *exporter.Record, e error) {
		return expectedRecord, err
	}

	stream := recordStream{
		recvFunc: fn,
	}
	recordClient.ExportMock.Set(func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
		return stream, nil
	})

	extractor := NewMainNetExtractor(uint32(batchSize), recordClient)
	jetDrops := extractor.GetJetDrops(ctx)

	select {
	case jd := <-jetDrops:
		require.NotNil(t, jd)
		require.Len(t, jd.Records, 1, "no records received")
		require.True(t, expectedRecord.Equal(jd.Records[0]), "jetDrops are not equal")
	}
}
