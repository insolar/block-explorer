// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter_mock

import (
	"github.com/insolar/block-explorer/etl/exporter"
)

type RecordServerMock struct {
	*DataMock
}

func NewRecordServerMock(d *DataMock) *RecordServerMock {
	return &RecordServerMock{d}
}

func (s *RecordServerMock) GetRecords(in *exporter.GetRecordsRequest, stream exporter.RecordExporter_GetRecordsServer) error {
	// for _, p := range s.RecordsByPulse {
	// 	for _, r := range p {
	// 		for _, proto := range in.Prototypes {
	// 			if bytes.Equal(proto, r.PrototypeReference) {
	// 				resp := &exporter.GetRecordsResponse{
	// 					Polymorph:           0,
	// 					RecordNumber:        uint32(r.Order),
	// 					Reference:           r.Reference,
	// 					Type:                string(r.Type),
	// 					ObjectReference:     r.ObjectReference,
	// 					PrototypeReference:  r.PrototypeReference,
	// 					Payload:             r.Payload,
	// 					PrevRecordReference: r.PrevRecordReference,
	// 					PulseNumber:         uint32(r.PulseNumber),
	// 					Timestamp:           uint32(r.Timestamp),
	// 				}
	// 				if err := stream.Send(resp); err != nil {
	// 					return err
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	return nil
}
