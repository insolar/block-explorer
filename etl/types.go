// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package etl

import (
	"time"
)

type JetDrop struct {
	sections []Section
	RawData  []byte
}

type Section interface {
	IsSection() bool
}

func (m MainSection) IsSection() bool {
	panic("implement me")
}

type MainSection struct {
	start        DropStart
	dropContinue DropContinue
	sections     []uint
	records      []Record
}

type AdditionalSection struct {
	recordExtensions []Record
}

type DropStart struct {
	pulseData           Pulse
	jetDropPrefix       []byte
	jetDropPrefixLength uint
}

type DropContinue struct {
	PrevDropHash [][]byte
}

type Pulse struct {
	pulseNo        int
	epochPulseNo   int
	pulseTimestamp time.Time
	nextPulseDelta int
	prevPulseDelta int
}

type RecordType int

const (
	// state type means activate, amend, deactivate records
	STATE RecordType = iota
	REQUEST
	RESULT
)

// Reference based on Insolar.Reference
type Reference []byte
type Record struct {
	Type                RecordType
	Ref                 Reference
	ObjectReference     Reference
	PrototypeReference  Reference
	PrevRecordReference Reference
	RecordPayload       []byte
	Hash                []byte
	RawData             []byte
}
