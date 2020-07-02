// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"encoding/base64"
	"fmt"

	"github.com/insolar/block-explorer/instrumentation"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"

	"github.com/insolar/block-explorer/etl/models"
)

func NullableString(s string) *string {
	return &s
}

func RecordToAPI(record models.Record) server.Record {
	pulseNumber := int64(record.PulseNumber)
	jetID := record.JetID
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
	if !instrumentation.IsEmpty(record.ObjectReference) {
		objectID := insolar.NewIDFromBytes(record.ObjectReference)
		if objectID != nil {
			response.ObjectReference = NullableString(insolar.NewReference(*objectID).String())
		}
	}
	if !instrumentation.IsEmpty(record.PrevRecordReference) {
		prevRecordReference := insolar.NewIDFromBytes(record.PrevRecordReference)
		if prevRecordReference != nil {
			response.PrevRecordReference = NullableString(prevRecordReference.String())
		}
	}
	if !instrumentation.IsEmpty(record.PrototypeReference) {
		prototypeReference := insolar.NewIDFromBytes(record.PrototypeReference)
		if prototypeReference != nil {
			response.PrototypeReference = NullableString(prototypeReference.String())
		}
	}
	reference := insolar.NewIDFromBytes(record.Reference)
	if reference != nil {
		response.Reference = NullableString(reference.String())
	}
	return response
}

func PulseToAPI(pulse models.Pulse, jetDropAmount, recordAmount int64) server.Pulse {
	pulseNumber := int64(pulse.PulseNumber)
	prevPulseNumber := int64(pulse.PrevPulseNumber)
	nextPulseNumber := int64(pulse.NextPulseNumber)
	response := server.Pulse{
		IsComplete:      &pulse.IsComplete,
		JetDropAmount:   &jetDropAmount,
		NextPulseNumber: &nextPulseNumber,
		PrevPulseNumber: &prevPulseNumber,
		PulseNumber:     &pulseNumber,
		RecordAmount:    &recordAmount,
		Timestamp:       &pulse.Timestamp,
	}
	return response
}

func JetDropToAPI(jetDrop models.JetDrop) server.JetDrop {
	pulseNumber := int64(jetDrop.PulseNumber)
	recordAmount := int64(jetDrop.RecordAmount)
	// TODO: set correct prev and next after PENV-348
	nextJetDropID := []string{"test_next_jet_drop"}
	prevJetDropID := []string{"test_prev_jet_drop"}
	result := server.JetDrop{
		Hash:      NullableString(base64.StdEncoding.EncodeToString(jetDrop.Hash)),
		JetDropId: NullableString(models.NewJetDropID(jetDrop.JetID, int64(jetDrop.PulseNumber)).ToString()),
		JetId:     NullableString(jetDrop.JetID),
		// todo implement this if needed
		NextJetDropId: &nextJetDropID,
		PrevJetDropId: &prevJetDropID,
		PulseNumber:   &pulseNumber,
		RecordAmount:  &recordAmount,
		Timestamp:     &jetDrop.Timestamp,
	}
	return result
}
