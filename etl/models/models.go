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
	JetID               string
	PulseNumber         int
	Order               int
	Timestamp           int64
}

type JetDrop struct {
	JetID          string `gorm:"primary_key;auto_increment:false"`
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
	IsFinal         bool
	Timestamp       int64
}

type JetDropID struct {
	JetID       string
	PulseNumber int64
}

func NewJetDropID(jetID string, pulseNumber int64) *JetDropID {
	return &JetDropID{JetID: jetID, PulseNumber: pulseNumber}
}

func NewJetDropIDFromString(jetDropID string) (*JetDropID, error) {
	var pulse int64
	s := strings.Split(jetDropID, ":")
	if len(s) != 2 {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	_, err := strconv.ParseInt(s[0], 2, 64)
	if err != nil {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	pulse, err = strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("wrong jet drop id format")
	}

	return &JetDropID{JetID: s[0], PulseNumber: pulse}, nil
}

func (j *JetDropID) ToString() string {
	return fmt.Sprintf("%s:%d", j.JetID, j.PulseNumber)
}
