// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"encoding/base64"
	"fmt"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"

	"github.com/insolar/block-explorer/instrumentation"

	"github.com/insolar/block-explorer/etl/models"
)

func NullableString(s string) *string {
	return &s
}

func RecordToAPI(record models.Record) server.Record {
	pulseNumber := record.PulseNumber
	jetDropID := models.NewJetDropID(record.JetID, pulseNumber)

	response := server.Record{
		RecordAbstract: server.RecordAbstract{
			Hash:        NullableString(base64.StdEncoding.EncodeToString(record.Hash)),
			JetId:       NullableString(jetDropID.JetIDToString()),
			PulseNumber: &pulseNumber,
			Timestamp:   &record.Timestamp,
		},
		JetDropId: NullableString(jetDropID.ToString()),
		Index:     NullableString(fmt.Sprintf("%d:%d", record.PulseNumber, record.Order)),
		Payload:   NullableString(base64.StdEncoding.EncodeToString(record.Payload)),
		Type:      NullableString(string(record.Type)),
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

func StateToAPI(state models.State) server.State {
	pulseNumber := state.PulseNumber
	order := int64(state.Order)
	jetDropID := models.NewJetDropID(state.JetID, pulseNumber)
	response := server.State{
		Hash:        NullableString(base64.StdEncoding.EncodeToString(state.Hash)),
		JetDropId:   NullableString(jetDropID.ToString()),
		JetId:       NullableString(jetDropID.JetIDToString()),
		Order:       &order,
		Payload:     NullableString(base64.StdEncoding.EncodeToString(state.Payload)),
		Index:       NullableString(fmt.Sprintf("%d:%d", state.PulseNumber, state.Order)),
		PulseNumber: &pulseNumber,
		Timestamp:   &state.Timestamp,
		Type:        NullableString(string(state.Type)),
	}

	objectID := insolar.NewIDFromBytes(state.ObjectReference)
	if objectID != nil {
		response.ObjectReference = NullableString(insolar.NewReference(*objectID).String())
	}
	if !instrumentation.IsEmpty(state.PrevStateReference) {
		prevStateReference := insolar.NewIDFromBytes(state.PrevStateReference)
		if prevStateReference != nil {
			response.PrevStateReference = NullableString(prevStateReference.String())
		}
	}

	if !instrumentation.IsEmpty(state.ImageReference) {
		prototypeReference := insolar.NewIDFromBytes(state.ImageReference)
		if prototypeReference != nil {
			response.PrototypeReference = NullableString(prototypeReference.String())
		}
	}

	if !instrumentation.IsEmpty(state.ParentReference) {
		parentReference := insolar.NewIDFromBytes(state.ParentReference)
		if parentReference != nil {
			response.ParentReference = NullableString(parentReference.String())
		}
	}
	if !instrumentation.IsEmpty(state.RequestReference) {
		requestReference := insolar.NewIDFromBytes(state.RequestReference)
		if requestReference != nil {
			response.RequestReference = NullableString(requestReference.String())
		}
	}

	reference := insolar.NewIDFromBytes(state.RecordReference)
	if reference != nil {
		response.Reference = NullableString(reference.String())
	}
	return response
}

func PulseToAPI(pulse models.Pulse) server.Pulse {
	pulseNumber := pulse.PulseNumber
	prevPulseNumber := pulse.PrevPulseNumber
	nextPulseNumber := pulse.NextPulseNumber
	response := server.Pulse{
		IsComplete:    &pulse.IsComplete,
		JetDropAmount: &pulse.JetDropAmount,
		PulseNumber:   &pulseNumber,
		RecordAmount:  &pulse.RecordAmount,
		Timestamp:     &pulse.Timestamp,
	}
	if prevPulseNumber != -1 {
		response.PrevPulseNumber = &prevPulseNumber
	}
	if nextPulseNumber != -1 {
		response.NextPulseNumber = &nextPulseNumber
	}
	return response
}

func JetDropToAPI(jetDrop models.JetDrop, prevJetDrops, nextJetDrops []server.NextPrevJetDrop) server.JetDrop {
	pulseNumber := jetDrop.PulseNumber
	recordAmount := int64(jetDrop.RecordAmount)

	jetDropID := models.NewJetDropID(jetDrop.JetID, jetDrop.PulseNumber)
	result := server.JetDrop{
		Hash:      NullableString(base64.StdEncoding.EncodeToString(jetDrop.Hash)),
		JetDropId: NullableString(jetDropID.ToString()),
		JetId:     NullableString(jetDropID.JetIDToString()),
		// todo implement this if needed
		NextJetDropId: &nextJetDrops,
		PrevJetDropId: &prevJetDrops,
		PulseNumber:   &pulseNumber,
		RecordAmount:  &recordAmount,
		Timestamp:     &jetDrop.Timestamp,
	}
	return result
}
