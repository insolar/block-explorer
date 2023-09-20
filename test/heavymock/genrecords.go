package heavymock

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
)

var SimpleRecord = &exporter.Record{
	Polymorph:    1,
	RecordNumber: 100,
	Record: record.Material{
		ID: *insolar.NewID(65537, nil),
	},
	ShouldIterateFrom: nil,
}

func GetRecordsByPulse(pulse insolar.PulseNumber, count int) []exporter.Record {
	var res []exporter.Record
	for i := 0; i < count; i++ {
		res = append(res, exporter.Record{
			Polymorph:    1,
			RecordNumber: uint32(i),
			Record: record.Material{
				ID: gen.IDWithPulse(pulse),
			},
			ShouldIterateFrom: &pulse,
		})
	}
	return res
}

func GetRecordsByPulseNumber(pulse pulse.Number, count int) []exporter.Record {
	pulseNumber := insolar.NewPulseNumber(pulse.Bytes())
	return GetRecordsByPulse(pulseNumber, count)
}
