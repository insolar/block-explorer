// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"testing"
	"time"

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
	require.NoError(t, err)
	withDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(batchSize, expectedRecord)

	stream := recordStream{
		recv: withDifferencePulses,
	}
	recordClient.ExportMock.Set(
		func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (
			r1 exporter.RecordExporter_ExportClient, err error) {
			return stream, nil
		})

	extractor := NewMainNetExtractor(uint32(batchSize), recordClient)
	jetDrops := extractor.GetJetDrops(ctx)

	for i := 0; i < 2; i++ {
		select {
		case jd := <-jetDrops:
			if i < 1 {
				// when i ∈ [0,1) we received records with some pulse
				// when i ≥ 2 we received records with different pulse, now records from i ∈ [0,1) should be returned
				continue
			}
			require.NotNil(t, jd)
			require.Len(t, jd.Records, 1, "no records received")
			require.True(t, expectedRecord.Equal(jd.Records[0]), "jetDrops are not equal")
		case <-time.After(time.Second * 1):
			t.Fatal("chan receive timeout ")
		}
	}
}

type recordStream struct {
	grpc.ClientStream
	recv func() (*exporter.Record, error)
}

func (s recordStream) Recv() (*exporter.Record, error) {
	return s.recv()
}
