// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package heavymock

import (
	"io"
	"sync"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"

	"github.com/insolar/block-explorer/testutils"
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
			if r.Equal(s.record) && !s.isSent {
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

func (s *ImporterServer) GetLowestUnsentPulse() (insolar.PulseNumber, []exporter.JetDropContinue) {
	pulse := insolar.PulseNumber(1<<32 - 1)
	jets := map[insolar.PulseNumber]map[insolar.JetID]exporter.JetDropContinue{}
	for _, r := range s.GetUnsentRecords() {
		if r.Record.ID.Pulse() > pulse {
			continue
		}
		pulse = r.Record.ID.Pulse()
		if jets[r.Record.ID.Pulse()] == nil {
			jets[r.Record.ID.Pulse()] = map[insolar.JetID]exporter.JetDropContinue{}
		}
		jets[r.Record.ID.Pulse()][r.Record.JetID] = exporter.JetDropContinue{JetID: r.Record.JetID, Hash: testutils.GenerateRandBytes()}
	}
	var res []exporter.JetDropContinue
	for _, jetDrop := range jets[pulse] {
		res = append(res, jetDrop)
	}
	return pulse, res
}
