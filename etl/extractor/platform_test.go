// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package extractor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/clients"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetJetDrops(t *testing.T) {
	ctx := context.Background()
	pulseCount := 1
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	withDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(pulseCount, 2)
	expectedRecord, err := withDifferencePulses()
	require.NoError(t, err)

	stream := recordStream{
		recvFunc: withDifferencePulses,
	}
	recordClient.ExportMock.Set(
		func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (
			r1 exporter.RecordExporter_ExportClient, err error) {
			return stream, nil
		})

	pulseClient := clients.GetTestPulseClient(1, nil)
	pulseExtractor := NewPlatformPulseExtractor(pulseClient)
	extractor := NewPlatformExtractor(uint32(pulseCount), pulseExtractor, recordClient)
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
		case <-time.After(time.Millisecond * 100):
			t.Fatal("chan receive timeout ")
		}
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
	recordClient := mock.NewRecordExporterClientMock(mc)

	for _, test := range tests {
		t.Run(fmt.Sprintf("pulse-count=%d,record-count=%d", test.differentPulseCount, test.recordCount), func(t *testing.T) {
			withDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(test.differentPulseCount+1, test.recordCount)
			expectedRecord, err := withDifferencePulses()
			require.NoError(t, err)
			startPulseNumber := int(expectedRecord.Record.ID.Pulse().AsUint32())

			stream := recordStream{
				recvFunc: withDifferencePulses,
			}
			recordClient.ExportMock.Set(
				func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (
					r1 exporter.RecordExporter_ExportClient, err error) {
					return stream, nil
				})

			pulseClient := clients.GetTestPulseClient(1, nil)
			pulseExtractor := NewPlatformPulseExtractor(pulseClient)
			extractor := NewPlatformExtractor(uint32(test.recordCount), pulseExtractor, recordClient)
			err = extractor.LoadJetDrops(ctx, startPulseNumber, startPulseNumber+10*test.differentPulseCount)
			require.NoError(t, err)
			// we are waiting only 2 times, because of 2 different pulses
			for i := 0; i < test.differentPulseCount; {
				select {
				case jd := <-extractor.GetJetDrops(ctx):
					require.NotNil(t, jd)
					if i == 0 {
						// test.recordCount-1 because we have already received the  first record from fist pulse.
						// see expectedRecord, err := withDifferencePulses()
						require.Len(t, jd.Records, test.recordCount-1, "no records received")
					} else {
						// two in each pulses from generator
						require.Len(t, jd.Records, test.recordCount, "no records received")
					}
					i++
				case <-time.After(time.Millisecond * 100):
					t.Fatal("chan receive timeout ")
				}
			}
		})
	}

}

func TestLoadJetDrops_fromPulseNumberCannotBeNegative(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	extractor := NewPlatformExtractor(1, nil, recordClient)
	err := extractor.LoadJetDrops(ctx, -1, 10)
	require.EqualError(t, err, "fromPulseNumber cannot be negative")
}

func TestLoadJetDrops_toPulseNumberCannotBeLess1(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	extractor := NewPlatformExtractor(1, nil, recordClient)
	err := extractor.LoadJetDrops(ctx, 1, 0)
	require.EqualError(t, err, "toPulseNumber cannot be less than 1")
}

func TestLoadJetDrops_toPulseNumberShouldBeGreater(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	recordClient := mock.NewRecordExporterClientMock(mc)

	extractor := NewPlatformExtractor(1, nil, recordClient)
	err := extractor.LoadJetDrops(ctx, 10, 9)
	require.EqualError(t, err, "fromPulseNumber cannot be greater than toPulseNumber")
}
