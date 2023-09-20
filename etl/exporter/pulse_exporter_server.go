package exporter

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/block-explorer/etl/interfaces"
)

type PulseServer struct {
	repository  interfaces.Storage
	pulsePeriod time.Duration
	logger      *log.Logger
}

func NewPulseServer(repo interfaces.Storage, pulsePeriod time.Duration, logger *log.Logger) *PulseServer {
	return &PulseServer{repo, pulsePeriod, logger}
}

func (s *PulseServer) GetNextPulse(req *GetNextPulseRequest, stream PulseExporter_GetNextPulseServer) error {
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
			if s.logger != nil {
				s.logger.Error(err)
			}
			return err
		}

		currentPN = receivedPulse.PulseNumber
	}
}
