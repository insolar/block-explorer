// +build unit

package exporter

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type pulseExporterTestServer struct {
	sender func(*GetNextPulseResponse) error
}

func (p *pulseExporterTestServer) Send(response *GetNextPulseResponse) error {
	return p.sender(response)
}

func (p *pulseExporterTestServer) SetHeader(md metadata.MD) error {
	panic("implement me")
}

func (p *pulseExporterTestServer) SendHeader(md metadata.MD) error {
	panic("implement me")
}

func (p *pulseExporterTestServer) SetTrailer(md metadata.MD) {
	panic("implement me")
}

func (p *pulseExporterTestServer) Context() context.Context {
	return context.TODO()
}

func (p *pulseExporterTestServer) SendMsg(m interface{}) error {
	panic("implement me")
}

func (p *pulseExporterTestServer) RecvMsg(m interface{}) error {
	panic("implement me")
}

func TestExporter_Pulse_Export_Fail(t *testing.T) {
	sm := mock.NewStorageMock(t)
	sm.GetNextCompletePulseFilterByPrototypeReferenceMock.Return(models.Pulse{PulseNumber: 1}, nil)
	pulseServer := NewPulseServer(sm, time.Nanosecond, nil)

	EOF := io.EOF
	iterations := 0
	sender := func(*GetNextPulseResponse) error {
		iterations++
		// exit point from sender
		return EOF
	}
	err := pulseServer.GetNextPulse(&GetNextPulseRequest{0, nil}, &pulseExporterTestServer{sender})

	require.Equal(t, 1, iterations, "iterations must be called one times because of error")
	require.Equal(t, err, EOF)
}

func TestExporter_Pulse_Export_Success(t *testing.T) {
	sm := mock.NewStorageMock(t)
	pulseForSend := models.Pulse{PulseNumber: 1}
	sm.GetNextCompletePulseFilterByPrototypeReferenceMock.
		Set(func(prevPulse int64, prototypes [][]byte) (models.Pulse, error) {
			return pulseForSend, nil
		})
	pulseServer := NewPulseServer(sm, time.Nanosecond, nil)

	EOF := io.EOF
	iterations := 0
	totalIterations := 5
	sender := func(*GetNextPulseResponse) error {
		pulseForSend.PulseNumber += 1
		iterations++
		if iterations >= totalIterations {
			// exit point from sender
			return EOF
		}
		return nil
	}
	err := pulseServer.GetNextPulse(&GetNextPulseRequest{0, nil}, &pulseExporterTestServer{sender})

	require.Equal(t, totalIterations, iterations, "sender must have been called defined times")
	require.Equal(t, totalIterations, int(sm.GetNextCompletePulseFilterByPrototypeReferenceBeforeCounter()),
		"storage mustn't be called more than totalIterations")
	require.Equal(t, err, EOF, "error should be io.EOF")
}

func TestExporter_Pulse_Export_All_Situations(t *testing.T) {
	sm := mock.NewStorageMock(t)
	pulseForSend := models.Pulse{PulseNumber: 0}
	iterations := 0
	sm.GetNextCompletePulseFilterByPrototypeReferenceMock.
		Set(func(prevPulse int64, prototypes [][]byte) (models.Pulse, error) {
			// if it's the first calling time
			if sm.GetNextCompletePulseFilterByPrototypeReferenceBeforeCounter() == 1 {
				// simulate the error from storage
				return models.Pulse{}, errors.New("some error")
			}

			if sm.GetNextCompletePulseFilterByPrototypeReferenceBeforeCounter() == 2 {
				return pulseForSend, nil
			}
			pulseForSend.PulseNumber += 1
			return pulseForSend, nil
		})
	pulseServer := NewPulseServer(sm, time.Nanosecond, nil)

	EOF := io.EOF

	totalIterations := 5
	sender := func(*GetNextPulseResponse) error {
		iterations++
		if iterations >= totalIterations {
			// exit point from sender
			return EOF
		}
		return nil
	}
	err := pulseServer.GetNextPulse(&GetNextPulseRequest{0, nil}, &pulseExporterTestServer{sender})

	require.Equal(t, totalIterations, iterations, "sender must have been called defined times")
	// we are calling GetNextCompletePulseFilterByPrototypeReference 2 times more
	// because of if statements
	require.Equal(t, totalIterations+2, int(sm.GetNextCompletePulseFilterByPrototypeReferenceBeforeCounter()),
		"storage must be called at 2 times more than totalIterations")
	require.Equal(t, err, EOF, "error should be io.EOF")
}
