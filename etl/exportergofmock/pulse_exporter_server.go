// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exportergofmock

import (
	"bytes"

	"github.com/insolar/block-explorer/etl/exporter"
)

type PulseServerMock struct {
	*DataMock
}

func NewPulseServerMock(d *DataMock) *PulseServerMock {
	return &PulseServerMock{d}
}

func (s *PulseServerMock) GetNextPulse(in *exporter.GetNextPulseRequest, stream exporter.PulseExporter_GetNextPulseServer) error {
	for _, proto := range in.Prototypes {
		for _, p := range s.Pulses {
			if p.PulseNumber < in.PulseNumberFrom {
				continue
			}
			var recsFound int64
			for _, r := range s.RecordsByPulseNumber[p.PulseNumber] {
				if bytes.Equal(r.PrototypeReference, proto) {
					recsFound++
				}
				// send empty pulses too
			}
			resp := &exporter.GetNextPulseResponse{
				PulseNumber:     p.PulseNumber,
				PrevPulseNumber: p.PulseNumber - 1,
				RecordAmount:    recsFound,
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
	return nil
}
