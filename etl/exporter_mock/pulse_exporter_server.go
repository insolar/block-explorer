// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter_mock

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
		for pNum, records := range s.RecordsByPulse {
			if pNum < in.PulseNumberFrom {
				continue
			}
			for _, r := range records {
				if bytes.Equal(r.PrototypeReference, proto) {
					resp := &exporter.GetNextPulseResponse{
						PulseNumber:     pNum,
						PrevPulseNumber: pNum - 1,
						// TODO: count records
						RecordAmount: 100,
					}
					if err := stream.Send(resp); err != nil {
						return err
					}
				}
				break
			}
		}
	}
	return nil
}
