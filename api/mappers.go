// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"

	"github.com/insolar/block-explorer/etl/models"
)

func NullableString(s string) *string {
	return &s
}

func RecordToAPI(record models.Record) server.Record {
	pulseNumber := int64(record.PulseNumber)
	jetID := jetIDToString(record.JetID)
	jetDropID := fmt.Sprintf("%s:%d", jetID, record.PulseNumber)
	response := server.Record{
		Hash:        NullableString(base64.StdEncoding.EncodeToString(record.Hash)),
		JetDropId:   NullableString(jetDropID),
		JetId:       NullableString(jetID),
		Index:       NullableString(fmt.Sprintf("%d:%d", record.PulseNumber, record.Order)),
		Payload:     NullableString(base64.StdEncoding.EncodeToString(record.Payload)),
		PulseNumber: &pulseNumber,
		Timestamp:   &record.Timestamp,
		Type:        NullableString(string(record.Type)),
	}
	if !bytes.Equal([]byte{}, record.ObjectReference) {
		objectID := insolar.NewIDFromBytes(record.ObjectReference)
		if objectID != nil {
			response.ObjectReference = NullableString(insolar.NewReference(*objectID).String())
		}
	}
	if !bytes.Equal([]byte{}, record.PrevRecordReference) {
		prevRecordReference := insolar.NewIDFromBytes(record.PrevRecordReference)
		if prevRecordReference != nil {
			response.PrevRecordReference = NullableString(prevRecordReference.String())
		}
	}
	prototypeReference := insolar.NewIDFromBytes(record.PrototypeReference)
	if prototypeReference != nil {
		response.PrototypeReference = NullableString(prototypeReference.String())
	}
	reference := insolar.NewIDFromBytes(record.Reference)
	if reference != nil {
		response.Reference = NullableString(reference.String())
	}
	return response
}

func jetIDToString(prefix []byte) string {
	res := strings.Builder{}
	for i := 0; i < 5; i++ {
		bytePos, bitPos := i/8, 7-i%8

		byteValue := prefix[bytePos]
		bitValue := byteValue >> uint(bitPos) & 0x01
		bitString := strconv.Itoa(int(bitValue))
		res.WriteString(bitString)
	}
	return res.String()
}
