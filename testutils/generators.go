// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"io"

	"github.com/insolar/insolar/insolar/gen"
	insrecord "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

// return a function for generating record
func GenerateRecords(batchSize int) func() (record *exporter.Record, e error) {
	pn := gen.PulseNumber()
	cnt := 0
	eof := true

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
				ID:    gen.IDWithPulse(pn),
				JetID: gen.JetID(),
			},
			ShouldIterateFrom: nil,
		}, nil
	}

	return generateRecords
}
