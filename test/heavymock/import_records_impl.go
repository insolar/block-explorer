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
	records []*savedRecord
	mux     sync.Mutex
}

type savedRecord struct {
	record *exporter.Record
	isSent bool
}

func NewHeavymockImporter() *ImporterServer {
	return &ImporterServer{}
}

func (s *ImporterServer) Import(stream HeavymockImporter_ImportServer) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	received := make([]*exporter.Record, 0)
	for {
		record, err := stream.Recv()
		if err == io.EOF {
			s.collectRecords(received)
			return stream.SendAndClose(&Ok{
				Ok: true,
			})
		}
		if err != nil {
			return err
		}
		received = append(received, record)
	}
}

func (s *ImporterServer) GetUnsentRecords() []*exporter.Record {
	s.mux.Lock()
	defer s.mux.Unlock()
	res := make([]*exporter.Record, 0)
	for _, r := range s.records {
		if !r.isSent {
			res = append(res, r.record)
		}
	}

	return res
}

func (s *ImporterServer) MarkAsSent(records []*exporter.Record) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, r := range records {
		for _, s := range s.records {
			if r.Equal(s.record) {
				s.isSent = true
				break
			}
		}
	}
}

func (s *ImporterServer) collectRecords(records []*exporter.Record) {
	l := len(records)
	slice := make([]*savedRecord, l)
	for i := 0; i < l; i++ {
		slice[i] = &savedRecord{records[i], false}
	}
	s.records = append(s.records, slice...)
}
