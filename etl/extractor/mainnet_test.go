// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetJetDrops(t *testing.T) {
	ctx := context.Background()
	batchSize := 1
	mc := minimock.NewController(t)
	recordClient := NewRecordExporterClientMock(mc)

	f := testutils.GenerateRecords(batchSize)
	expectedRecord, err := f()
	fn := func() (record *exporter.Record, e error) {
		return expectedRecord, err
	}

	stream := recordStream{
		recv: fn,
	}
	recordClient.funcExport = func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
		return stream, nil
	}

	extractor := NewMainNetExtractor(uint32(batchSize), recordClient)
	jetDrops, errors := extractor.GetJetDrops(ctx)

	select {
	case err := <-errors:
		require.NoError(t, err)
	case jd := <-jetDrops:
		require.NotNil(t, jd)
		require.Len(t, jd.Records, 1, "no records received")
		require.True(t, expectedRecord.Equal(jd.Records[0]), "jetDrops are not equal")
	}
}

type recordStream struct {
	grpc.ClientStream
	recv func() (*exporter.Record, error)
}

func (s recordStream) Recv() (*exporter.Record, error) {
	return s.recv()
}
