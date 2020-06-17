// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package models

import (
	"fmt"
	"strconv"
	"strings"

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

func JetIDToString(jetID []byte) string {
	res := strings.Builder{}
	for i := 0; i < 5; i++ {
		bytePos, bitPos := i/8, 7-i%8

		byteValue := jetID[bytePos]
		bitValue := byteValue >> uint(bitPos) & 0x01
		bitString := strconv.Itoa(int(bitValue))
		res.WriteString(bitString)
	}
	return res.String()
}

func JetDropID(jetID []byte, pulseNumber int64) string {
	return fmt.Sprintf("%s:%d", JetIDToString(jetID), pulseNumber)
}
