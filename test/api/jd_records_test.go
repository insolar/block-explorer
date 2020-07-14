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
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

func TestGetRecordsByJetDropID_severalJds(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 2
	recordsInJetDropCount := 3
	recordsJDOne := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsInJetDropCount)
	recordsJDTwo := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsInJetDropCount)
	recordsJDThree := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsInJetDropCount)

	records := make([]*exporter.Record, 0)
	records = append(records, recordsJDOne...)
	records = append(records, recordsJDTwo...)
	records = append(records, recordsJDThree...)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	jds := make(map[string][]string, 0)
	var maxPn insolar.PulseNumber = 0
	for _, r := range records {
		jetID := converter.JetIDToString(r.Record.JetID)
		pn := r.Record.ID.Pulse()
		if maxPn.AsUint32() < pn.AsUint32() {
			maxPn = pn
		}
		jetDropID := fmt.Sprintf("%v:%v", jetID, pn.String())
		// get reference from record
		ref := r.Record.ID.String()
		if jds[jetDropID] == nil {
			jds[jetDropID] = []string{ref}
		} else {
			jds[jetDropID] = append(jds[jetDropID], ref)
		}
	}
	jdsCount := 3 * pulsesCount
	require.Len(t, jds, jdsCount)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{testutils.GenerateRecordInNextPulse(maxPn)}))
	ts.WaitRecordsCount(t, len(records), 2000)

	c := GetHTTPClient()
	for jd := range jds {
		response, err := c.JetDropRecords(t, jd, nil)
		require.NoError(t, err)

		require.Equal(t, int64(len(jds[jd])), response.Total)
		require.Len(t, response.Result, len(jds[jd]))
		for _, r := range response.Result {
			require.Contains(t, jds[jd], r.Reference)
		}

		require.Empty(t, response.Code)
		require.Empty(t, response.Message)
		require.Empty(t, response.Description)
		require.Empty(t, response.Link)
		require.Empty(t, response.ValidationFailures)
	}
}

func TestGetRecordsByJetDropID_oneJdCheckFields(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 1
	recordsInJetDropCount := 9
	records := testutils.GenerateObjectLifeline(pulsesCount, recordsInJetDropCount).StateRecords[0].Records
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))

	expResult := make(map[string]client.ObjectLifelineResponse200Result, len(records))
	var maxPn insolar.PulseNumber = 0
	var jetDropID string
	for i, r := range records {
		jetID := converter.JetIDToString(r.Record.JetID)
		pn := r.Record.ID.Pulse()
		jetDropID = fmt.Sprintf("%v:%v", jetID, pn.String())
		if maxPn.AsUint32() < pn.AsUint32() {
			maxPn = pn
		}

		objID := r.Record.ObjectID
		objRef := insolar.NewReference(objID)
		recordRef := r.Record.ID.String()
		expResult[recordRef] = client.ObjectLifelineResponse200Result{
			Reference:       recordRef,
			ObjectReference: objRef.String(),
			Type:            "state",
			PulseNumber:     int64(pn.AsUint32()),
			JetId:           jetID,
			JetDropId:       jetDropID,
			Index:           fmt.Sprintf("%v:%v", pn, i),
		}
	}

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{testutils.GenerateRecordInNextPulse(maxPn)}))
	ts.WaitRecordsCount(t, len(records), 2000)

	c := GetHTTPClient()
	response, err := c.JetDropRecords(t, jetDropID, nil)
	require.NoError(t, err)

	require.Equal(t, int64(len(records)), response.Total)
	require.Len(t, response.Result, len(records))
	for _, r := range response.Result {
		var expRecord client.ObjectLifelineResponse200Result
		var ok bool
		if expRecord, ok = expResult[r.Reference]; !ok {
			t.Fatalf("Got unexpected record with reference: %v", r.Reference)
		}
		require.Equal(t, expRecord.Reference, r.Reference)
		require.Equal(t, expRecord.ObjectReference, r.ObjectReference)
		require.Equal(t, expRecord.Type, r.Type)
		require.Equal(t, expRecord.PulseNumber, r.PulseNumber)
		require.Equal(t, expRecord.JetId, r.JetId)
		require.Equal(t, expRecord.JetDropId, r.JetDropId)
		require.Equal(t, expRecord.Index, r.Index)
		require.NotEmpty(t, r.Hash)
		require.NotEmpty(t, r.Timestamp)
	}
	require.Empty(t, response.Code)
	require.Empty(t, response.Message)
	require.Empty(t, response.Description)
	require.Empty(t, response.Link)
	require.Empty(t, response.ValidationFailures)
}
