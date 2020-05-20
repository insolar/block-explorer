// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package models

type RecordType string

const (
	State   RecordType = "state"
	Request RecordType = "request"
	Result  RecordType = "result"
)

type Reference []byte

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
}

type Pulse struct {
	PulseNumber     int `gorm:"primary_key;auto_increment:false"`
	PrevPulseNumber int
	NextPulseNumber int
	IsComplete      bool
	Timestamp       int64
}
