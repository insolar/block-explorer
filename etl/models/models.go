// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package models

import (
	"github.com/insolar/block-explorer/etl/types"
)

type RecordType string

func RecordTypeFromTypes(rt types.RecordType) RecordType {
	return []RecordType{"state", "request", "result"}[rt]
}

const (
	State   RecordType = "state"
	Request RecordType = "request"
	Result  RecordType = "result"
)

type Reference []byte

func ReferenceFromTypes(r types.Reference) Reference {
	return Reference(r)
}

type Record struct {
	Reference           Reference `gorm:"primary_key;auto_increment:false"`
	Type                RecordType
	ObjectReference     Reference
	PrototypeReference  Reference
	Payload             []byte
	PrevRecordReference Reference
	Hash                []byte
	RawData             []byte
	JetID               []byte
	PulseNumber         int
	Order               int
	Timestamp           int64
}

type JetDrop struct {
	JetID          []byte `gorm:"primary_key;auto_increment:false"`
	PulseNumber    int    `gorm:"primary_key;auto_increment:false"`
	FirstPrevHash  []byte
	SecondPrevHash []byte
	Hash           []byte
	RawData        []byte
	Timestamp      int64
	RecordAmount   int
}

type Pulse struct {
	PulseNumber     int `gorm:"primary_key;auto_increment:false"`
	PrevPulseNumber int
	NextPulseNumber int
	IsComplete      bool
	Timestamp       int64
}
