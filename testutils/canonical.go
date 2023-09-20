package testutils

import (
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/insolar/insolar/gen"

	"github.com/insolar/block-explorer/etl/types"
)

// CreateJetDropCanonical returns generated jet drop with provided record and without prevHash
func CreateJetDropCanonical(records []types.Record) types.JetDrop {
	pn := int64(gen.PulseNumber())
	return types.JetDrop{
		MainSection: &types.MainSection{
			Start: types.DropStart{
				PulseData: types.Pulse{
					PulseNo:         pn,
					PrevPulseNumber: pn - 10,
					NextPulseNumber: pn + 10,
				},
				JetDropPrefix:       converter.JetIDToString(gen.JetID()),
				JetDropPrefixLength: uint(gen.JetID().Depth()),
			},
			DropContinue: types.DropContinue{},
			Records:      records,
		},
		Sections: nil,
		RawData:  GenerateRandBytes(),
		Hash:     GenerateRandBytes(),
	}
}

// CreateRecordCanonical returns generated record
func CreateRecordCanonical() types.Record {
	return types.Record{
		Type:                types.STATE,
		Ref:                 gen.Reference().Bytes(),
		ObjectReference:     gen.Reference().Bytes(),
		PrototypeReference:  gen.Reference().Bytes(),
		PrevRecordReference: gen.Reference().Bytes(),
		RecordPayload:       GenerateRandBytes(),
		Hash:                GenerateRandBytes(),
		RawData:             GenerateRandBytes(),
		Order:               0,
	}
}
