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
	Records      []IRecord
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

// Reference based on Insolar.Reference
type Reference []byte

// TODO: https://insolar.atlassian.net/browse/PENV-802
type IRecord interface {
	TypeOf() RecordType
	Reference() Reference
}

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

type RecordType int

const (
	REQUEST RecordType = iota
	STATE
	RESULT
)

func (r Record) TypeOf() RecordType {
	return r.Type
}

func (r Record) Reference() Reference {
	return r.Ref
}

type StateType int

const (
	ACTIVATE StateType = iota
	AMEND
	DEACTIVATE
)

type State struct {
	RecordReference Reference // ref = r.RecordReference.ID.Bytes()
	Type            StateType
	ObjectReference Reference
	Request         Reference // reference to request
	Parent          Reference // has activate link to parent object
	IsPrototype     bool      // has activate, amend
	Image           Reference // has activate, amend
	PrevState       Reference // has amend, deactivate
	Payload         []byte
	RawData         []byte
	Hash            []byte // hash of record
	Order           uint32 // record number
}

func (s State) TypeOf() RecordType {
	return STATE
}

func (s State) Reference() Reference {
	return s.RecordReference
}

type Request struct {
}

func (r Request) TypeOf() RecordType {
	return REQUEST
}

type Result struct {
}

func (r Result) TypeOf() RecordType {
	return RESULT
}
