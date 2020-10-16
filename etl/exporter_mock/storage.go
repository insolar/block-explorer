package exporter_mock

import (
	"github.com/insolar/block-explorer/etl/models"
)

type DataMock struct {
	CurrentPulse   int64
	Pulses         []models.Pulse
	Records        []models.Record
	RecordsByPulse map[int64][]models.Record
}

func NewDataMock() *DataMock {
	return &DataMock{
		CurrentPulse:   4000000,
		Pulses:         make([]models.Pulse, 0),
		Records:        make([]models.Record, 0),
		RecordsByPulse: make(map[int64][]models.Record),
	}
}
