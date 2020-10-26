// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exportergofmock

import (
	"github.com/insolar/block-explorer/etl/models"
)

type DataMock struct {
	CurrentPulse         int64
	Pulses               []models.Pulse
	RecordsByPulseNumber map[int64][]models.Record
}

func NewDataMock(initPulse int64) *DataMock {
	return &DataMock{
		CurrentPulse:         initPulse,
		Pulses:               make([]models.Pulse, 0),
		RecordsByPulseNumber: make(map[int64][]models.Record),
	}
}
