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
	err = extractor.Start(ctx)
	require.NoError(t, err)
	defer extractor.Stop(ctx)
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

func TestLoadJetDrops_returnsRecordByPulses(t *testing.T) {
	ctx := context.Background()
	batchSize := 1
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	f := testutils.GenerateRecords(batchSize)
	expectedRecord, err := f()
	require.NoError(t, err)
	startPulseNumber := int(expectedRecord.Record.ID.Pulse().AsUint32())
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
	err = extractor.LoadJetDrops(ctx, startPulseNumber, startPulseNumber+10)
	require.NoError(t, err)
	// we are waiting only 2 times, because of 2 different pulses
	for i := 0; i < 2; {
		select {
		case jd := <-extractor.mainJetDropsChan:
			require.NotNil(t, jd)
			// two in each pulses from generator
			require.Len(t, jd.Records, 2, "no records received")
			i++
		case <-time.After(time.Millisecond * 100):
			t.Fatal("chan receive timeout ")
		}
	}
}

func TestLoadJetDrops_fromPulseNumberCannotBeNegative(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	extractor := NewMainNetExtractor(1, recordClient)
	err := extractor.LoadJetDrops(ctx, -1, 10)
	require.EqualError(t, err, "fromPulseNumber cannot be negative")
}

func TestLoadJetDrops_toPulseNumberCannotBeLess1(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	extractor := NewMainNetExtractor(1, recordClient)
	err := extractor.LoadJetDrops(ctx, 1, 0)
	require.EqualError(t, err, "toPulseNumber cannot be less than 1")
}

func TestLoadJetDrops_toPulseNumberShouldBeGreater(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	extractor := NewMainNetExtractor(1, recordClient)
	err := extractor.LoadJetDrops(ctx, 10, 9)
	require.EqualError(t, err, "fromPulseNumber cannot be greater than toPulseNumber")
}

type recordStream struct {
	grpc.ClientStream
	recv func() (*exporter.Record, error)
}

func (s recordStream) Recv() (*exporter.Record, error) {
	return s.recv()
}
