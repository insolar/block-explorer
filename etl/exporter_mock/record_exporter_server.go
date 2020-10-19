// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter_mock

import (
	"bytes"

	"github.com/insolar/block-explorer/etl/exporter"
)

type RecordServerMock struct {
	*DataMock
}

func NewRecordServerMock(d *DataMock) *RecordServerMock {
	return &RecordServerMock{d}
}

func (s *RecordServerMock) GetRecords(in *exporter.GetRecordsRequest, stream exporter.RecordExporter_GetRecordsServer) error {
	p, ok := s.RecordsByPulse[in.PulseNumber]
	if !ok {
		return nil
	}
	iterateToRecord := int(in.RecordNumber + in.Count)
	if iterateToRecord > len(p) {
		iterateToRecord = len(p)
	}
	for i := int(in.RecordNumber); i < iterateToRecord; i++ {
		for _, proto := range in.Prototypes {
			if bytes.Equal(proto, p[i].PrototypeReference) {
				resp := &exporter.GetRecordsResponse{
					Polymorph:           0,
					RecordNumber:        uint32(p[i].Order),
					Reference:           p[i].Reference,
					Type:                string(p[i].Type),
					ObjectReference:     p[i].ObjectReference,
					PrototypeReference:  p[i].PrototypeReference,
					Payload:             p[i].Payload,
					PrevRecordReference: p[i].PrevRecordReference,
					PulseNumber:         p[i].PulseNumber,
					Timestamp:           uint32(p[i].Timestamp),
				}
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
