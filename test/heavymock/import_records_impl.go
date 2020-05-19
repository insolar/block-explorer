// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package heavymock

import (
	"io"

	"github.com/insolar/insolar/ledger/heavy/exporter"
)

type ImporterServer struct {
	savedRecords []exporter.Record
}

func NewHeavymockImporter() *ImporterServer {
	return &ImporterServer{}
}

func (s *ImporterServer) Import(stream HeavymockImporter_ImportServer) error {
	for {
		record, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&Ok{
				Ok: true,
			})
		}
		if err != nil {
			return err
		}
		s.savedRecords = append(s.savedRecords, *record)
	}
}

func (s *ImporterServer) GetSavedRecords() []exporter.Record {
	return s.savedRecords
}
