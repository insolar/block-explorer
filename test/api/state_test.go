// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"testing"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/require"
)

const (
	stateTypeActivate   = "activate"
	stateTypeAmend      = "amend"
	stateTypeDeactivate = "deactivate"
)

func TestGetStateByStateReference(t *testing.T) {
	t.Skip("endpoint deprecated in be-api specification")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 2
	recordsPulseCount := 2
	lifeline := testutils.GenerateObjectLifeline(pulsesCount, recordsPulseCount)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.GetAllRecords()))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(lifeline.GetStateRecords()), 5000)

	c := GetHTTPClient()
	stateRecords := lifeline.GetStateRecords()

	t.Run("amend record", func(t *testing.T) {
		amendRecord := stateRecords[1]
		id := amendRecord.Record.ID.Bytes()
		ref := insolar.NewRecordReference(*insolar.NewIDFromBytes(id)).String()
		amend, ok := amendRecord.Record.Virtual.Union.(*ins_record.Virtual_Amend)
		require.True(t, ok)
		a := amend.Amend
		prevState := a.PrevState.Bytes()
		prevStateRef := insolar.NewRecordReference(*insolar.NewIDFromBytes(prevState)).String()
		request := a.Request.Bytes()
		requestReference := insolar.NewRecordReference(*insolar.NewIDFromBytes(request)).String()
		prototype := a.Image.Bytes()
		prototypeRef := insolar.NewRecordReference(*insolar.NewIDFromBytes(prototype)).String()
		expJetID := converter.JetIDToString(amendRecord.Record.JetID)

		response := c.State(t, ref)
		require.Equal(t, ref, response.Reference)
		require.Equal(t, stateTypeAmend, response.Type)
		require.Equal(t, amendRecord.Record.ID.Pulse().AsUint32(), uint32(response.PulseNumber))
		require.Equal(t, requestReference, response.RequestReference)
		require.Equal(t, insolar.NewReference(amendRecord.Record.ObjectID).String(), response.ObjectReference)
		require.Equal(t, prevStateRef, response.PrevStateReference)
		require.Equal(t, expJetID, response.JetId)
		require.Equal(t, fmt.Sprintf("%v:%v", expJetID, amendRecord.Record.ID.Pulse().String()), response.JetDropId)
		require.Equal(t, prototypeRef, response.PrototypeReference)
		require.Empty(t, ref, response.ParentReference) // only activate record has not empty value
		require.NotEmpty(t, ref, response.Order)
		require.NotEmpty(t, ref, response.Hash)
		require.NotEmpty(t, ref, response.Payload)
		require.NotEmpty(t, ref, response.Timestamp)
		require.Empty(t, ref, response.Code)
		require.Empty(t, ref, response.Message)
		require.Empty(t, ref, response.Description)
		require.Empty(t, ref, response.ValidationFailures)
	})
	t.Run("activate record", func(t *testing.T) {
		activateRecord := lifeline.StateRecords[0].Records[0]
		id := activateRecord.Record.ID.Bytes()
		ref := insolar.NewRecordReference(*insolar.NewIDFromBytes(id)).String()
		activate, ok := activateRecord.Record.Virtual.Union.(*ins_record.Virtual_Activate)
		require.True(t, ok)
		a := activate.Activate
		request := a.Request.Bytes()
		requestReference := insolar.NewRecordReference(*insolar.NewIDFromBytes(request)).String()
		expJetID := converter.JetIDToString(activateRecord.Record.JetID)
		prototype := a.Image.Bytes()
		prototypeRef := insolar.NewRecordReference(*insolar.NewIDFromBytes(prototype)).String()
		parent := a.Parent.Bytes()
		parentRef := insolar.NewRecordReference(*insolar.NewIDFromBytes(parent)).String()

		response := c.State(t, ref)
		require.Equal(t, ref, response.Reference)
		require.Equal(t, stateTypeActivate, response.Type)
		require.Equal(t, activateRecord.Record.ID.Pulse().AsUint32(), uint32(response.PulseNumber))
		require.Equal(t, requestReference, response.RequestReference)
		require.Equal(t, insolar.NewReference(activateRecord.Record.ObjectID).String(), response.ObjectReference)
		require.Empty(t, response.PrevStateReference)
		require.Equal(t, expJetID, response.JetId)
		require.Equal(t, fmt.Sprintf("%v:%v", expJetID, activateRecord.Record.ID.Pulse().String()), response.JetDropId)
		require.Equal(t, prototypeRef, response.PrototypeReference)
		require.Equal(t, parentRef, response.ParentReference)
		require.NotEmpty(t, ref, response.Order)
		require.NotEmpty(t, ref, response.Hash)
		require.NotEmpty(t, ref, response.Payload)
		require.NotEmpty(t, ref, response.Timestamp)
		require.Empty(t, ref, response.Code)
		require.Empty(t, ref, response.Message)
		require.Empty(t, ref, response.Description)
		require.Empty(t, ref, response.ValidationFailures)
	})
	t.Run("deactivate record", func(t *testing.T) {
		records := lifeline.StateRecords[pulsesCount-1].Records
		deactivateRecord := records[len(records)-1]
		id := deactivateRecord.Record.ID.Bytes()
		ref := insolar.NewRecordReference(*insolar.NewIDFromBytes(id)).String()
		deactivate, ok := deactivateRecord.Record.Virtual.Union.(*ins_record.Virtual_Deactivate)
		require.True(t, ok)
		d := deactivate.Deactivate
		request := d.Request.Bytes()
		requestReference := insolar.NewRecordReference(*insolar.NewIDFromBytes(request)).String()
		prevState := d.PrevState.Bytes()
		prevStateRef := insolar.NewRecordReference(*insolar.NewIDFromBytes(prevState)).String()
		expJetID := converter.JetIDToString(deactivateRecord.Record.JetID)

		response := c.State(t, ref)
		require.Equal(t, ref, response.Reference)
		require.Equal(t, stateTypeDeactivate, response.Type)
		require.Equal(t, deactivateRecord.Record.ID.Pulse().AsUint32(), uint32(response.PulseNumber))
		require.Equal(t, requestReference, response.RequestReference)
		require.Equal(t, insolar.NewReference(deactivateRecord.Record.ObjectID).String(), response.ObjectReference)
		require.Equal(t, prevStateRef, response.PrevStateReference)
		require.Equal(t, expJetID, response.JetId)
		require.Equal(t, fmt.Sprintf("%v:%v", expJetID, deactivateRecord.Record.ID.Pulse().String()), response.JetDropId)
		require.Empty(t, response.PrototypeReference)
		require.Empty(t, response.ParentReference)
		require.NotEmpty(t, ref, response.Order)
		require.NotEmpty(t, ref, response.Hash)
		require.NotEmpty(t, ref, response.Payload)
		require.NotEmpty(t, ref, response.Timestamp)
		require.Empty(t, ref, response.Code)
		require.Empty(t, ref, response.Message)
		require.Empty(t, ref, response.Description)
		require.Empty(t, ref, response.ValidationFailures)
	})
	t.Run("error get nonexistent record", func(t *testing.T) {
		ref := insolar.NewRecordReference(*insolar.NewIDFromBytes(gen.Reference().Bytes())).String()
		response := c.StateWithError(t, ref, notFound404)
		require.NotEmpty(t, response.Message)
		require.NotEmpty(t, response.Code)
	})
}
