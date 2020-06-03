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
					Union:     nil,
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

// GenerateRecordsWithDifferencePulses generates records with recordCount for each pulse
func GenerateRecordsWithDifferencePulses(differentPulseSize int, recordCount int) func() (record *exporter.Record, e error) {
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
