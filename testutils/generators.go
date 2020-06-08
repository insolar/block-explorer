// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"encoding/binary"
	"io"
	"math/rand"
	"sync"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	insrecord "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateRequestRecord(pulse insolar.PulseNumber, objectID insolar.ID) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	reference := insolar.NewReference(id)
	r.Record.Virtual.Union = &insrecord.Virtual_IncomingRequest{
		IncomingRequest: &insrecord.IncomingRequest{
			Object: reference,
		},
	}
	return r
}

func GenerateVirtualActivateRecord(pulse insolar.PulseNumber, objectID, requestID insolar.ID) (record *exporter.Record) {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	requestRerence := insolar.NewReference(requestID)
	r.Record.Virtual.Union = &insrecord.Virtual_Activate{
		Activate: &insrecord.Activate{
			Image:   gen.Reference(),
			Request: *requestRerence,
		},
	}
	return r
}

func GenerateVirtualAmendRecordsLinkedArray(pulse insolar.PulseNumber, jetID insolar.JetID, objectID, prevStateID insolar.ID, recordsCount int) []*exporter.Record {
	result := make([]*exporter.Record, recordsCount)
	for i := 0; i < recordsCount; i++ {
		r := GenerateVirtualAmendRecord(pulse, objectID, prevStateID)
		r.Record.JetID = jetID
		result[i] = r
		prevStateID = r.Record.ID
	}
	return result
}

func GenerateVirtualAmendRecord(pulse insolar.PulseNumber, objectID, prevStateID insolar.ID) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.Record.Virtual.Union = &insrecord.Virtual_Amend{
		Amend: &insrecord.Amend{
			Image:     gen.Reference(),
			PrevState: prevStateID,
		},
	}
	return r
}

func GenerateVirtualDeactivateRecord(pulse insolar.PulseNumber, objectID, prevStateID insolar.ID) *exporter.Record {
	r := GenerateRecordsSilence(1)[0]
	id := gen.IDWithPulse(pulse)
	r.Record.ID = id
	r.Record.ObjectID = objectID
	r.Record.Virtual.Union = &insrecord.Virtual_Deactivate{
		Deactivate: &insrecord.Deactivate{
			PrevState: prevStateID,
		},
	}
	return r
}

type ObjectLifeline struct {
	States []StateByPulse
	ObjID  insolar.ID
}

type StateByPulse struct {
	Pn      insolar.PulseNumber
	Records []*exporter.Record
}

func GenerateObjectLifeline(pulsesNumber, recordsInPulse int) ObjectLifeline {
	objectID := gen.ID()
	var prevState insolar.ID
	states := make([]StateByPulse, pulsesNumber)
	pn := gen.PulseNumber()
	for i := 0; i < pulsesNumber; i++ {
		jetID := GenerateUniqueJetID()
		pn = pn + 10
		records := make([]*exporter.Record, recordsInPulse)
		if i == 0 {
			records = make([]*exporter.Record, recordsInPulse+2)
			request := GenerateRequestRecord(pn, objectID)
			records[len(records)-1] = request

			activate := GenerateVirtualActivateRecord(pn, objectID, request.Record.ID)
			records[len(records)-2] = activate
			prevState = activate.Record.ID
		}
		amends := GenerateVirtualAmendRecordsLinkedArray(pn, jetID, objectID, prevState, recordsInPulse)
		for ii, r := range amends {
			records[ii] = r
		}
		prevState = amends[len(amends)-1].Record.ID
		if i == pulsesNumber-1 {
			deactivate := GenerateVirtualDeactivateRecord(pn, objectID, prevState)
			deactivate.Record.JetID = jetID
			records = append(records, deactivate)
		}

		states[i] = StateByPulse{
			Pn:      pn,
			Records: records,
		}
	}

	return ObjectLifeline{
		States: states,
		ObjID:  objectID,
	}

}

// GenerateRecords returns a function for generating record with error
func GenerateRecords(batchSize int) func() (record *exporter.Record, e error) {
	pn := gen.PulseNumber()
	cnt := 0
	eof := true
	randNum := func() int64 {
		return RandNumberOverRange(100, 500)
	}

	generateRecords := func() (record *exporter.Record, e error) {
		if !eof && cnt%batchSize == 0 {
			eof = true
			return &exporter.Record{}, io.EOF
		}
		cnt++
		eof = false
		return &exporter.Record{
			RecordNumber: uint32(cnt),
			Record: insrecord.Material{
				Polymorph: int32(randNum()),
				Virtual: insrecord.Virtual{
					Polymorph: int32(randNum()),
					Union: &insrecord.Virtual_IncomingRequest{
						IncomingRequest: &insrecord.IncomingRequest{
							Object: nil,
						},
					},
					Signature: []byte{0, 1, 2},
				},
				ID:        gen.IDWithPulse(pn),
				ObjectID:  gen.ID(),
				JetID:     GenerateUniqueJetID(),
				Signature: []byte{0, 1, 2},
			},
			ShouldIterateFrom: &pn,
			Polymorph:         uint32(randNum()),
		}, nil
	}

	return generateRecords
}

func GenerateRecordsWithDifferencePulsesSilence(differentPulseSize, recordCount int) []*exporter.Record {
	record := GenerateRecordsWithDifferencePulses(differentPulseSize, recordCount)
	result := make([]*exporter.Record, 0)
	for i := 0; i < differentPulseSize*recordCount; i++ {
		r, err := record()
		if err != nil {
			continue
		}
		result = append(result, r)
	}
	return result
}

// GenerateRecordsFromPulseSilence returns new generated records without errors
func GenerateRecordsFromOneJetSilence(differentPulseSize, recordCount int) []*exporter.Record {
	records := GenerateRecordsWithDifferencePulsesSilence(differentPulseSize, recordCount)
	jetID := GenerateUniqueJetID()
	for _, r := range records {
		r.Record.JetID = jetID
	}
	return records
}

// GenerateRecordsWithDifferencePulses generates records with recordCount for each pulse
func GenerateRecordsWithDifferencePulses(differentPulseSize, recordCount int) func() (record *exporter.Record, e error) {
	var mu = &sync.Mutex{}
	i := 0
	localRecordCount := 0
	var prevRecord *exporter.Record = GenerateRecordsSilence(1)[0]
	fn := func() (*exporter.Record, error) {
		mu.Lock()
		defer mu.Unlock()
		if i < differentPulseSize {

			record := GenerateRecordsSilence(1)[0]
			record.ShouldIterateFrom = nil
			if localRecordCount < recordCount {
				localRecordCount++
				record.Record.ID = gen.IDWithPulse(prevRecord.Record.ID.Pulse())
				prevRecord = record
			} else {
				i++
				localRecordCount = 1
				record.Record.ID = gen.IDWithPulse(prevRecord.Record.ID.Pulse() + 10)
				prevRecord = record
			}
			return record, nil
		}
		return nil, io.EOF
	}
	return fn
}

// GenerateRecordsSilence returns new generated records without errors
func GenerateRecordsSilence(count int) []*exporter.Record {
	res := make([]*exporter.Record, count)
	f := GenerateRecords(count)
	for i := 0; i < count; {
		record, err := f()
		if err != nil {
			continue
		}
		res[i] = record
		i++
	}
	return res
}

var uniqueJetID = make(map[uint64]bool)
var mutex = &sync.Mutex{}

func GenerateUniqueJetID() insolar.JetID {
	for {
		jetID := gen.JetID()
		id := binary.BigEndian.Uint64(jetID.Prefix())
		mutex.Lock()
		_, hasKey := uniqueJetID[id]
		if !hasKey {
			uniqueJetID[id] = true
			mutex.Unlock()
			return jetID
		}
		mutex.Unlock()
	}
}

// RandNumberOverRange generates random number over a range
func RandNumberOverRange(min int64, max int64) int64 {
	return rand.Int63n(max-min+1) + min
}

// GenerateRandBytes generates random bytes array
func GenerateRandBytes() []byte {
	var hash []byte
	fuzz.New().NilChance(0).Fuzz(&hash)
	return hash
}
