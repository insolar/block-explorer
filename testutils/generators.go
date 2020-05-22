// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"encoding/binary"
	"io"
	"sync"

	utils "github.com/insolar/common-test/generator"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	insrecord "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

// return a function for generating record
func GenerateRecords(batchSize int) func() (record *exporter.Record, e error) {
	pn := gen.PulseNumber()
	cnt := 0
	eof := true
	randNum := func() int64 {
		return utils.RandNumberOverRange(100, 500)
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
				JetID:     gen.JetID(),
				Signature: []byte{0, 1, 2},
			},
			ShouldIterateFrom: &pn,
			Polymorph:         uint32(randNum()),
		}, nil
	}

	return generateRecords
}

func GenerateRecordsList(count int) *[]exporter.Record {
	var res []exporter.Record
	f := GenerateRecords(count)
	for {
		record, err := f()
		if err != nil {
			break
		}
		res = append(res, *record)
	}
	return &res
}

var uniqueJetId = make(map[uint64]bool)
var mutex = &sync.Mutex{}

func GenerateUniqueJetID() insolar.JetID {
	for {
		jetID := gen.JetID()
		id := binary.BigEndian.Uint64(jetID.Prefix())
		mutex.Lock()
		_, hasKey := uniqueJetId[id]
		if !hasKey {
			uniqueJetId[id] = true
			mutex.Unlock()
			return jetID
		}
		mutex.Unlock()
	}
}
