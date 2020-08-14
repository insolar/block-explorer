// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package types

import (
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"go.opencensus.io/stats/view"
)

func init() {
	// todo fix problem with importing two loggers PENV-344
	view.Unregister(&view.View{Name: "log_write_delays"})
}

// PlatformPulseData represents on the missing struct in the Platform
type PlatformPulseData struct {
	Pulse   *exporter.FullPulse
	Records []*exporter.Record
}

type JetDrop struct {
	MainSection *MainSection
	Sections    []Section
	RawData     []byte
	Hash        []byte
}

type Section interface {
	IsSection() bool
}

func (m MainSection) IsSection() bool { return true }

type MainSection struct {
	Start        DropStart
	DropContinue DropContinue
	Records      []Record
}

type AdditionalSection struct {
	RecordExtensions []Record
}

type DropStart struct {
	PulseData           Pulse
	JetDropPrefix       string
	JetDropPrefixLength uint
}

type DropContinue struct {
	PrevDropHash [][]byte
}

type Pulse struct {
	PulseNo         int64
	EpochPulseNo    int64
	PulseTimestamp  int64
	NextPulseNumber int64
	PrevPulseNumber int64
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
	Order               uint32
}
