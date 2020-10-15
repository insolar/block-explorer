// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter

import (
	"time"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type PulseServer struct {
	repository  interfaces.Storage
	pulsePeriod time.Duration
}

func NewPulseServer(repo interfaces.Storage, pulsePeriod time.Duration) *PulseServer {
	return &PulseServer{repo, pulsePeriod}
}

func (s *PulseServer) GetNextPulse(req *GetNextPulseRequest, stream PulseExporter_GetNextPulseServer) error {
	ctx := stream.Context()
	logger := belogger.FromContext(ctx)

	currentPN := req.GetPulseNumberFrom()
	protos := req.GetPrototypes()

	for {
		receivedPulse, err := s.repository.GetNextCompletePulseFilterByPrototypeReference(currentPN, protos)
		// try again while error occurred
		if err != nil {
			continue
		}

		// if we have received current pulse_number we need to wait a bit
		if currentPN >= receivedPulse.PulseNumber {
			time.Sleep(s.pulsePeriod)
			continue
		}

		err = stream.Send(&GetNextPulseResponse{
			PulseNumber:     receivedPulse.PulseNumber,
			PrevPulseNumber: receivedPulse.PrevPulseNumber,
			RecordAmount:    receivedPulse.RecordAmount,
		})
		if err != nil {
			logger.Error(err)
			return err
		}

		currentPN = receivedPulse.PulseNumber
	}
}
