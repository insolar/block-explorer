// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package heavymock

import (
	"io"
	"sync"

	"github.com/insolar/insolar/ledger/heavy/exporter"
)

type ImporterServer struct {
	savedRecords []exporter.Record
	mux          sync.Mutex
}

func NewHeavymockImporter() *ImporterServer {
	return &ImporterServer{}
}

func (s *ImporterServer) Import(stream HeavymockImporter_ImportServer) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	received := make([]exporter.Record, 0)
	for {
		record, err := stream.Recv()
		if err == io.EOF {
			s.savedRecords = received
			return stream.SendAndClose(&Ok{
				Ok: true,
			})
		}
		if err != nil {
			return err
		}
		received = append(received, *record)
	}
}

func (s *ImporterServer) GetSavedRecords() []exporter.Record {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.savedRecords
}

func (s *ImporterServer) Cleanup() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.savedRecords = make([]exporter.Record, 0)
}
