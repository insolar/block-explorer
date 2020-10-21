// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exportergofmock

import (
	"bytes"
	"sort"

	"github.com/insolar/block-explorer/etl/exporter"
)

type PulseServerMock struct {
	*DataMock
}

func NewPulseServerMock(d *DataMock) *PulseServerMock {
	return &PulseServerMock{d}
}

func (s *PulseServerMock) GetNextPulse(in *exporter.GetNextPulseRequest, stream exporter.PulseExporter_GetNextPulseServer) error {
	pulses := make([]int64, 0)
	for pKey := range s.RecordsByPulseNumber {
		pulses = append(pulses, pKey)
	}
	sort.Slice(pulses, func(i, j int) bool { return pulses[i] < pulses[j] })
	for _, proto := range in.Prototypes {
		for _, pNum := range pulses {
			if pNum < in.PulseNumberFrom {
				continue
			}
			var recsFound int64
			for _, r := range s.RecordsByPulseNumber[pNum] {
				if bytes.Equal(r.PrototypeReference, proto) {
					recsFound++
				}
				// send empty pulses too
			}
			resp := &exporter.GetNextPulseResponse{
				PulseNumber:     pNum,
				PrevPulseNumber: pNum - 1,
				RecordAmount:    recsFound,
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
	return nil
}
