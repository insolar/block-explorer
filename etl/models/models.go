// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package models

import (
	"time"
)

type RecordType string

const (
	State   RecordType = "state"
	Request RecordType = "request"
	Result  RecordType = "result"
)

type Reference []byte

type Record struct {
	Reference           Reference
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
	Timestamp           time.Time
}

type JetDrop struct{}

type Pulse struct{}
