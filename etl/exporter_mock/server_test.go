package exporter_mock

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
	e := NewBEExporter()
	err := e.Start()
	defer e.Stop()
	require.NoError(t, err)

	var initPulseNum int64 = 400000

	e.SetInitPulseNumber(initPulseNum)

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
		RecordType:   models.State,
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
		RecordType:   models.State,
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
		RecordType:   models.State,
	})
	e.NewPulse(true, true)
	e.NewCurrentPulseRecords(RecordsTemplate{
		Records:      10,
		PrototypeRef: proto3,
		ObjectRef:    obj3,
		Payload:      objPayload3,
		RecordType:   models.State,
	})

	c := NewClient(e.Listen)

	t.Run("it gets one pulse which has records of proto1", func(t *testing.T) {
		protos := [][]byte{proto1}
		stream, err := c.GetNextPulse(context.Background(), &exporter.GetNextPulseRequest{
			PulseNumberFrom: uint32(initPulseNum + 1),
			Prototypes:      protos,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		pulses := c.ReadAllPulses(stream)
		require.Equal(t, 1, len(pulses))
	})

	t.Run("no pulse is found if PulseNumberFrom > pulse stored in mock", func(t *testing.T) {
		protos := [][]byte{proto1}
		stream, err := c.GetNextPulse(context.Background(), &exporter.GetNextPulseRequest{
			PulseNumberFrom: uint32(initPulseNum + 2),
			Prototypes:      protos,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		pulses := c.ReadAllPulses(stream)
		require.Equal(t, 0, len(pulses))
	})

	t.Run("records for proto3 found in pulses 3 and 4", func(t *testing.T) {
		protos := [][]byte{proto3}
		stream, err := c.GetNextPulse(context.Background(), &exporter.GetNextPulseRequest{
			PulseNumberFrom: uint32(initPulseNum),
			Prototypes:      protos,
		}, grpc.WaitForReady(true))
		require.NoError(t, err)
		pulses := c.ReadAllPulses(stream)
		require.Equal(t, 2, len(pulses))
	})
}
