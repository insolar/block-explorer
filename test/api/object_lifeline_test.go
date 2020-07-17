// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"testing"

	"github.com/antihax/optional"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar/gen"
	ins_record "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

func TestLifeline_onePulse(t *testing.T) {
	t.Log("C4993 Receive object lifeline, states belong to one pulse")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 1
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)

	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[0].Pn + 10)

	records := make([]*exporter.Record, 0)
	lifelineRecords := lifeline.GetAllRecords()
	records = append(records, lifelineRecords...)
	records = append(records, lastPulseRecord)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(lifelineRecords)+1, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), nil)
	require.NoError(t, err)
	require.Len(t, response.Result, len(lifeline.GetStateRecords()))
	for _, res := range response.Result {
		require.Contains(t, lifeline.ObjID.String(), res.ObjectReference)
		require.Equal(t, int64(lifeline.StateRecords[0].Pn), res.PulseNumber)
	}
}

func TestLifeline_severalPulses(t *testing.T) {
	t.Log("C4994 Receive object lifeline, states belong to several pulses")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 4
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)

	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[0].Pn + 10)
	records := make([]*exporter.Record, 0)
	lifelineRecords := lifeline.GetAllRecords()
	records = append(records, lifelineRecords...)
	records = append(records, lastPulseRecord)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(lifelineRecords)+1, 5000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, len(lifeline.GetStateRecords()))
	pulses := make([]int64, pulsesNumber)
	for i, s := range lifeline.StateRecords {
		pulses[i] = int64(s.Pn)
	}
	for _, res := range response.Result {
		require.Contains(t, lifeline.ObjID.String(), res.ObjectReference)
		require.Contains(t, pulses, res.PulseNumber)
	}
}

func TestLifeline_amendRecords(t *testing.T) {
	t.Log("C4999 Receive object lifeline, if received only linked amend records")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	count := 5
	lifeline := testutils.GenerateObjectLifeline(1, count)
	allRecords := lifeline.GetStateRecords()

	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[0].Pn)
	allRecords = append(allRecords, lastPulseRecord)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, allRecords)
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, count+1, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, count)
}

func TestLifeline_removedStatesBetweenPulses(t *testing.T) {
	t.Log("C5000 Receive object lifeline, if there are skipped object states at the end and at the beginning of the pulses")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	allRecords := make([]*exporter.Record, 0)
	pn := gen.PulseNumber()
	jID := testutils.GenerateUniqueJetID()
	prevState := gen.ID()
	objID := gen.ID()
	count := 5
	recordsFirstArray := testutils.GenerateVirtualAmendRecordsLinkedArray(pn, jID, objID, prevState, count)
	prevState = recordsFirstArray[len(recordsFirstArray)-1].Record.ID
	recordsSecondArray := testutils.GenerateVirtualAmendRecordsLinkedArray(pn+10, jID, objID, prevState, count)
	prevState = recordsSecondArray[len(recordsSecondArray)-1].Record.ID
	recordsThirdArray := testutils.GenerateVirtualAmendRecordsLinkedArray(pn+20, jID, objID, prevState, count)

	// Take a part of linked records.
	// Removing records from the beginning and from the end of an array within a pulse
	// NOTE: when the Extractor logic will change, this test should fail and must be refactored to a negative)
	allRecords = append(allRecords, recordsFirstArray[:3]...)
	allRecords = append(allRecords, recordsSecondArray[2:]...)
	allRecords = append(allRecords, recordsThirdArray[:3]...)

	lastPulseRecord := testutils.GenerateRecordInNextPulse(pn + 30)
	allRecords = append(allRecords, lastPulseRecord)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, allRecords)
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	expCount := len(allRecords) - 1
	ts.WaitRecordsCount(t, expCount+1, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, objID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, expCount)
}

func TestLifeline_removedStatesWithinPulses(t *testing.T) {
	t.Log("C5110 Receive object lifeline, unable to build lifeline for pulse if there are skipped object states within pulse")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 2
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)
	records := make([]*exporter.Record, 0)
	// skipping records within one pulse, all these records won't be processed
	records = append(records, lifeline.StateRecords[0].Records[:4]...)
	records = append(records, lifeline.StateRecords[0].Records[6:]...)
	// records from second pulse will be processed and expected in DB
	records = append(records, lifeline.StateRecords[1].Records...)
	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[0].Pn + 100)
	records = append(records, lastPulseRecord)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, recordsInPulse+1, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, recordsInPulse)
}

func TestLifeline_recordsHaveSamePrevState(t *testing.T) {
	t.Log("C5004 Receive object lifeline, unable to build lifeline for pulse if several states have the same prev state")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 3
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)
	records := make([]*exporter.Record, 0)
	stateRecordsFirstPulse := lifeline.StateRecords[0]
	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[1].Pn)

	prevState := stateRecordsFirstPulse.Records[3].Record.ID
	for i := 5; i < len(stateRecordsFirstPulse.Records); i++ {
		union := stateRecordsFirstPulse.Records[i].Record.Virtual.Union.(*ins_record.Virtual_Amend)
		union.Amend.PrevState = prevState
		stateRecordsFirstPulse.Records[i].Record.Virtual.Union = union
	}
	records = append(records, stateRecordsFirstPulse.Records...)
	records = append(records, lifeline.StateRecords[1].Records...)
	records = append(records, lastPulseRecord)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, recordsInPulse+1, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, recordsInPulse)
}

func TestLifeline_receiveNewObjectStates(t *testing.T) {
	t.Log("C5082 Receive object lifeline, receive new object states over incoming pulses")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 5
	recordsInPulse := 2
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.StateRecords[0].Records)
	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.StateRecords[1].Records)
	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.StateRecords[2].Records)
	require.NoError(t, err)
	// expected records from pulses 1, 2, 3

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, recordsInPulse*3, 1000)

	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.StateRecords[3].Records)
	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.StateRecords[4].Records)
	require.NoError(t, err)
	// expected records from pulses 1, 2, 3, 4, 5

	ts.WaitRecordsCount(t, recordsInPulse*5, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, recordsInPulse*pulsesNumber)
}

func TestLifeline_fillMissedStates(t *testing.T) {
	t.Log("C5083 Receive object lifeline, fill missed object states between gaps over incoming records in one pulse")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 2
	recordsInPulse := 5
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)
	records := make([]*exporter.Record, 0)
	recordsPulseOne := lifeline.StateRecords[0].Records
	records = append(records, recordsPulseOne[:2]...)
	records = append(records, recordsPulseOne[3:]...)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.CheckForRecordsNotChanged(t, 0, 500)

	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{recordsPulseOne[2]})
	require.NoError(t, err)

	ts.CheckForRecordsNotChanged(t, 0, 500)

	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, lifeline.StateRecords[1].Records)
	require.NoError(t, err)
	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[1].Pn)
	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{lastPulseRecord})
	lenExpRecords := recordsInPulse * pulsesNumber

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, lenExpRecords+1, 1000)

	c := GetHTTPClient()
	response, err := c.ObjectLifeline(t, lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, lenExpRecords)
}
