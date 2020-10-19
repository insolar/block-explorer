package exportergofmock

import (
	"github.com/insolar/block-explorer/etl/models"
)

type DataMock struct {
	CurrentPulse   int64
	Pulses         []models.Pulse
	RecordsByPulse map[int64][]models.Record
}

func NewDataMock(initPulse int64) *DataMock {
	return &DataMock{
		CurrentPulse:   initPulse,
		Pulses:         make([]models.Pulse, 0),
		RecordsByPulse: make(map[int64][]models.Record),
	}
}
