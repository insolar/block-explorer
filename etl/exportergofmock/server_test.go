// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package exportergofmock

import (
	"context"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/exporter"
	"github.com/insolar/block-explorer/etl/models"
)

func TestNewMockBEAPIServerGetPulses(t *testing.T) {
	var initPulseNum int64 = 400000
	e := NewExporterMock(initPulseNum)
	defer e.Stop()

	// 10 records of proto1/obj1 in first pulse
	e.NewPulse(true, true)
	proto1 := e.NewCurrentPulseRef()
	obj1 := e.NewCurrentPulseRef()
	objPayload1 := insolar.MustSerialize(struct{}{})
	e.NewCurrentPulseRecords(RecordsTemplate{
		Records:      10,
		PrototypeRef: proto1,
		ObjectRef:    obj1,
		Payload:      objPayload1,
		RecordType:   models.StateRecord,
	})

	// 10 records of proto2/obj2 in second pulse
	e.NewPulse(true, true)
	proto2 := e.NewCurrentPulseRef()
	obj2 := e.NewCurrentPulseRef()
	objPayload2 := insolar.MustSerialize(struct{}{})
	e.NewCurrentPulseRecords(RecordsTemplate{
		Records:      10,
		PrototypeRef: proto2,
		ObjectRef:    obj2,
		Payload:      objPayload2,
		RecordType:   models.StateRecord,
	})

	// proto3/obj3 span over pulses 3 and 4
	e.NewPulse(true, true)
	proto3 := e.NewCurrentPulseRef()
	obj3 := e.NewCurrentPulseRef()
	objPayload3 := insolar.MustSerialize(struct{}{})
	e.NewCurrentPulseRecords(RecordsTemplate{
		Records:      10,
		PrototypeRef: proto3,
		ObjectRef:    obj3,
		Payload:      objPayload3,
		RecordType:   models.StateRecord,
	})
	e.NewPulse(true, true)
	e.NewCurrentPulseRecords(RecordsTemplate{
		Records:      10,
		PrototypeRef: proto3,
		ObjectRef:    obj3,
		Payload:      objPayload3,
		RecordType:   models.StateRecord,
	})

	c := NewClient(e.Listen)

	t.Run("it gets one pulse which has records of proto1", func(t *testing.T) {
		stream, err := c.GetNextPulse(context.Background(), &exporter.GetNextPulseRequest{
			PulseNumberFrom: initPulseNum + 1,
			Prototypes:      [][]byte{proto1},
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		pulses := c.ReadAllPulses(stream)
		require.Equal(t, 4, len(pulses))
		require.Equal(t, int64(10), pulses[0].RecordAmount)
		require.Equal(t, int64(0), pulses[1].RecordAmount)
		require.Equal(t, int64(0), pulses[2].RecordAmount)
		require.Equal(t, int64(0), pulses[3].RecordAmount)
	})

	t.Run("no pulse is found if PulseNumberFrom > pulse stored in mock", func(t *testing.T) {
		stream, err := c.GetNextPulse(context.Background(), &exporter.GetNextPulseRequest{
			PulseNumberFrom: initPulseNum + 2,
			Prototypes:      [][]byte{proto1},
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		pulses := c.ReadAllPulses(stream)
		require.Equal(t, 3, len(pulses))
		require.Equal(t, int64(0), pulses[0].RecordAmount)
		require.Equal(t, int64(0), pulses[1].RecordAmount)
		require.Equal(t, int64(0), pulses[2].RecordAmount)
	})

	t.Run("records for proto3 found in pulses 3 and 4", func(t *testing.T) {
		stream, err := c.GetNextPulse(context.Background(), &exporter.GetNextPulseRequest{
			PulseNumberFrom: initPulseNum,
			Prototypes:      [][]byte{proto3},
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		pulses := c.ReadAllPulses(stream)
		require.Equal(t, 4, len(pulses))
		require.Equal(t, int64(0), pulses[0].RecordAmount)
		require.Equal(t, int64(0), pulses[1].RecordAmount)
		require.Equal(t, int64(10), pulses[2].RecordAmount)
		require.Equal(t, int64(10), pulses[3].RecordAmount)
	})

	t.Run("it gets all records from pulse 1 in one batch", func(t *testing.T) {
		stream, err := c.GetRecords(context.Background(), &exporter.GetRecordsRequest{
			Polymorph:    0,
			PulseNumber:  initPulseNum + 1,
			Prototypes:   [][]byte{proto1},
			RecordNumber: 0,
			Count:        10,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		recs := c.ReadAllRecords(stream)
		require.Equal(t, 10, len(recs))
	})

	t.Run("it gets all records from pulse 1 in two batches", func(t *testing.T) {
		{
			stream, err := c.GetRecords(context.Background(), &exporter.GetRecordsRequest{
				Polymorph:    0,
				PulseNumber:  initPulseNum + 1,
				Prototypes:   [][]byte{proto1},
				RecordNumber: 0,
				Count:        5,
			}, grpc.WaitForReady(true))
			require.NoError(t, err)
			recs := c.ReadAllRecords(stream)
			require.Equal(t, 5, len(recs))
		}
		{
			stream, err := c.GetRecords(context.Background(), &exporter.GetRecordsRequest{
				Polymorph:    0,
				PulseNumber:  initPulseNum + 1,
				Prototypes:   [][]byte{proto1},
				RecordNumber: 5,
				Count:        5,
			}, grpc.WaitForReady(true))
			require.NoError(t, err)
			recs := c.ReadAllRecords(stream)
			require.Equal(t, 5, len(recs))
		}
	})

	t.Run("it gets all remaining records if count > records we have", func(t *testing.T) {
		stream, err := c.GetRecords(context.Background(), &exporter.GetRecordsRequest{
			Polymorph:    0,
			PulseNumber:  initPulseNum + 1,
			Prototypes:   [][]byte{proto1},
			RecordNumber: 0,
			Count:        20,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		recs := c.ReadAllRecords(stream)
		require.Equal(t, 10, len(recs))
	})

	t.Run("it gets no records if RecordNumber > record order we have", func(t *testing.T) {
		stream, err := c.GetRecords(context.Background(), &exporter.GetRecordsRequest{
			Polymorph:    0,
			PulseNumber:  initPulseNum + 1,
			Prototypes:   [][]byte{proto1},
			RecordNumber: 100,
			Count:        10,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		recs := c.ReadAllRecords(stream)
		require.Equal(t, 0, len(recs))
	})

	t.Run("no records found for not existing pulse", func(t *testing.T) {
		stream, err := c.GetRecords(context.Background(), &exporter.GetRecordsRequest{
			Polymorph:    0,
			PulseNumber:  initPulseNum + 100,
			Prototypes:   [][]byte{proto1},
			RecordNumber: 0,
			Count:        10,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		recs := c.ReadAllRecords(stream)
		require.Equal(t, 0, len(recs))
	})
}
