// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package extractor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/clients"
)

func TestGetJetDrops(t *testing.T) {
	ctx := context.Background()
	pulseCount := 1
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	withDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(pulseCount, 2, StartPulseNumber)
	expectedRecord, err := withDifferencePulses()
	require.NoError(t, err) // you are testing yours testutils

	stream := recordStream{
		recvFunc: withDifferencePulses,
	}
	recordClient.ExportMock.Set(
		func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (
			r1 exporter.RecordExporter_ExportClient, err error) {
			mtd, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok)
			require.Equal(t, exporter.ValidateHeavyVersion.String(), mtd.Get(exporter.KeyClientType)[0])
			require.Equal(t, PlatformAPIVersion, mtd.Get(exporter.KeyClientVersionHeavy)[0])

			return stream, nil
		})

	pulseClient := clients.GetTestPulseClient(65537, nil)
	pulseExtractor := NewPlatformPulseExtractor(pulseClient)
	extractor := NewPlatformExtractor(uint32(pulseCount), 0, 100, pulseExtractor, recordClient, func() {})
	err = extractor.Start(ctx)
	require.NoError(t, err)
	defer extractor.Stop(ctx)
	jetDrops := extractor.GetJetDrops(ctx)

	for i := 0; i < pulseCount; i++ {
		select {
		case jd := <-jetDrops:
			if i < 1 {
				// when i ∈ [0,1) we received records with some pulse
				// when i ≥ 2 we received records with different pulse, now records from i ∈ [0,1) should be returned
				continue
			}
			require.NotNil(t, jd)
			require.Len(t, jd.Records, 1, "no records received")
			require.Equal(t,
				expectedRecord.Record.ID.Pulse().String(),
				jd.Records[0].Record.ID.Pulse().String(),
				"jetDrops are not equal")
			require.NotEqual(t,
				expectedRecord.Record.ID.String(),
				jd.Records[0].Record.ID.String(),
				"record reference should be different")
		case <-time.After(time.Second * 10):
			t.Fatal("chan receive timeout ")
		}
	}
}

func TestGetJetDrops_WrongVersionOnPulseError(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	var called int32 = 0
	shutdownBETestFunc := func() {
		atomic.AddInt32(&called, 1)
	}
	pulseExtractor := mock.NewPulseExtractorMock(mc)
	pulseExtractor.GetNextFinalizedPulseMock.Set(func(ctx context.Context, p int64) (fp1 *exporter.FullPulse, err error) {
		return nil, errors.New("unknown heavy-version")
	})
	extractor := NewPlatformExtractor(uint32(1), 0, 100, pulseExtractor, recordClient, shutdownBETestFunc)
	err := extractor.Start(ctx)
	defer extractor.Stop(ctx)
	require.NoError(mc, err)

	jetDrops := extractor.GetJetDrops(ctx)

	select {
	case <-jetDrops:
		require.True(t, false, "received something")
	case <-time.After(time.Second * 1):
		t.Log("received nothing. ok")
	}
	require.Equal(t, atomic.LoadInt32(&called), int32(1), "shutdownBETestFunc should be invoked")
}

func TestGetJetDrops_WrongVersionOnRecordError(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	recordClient.ExportMock.Set(
		func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (
			r1 exporter.RecordExporter_ExportClient, err error) {

			mtd, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok)
			require.Equal(t, exporter.ValidateHeavyVersion.String(), mtd.Get(exporter.KeyClientType)[0])
			require.Equal(t, PlatformAPIVersion, mtd.Get(exporter.KeyClientVersionHeavy)[0])
			return recordStream{}, exporter.ErrDeprecatedClientVersion
		})
	var called int32 = 0
	shutdownBETestFunc := func() {
		atomic.AddInt32(&called, 1)
	}
	pulseClient := clients.GetTestPulseClient(65537, nil)
	pulseExtractor := NewPlatformPulseExtractor(pulseClient)
	extractor := NewPlatformExtractor(uint32(1), 0, 100, pulseExtractor, recordClient, shutdownBETestFunc)
	err := extractor.Start(ctx)
	defer extractor.Stop(ctx)
	require.NoError(mc, err)

	jetDrops := extractor.GetJetDrops(ctx)

	select {
	case <-jetDrops:
		require.True(t, false, "received something")
	case <-time.After(time.Second * 1):
		t.Log("received nothing. ok")
	}
	require.Equal(t, atomic.LoadInt32(&called), int32(1), "shutdownBETestFunc should be invoked")
}

func recordTapeFunc(t *testing.T, tape []*exporter.Record) func() (record *exporter.Record, e error) {
	used := 0
	return func() (record *exporter.Record, e error) {
		if len(tape) == used {
			return nil, io.EOF
		}
		ret := tape[used]
		used++
		return ret, nil
	}
}

func TestLoadJetDrops_returnsRecordByPulses(t *testing.T) {
	tests := []struct {
		differentPulseCount int
		recordCount         int
	}{
		{
			differentPulseCount: 1,
			recordCount:         1,
		}, {
			differentPulseCount: 1,
			recordCount:         2,
		}, {
			differentPulseCount: 2,
			recordCount:         1,
		}, {
			differentPulseCount: 2,
			recordCount:         2,
		},
	}

	ctx := context.Background()
	mc := minimock.NewController(t)
	for _, test := range tests {
		t.Run(fmt.Sprintf("pulse-count=%d,record-count=%d", test.differentPulseCount, test.recordCount), func(t *testing.T) {
			recordClient := mock.NewRecordExporterClientMock(mc)

			recordTape := make(map[int][]*exporter.Record)
			startPulseNumber := 65537
			for p := 0; p < test.differentPulseCount; p++ {
				pulse := startPulseNumber + p*10
				for r := 0; r < test.recordCount; r++ {
					recordTape[pulse] = append(recordTape[pulse], &exporter.Record{
						Record: record.Material{ID: *insolar.NewID(insolar.PulseNumber(pulse), nil)},
					})
				}
			}
			lastPulse := startPulseNumber + 10*test.differentPulseCount
			lastRecord := &exporter.Record{
				Record: record.Material{ID: *insolar.NewID(insolar.PulseNumber(lastPulse), nil)},
			}
			recordTape[lastPulse] = append(recordTape[lastPulse], lastRecord)

			recordClient.ExportMock.Set(
				func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (
					r1 exporter.RecordExporter_ExportClient, err error) {
					pu := int(in.PulseNumber)
					slice := in.RecordNumber
					if int(slice) > len(recordTape[pu]) {
						return recordStream{
							recvFunc: recordTapeFunc(t, recordTape[lastPulse]),
						}, nil
					}
					return recordStream{
						recvFunc: recordTapeFunc(t, append(recordTape[pu][slice:], lastRecord)),
					}, nil
				})

			pulseIteration := 0
			pulseExtractor := mock.NewPulseExtractorMock(t)
			pulseExtractor.GetNextFinalizedPulseMock.Set(
				func(ctx context.Context, p int64) (fp1 *exporter.FullPulse, err error) {
					pp, err := clients.GetFullPulse(uint32(startPulseNumber+10*pulseIteration), nil)
					pulseIteration++
					return pp, err
				})

			extractor := NewPlatformExtractor(77, 0, 100, pulseExtractor, recordClient, func() {})
			err := extractor.LoadJetDrops(ctx, int64(startPulseNumber-10), int64(startPulseNumber+10*(test.differentPulseCount-1)))
			require.NoError(t, err)
			for i := 0; i < test.differentPulseCount; i++ {
				select {
				case jd := <-extractor.GetJetDrops(ctx):
					require.NotNil(t, jd)
					require.Len(t, jd.Records, test.recordCount, "no records received")
				case <-time.After(time.Millisecond * 100):
					t.Fatal("chan receive timeout ")
				}
			}
		})
	}
}
