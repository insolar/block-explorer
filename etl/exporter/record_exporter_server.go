// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter

type RecordServer struct {
}

func NewRecordServer() *RecordServer {
	return &RecordServer{}
}

func (s *RecordServer) GetRecords(*GetRecordsRequest, RecordExporter_GetRecordsServer) error {
	return nil
}
