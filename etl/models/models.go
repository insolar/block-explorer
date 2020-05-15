// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package models

import (
	"time"

	"github.com/insolar/block-explorer/etl"
)

type Record struct {
	Reference          []byte
	JetID              []byte
	PulseNumber        int
	Type               etl.RecordType
	ObjectReference    []byte
	PrototypeReference []byte
	Payload            []byte
	PrevState          []byte
	Hash               []byte
	Order              int
	RawData            []byte
	Timestamp          time.Time
}
