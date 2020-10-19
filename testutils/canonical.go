// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/insolar/insolar/gen"

	"github.com/insolar/block-explorer/etl/types"
)

// CreateJetDropCanonical returns generated jet drop with provided record and without prevHash
func CreateJetDropCanonical(records []types.IRecord) types.JetDrop {
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

// CreateStateCanonical returns generated state
func CreateStateCanonical(stateType types.StateType) types.State {
	return types.State{
		RecordReference: gen.Reference().Bytes(),
		Type:            stateType,
		ObjectReference: gen.Reference().Bytes(),
		Request:         gen.Reference().Bytes(),
		Parent:          gen.Reference().Bytes(),
		IsPrototype:     false,
		Image:           gen.Reference().Bytes(),
		PrevState:       gen.Reference().Bytes(),
		Payload:         GenerateRandBytes(),
		Hash:            GenerateRandBytes(),
		RawData:         GenerateRandBytes(),
		Order:           0,
	}
}
