// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/antihax/optional"
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/jet"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

const (
	requestType = "request"
	stateType   = "state"
	resultType  = "result"
)

func TestGetRecordsByJetDropID(t *testing.T) {
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

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 2000)

	c := GetHTTPClient()

	t.Run("get records by jetdrops", func(t *testing.T) {
		t.Log("C5323 Get records by different JetDropIDs")
		for jd := range jds {
			response := c.JetDropRecords(t, jd, nil)

			require.Equal(t, int64(len(jds[jd])), response.Total)
			require.Len(t, response.Result, len(jds[jd]))
			for _, r := range response.Result {
				require.Contains(t, jds[jd], r.Reference)
			}

			require.Empty(t, response.Code)
			require.Empty(t, response.Message)
			require.Empty(t, response.Description)
			require.Empty(t, response.ValidationFailures)
		}
	})
	t.Run("nonexistent JetDropID", func(t *testing.T) {
		t.Log("C5324 Get records by JetDropID if JetDropID is nonexistent")
		pn := gen.PulseNumber()
		jetID := converter.JetIDToString(testutils.GenerateUniqueJetID())
		val := fmt.Sprintf("%v:%v", jetID, pn)
		response := c.JetDropRecords(t, val, nil)
		require.Empty(t, response.Result)
		require.Empty(t, response.Total)
	})
	t.Run("value with star", func(t *testing.T) {
		t.Log("C5325 Get records by JetDropID, no results if \"*:pulse\"")
		val := "*:65538"
		response := c.JetDropRecords(t, val, nil)
		require.Empty(t, response.Result)
		require.Empty(t, response.Total)
	})
}

func TestGetRecordsByJetDropID_queryParams(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 1
	recordsInJetDropCount := 22
	lifeline := testutils.GenerateObjectLifeline(pulsesCount, recordsInJetDropCount)
	records := lifeline.StateRecords[0].Records

	jetID := records[0].Record.JetID
	jetIDstr := converter.JetIDToString(jetID)
	pn := records[0].Record.ID.Pulse()
	jetDropID := fmt.Sprintf("%v:%v", jetIDstr, pn.String())

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{testutils.GenerateRecordInNextPulse(records[0].Record.ID.Pulse())}))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 5000)

	c := GetHTTPClient()

	t.Run("limit offset", func(t *testing.T) {
		t.Log("C5326 Get records by JetDropID with limit and offset set")
		queryParams := client.JetDropRecordsOpts{
			Limit:  optional.NewInt32(int32(recordsInJetDropCount - 2)),
			Offset: optional.NewInt32(int32(1)),
		}
		response := c.JetDropRecords(t, jetDropID, &queryParams)
		require.Equal(t, int64(recordsInJetDropCount), response.Total)
		require.Len(t, response.Result, recordsInJetDropCount-2)
		require.Equal(t, records[1].Record.ID.String(), response.Result[0].Reference)
	})
	t.Run("FromIndex", func(t *testing.T) {
		t.Log("5327 Get records by JetDropID with FromIndex set")
		fromIdx := 5
		queryParams := client.JetDropRecordsOpts{
			FromIndex: optional.NewString(fmt.Sprintf("%v:%v", lifeline.StateRecords[0].Pn, fromIdx)),
		}
		response := c.JetDropRecords(t, jetDropID, &queryParams)
		require.Equal(t, int64(recordsInJetDropCount-fromIdx), response.Total)
		require.Len(t, response.Result, recordsInJetDropCount-fromIdx)
		require.Equal(t, records[fromIdx].Record.ID.String(), response.Result[0].Reference)
	})
}

func TestGetRecordsByJetDropID_byType(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 1
	recordsInJetDropCount := 1
	lifeline := testutils.GenerateObjectLifeline(pulsesCount, recordsInJetDropCount)
	records := lifeline.StateRecords[0].Records

	jetID := records[0].Record.JetID
	jetIDstr := converter.JetIDToString(jetID)
	pn := records[0].Record.ID.Pulse()
	jetDropID := fmt.Sprintf("%v:%v", jetIDstr, pn.String())
	objID := records[0].Record.ObjectID

	requestRecord := testutils.GenerateVirtualRequestRecord(pn, objID)
	requestRecord.Record.JetID = jetID
	resultRecord := testutils.GenerateVirtualResultRecord(pn, objID, gen.ID())
	resultRecord.Record.JetID = jetID

	records = append(records, requestRecord)
	records = append(records, resultRecord)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{testutils.GenerateRecordInNextPulse(records[0].Record.ID.Pulse())}))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 5000)

	c := GetHTTPClient()
	t.Run("Type state", func(t *testing.T) {
		t.Log("C5328 Get records by JetDropID with Type set to State")
		queryParams := client.JetDropRecordsOpts{
			Type_: optional.NewString(stateType),
			Limit: optional.NewInt32(int32(recordsInJetDropCount)),
		}
		response := c.JetDropRecords(t, jetDropID, &queryParams)
		require.Equal(t, int64(recordsInJetDropCount), response.Total)
		require.Len(t, response.Result, recordsInJetDropCount)
		require.Equal(t, records[0].Record.ID.String(), response.Result[0].Reference)
		require.Equal(t, stateType, response.Result[0].Type)
	})
	t.Run("Type request", func(t *testing.T) {
		t.Log("C5329 Get records by JetDropID with Type set to Request")
		queryParams := client.JetDropRecordsOpts{
			Type_: optional.NewString(requestType),
			Limit: optional.NewInt32(int32(recordsInJetDropCount)),
		}
		response := c.JetDropRecords(t, jetDropID, &queryParams)
		require.Equal(t, int64(recordsInJetDropCount), response.Total)
		require.Len(t, response.Result, recordsInJetDropCount)
		require.Equal(t, requestRecord.Record.ID.String(), response.Result[0].Reference)
		require.Equal(t, requestType, response.Result[0].Type)
	})
	t.Run("Type result", func(t *testing.T) {
		t.Log("C5330 Get records by JetDropID with Type set to Result")
		queryParams := client.JetDropRecordsOpts{
			Type_: optional.NewString(resultType),
			Limit: optional.NewInt32(int32(recordsInJetDropCount)),
		}
		response := c.JetDropRecords(t, jetDropID, &queryParams)
		require.Equal(t, int64(recordsInJetDropCount), response.Total)
		require.Len(t, response.Result, recordsInJetDropCount)
		require.Equal(t, resultRecord.Record.ID.String(), response.Result[0].Reference)
		require.Equal(t, resultType, response.Result[0].Type)
	})
}

func TestGetRecordsByJetDropID_star(t *testing.T) {
	t.Log("C5331 Get records by JetDropID, get genesis records by a star char")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount, recordsInJetDropCount := 1, 9
	records := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsInJetDropCount)
	for i := 2; i < 4; i++ {
		records[i].Record.JetID = jet.NewIDFromString("")
	}
	recordsNextPulse := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsInJetDropCount)
	pn := records[0].Record.ID.Pulse()
	nexPn := pn + 10
	for _, r := range recordsNextPulse {
		r.Record.ID = gen.IDWithPulse(nexPn)
		r.Record.JetID = jet.NewIDFromString("")
	}
	nextRecord := testutils.GenerateRecordInNextPulse(nexPn)
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, recordsNextPulse))
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{nextRecord}))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, recordsInJetDropCount*2+1, 5000)

	val := fmt.Sprintf("*:%v", pn.String())
	c := GetHTTPClient()
	response := c.JetDropRecords(t, val, nil)
	require.Len(t, response.Result, 2)
	require.Equal(t, int64(2), response.Total)
}

func TestGetRecordsByJetDropID_oneJdCheckFields(t *testing.T) {
	t.Log("C5332 Get records by JetDropIDs and verify all fields")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 1
	recordsInJetDropCount := 9
	records := testutils.GenerateObjectLifeline(pulsesCount, recordsInJetDropCount).StateRecords[0].Records

	expResult := make(map[string]client.RecordsResponse200Result, len(records))
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
		expResult[recordRef] = client.RecordsResponse200Result{
			Reference:       recordRef,
			ObjectReference: objRef.String(),
			Type:            "state",
			PulseNumber:     int64(pn.AsUint32()),
			JetId:           jetID,
			JetDropId:       jetDropID,
			Index:           fmt.Sprintf("%v:%v", pn, i),
		}
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(records), func(i, j int) { records[i], records[j] = records[j], records[i] })

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{testutils.GenerateRecordInNextPulse(maxPn)}))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 5000)

	c := GetHTTPClient()
	response := c.JetDropRecords(t, jetDropID, nil)

	require.Equal(t, int64(len(records)), response.Total)
	require.Len(t, response.Result, len(records))
	for _, r := range response.Result {
		var expRecord client.RecordsResponse200Result
		var ok bool
		if expRecord, ok = expResult[r.Reference]; !ok {
			t.Fatalf("Not found record in response, reference: %v", r.Reference)
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
	require.Empty(t, response.ValidationFailures)
}

func TestGetRecordsByJetDropID_negative(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)
	c := GetHTTPClient()

	id := gen.ID()
	jetID := converter.JetIDToString(testutils.GenerateUniqueJetID())
	pn := gen.PulseNumber()
	jetDropID := fmt.Sprintf("%v:%v", jetID, pn.String())
	invalidValue := "0qwerty123:!@:#$%^"
	jetDropWithBigLengthPrefix := fmt.Sprintf("%v:%v", strings.Repeat(jetDropID, 20), pn)
	jetDropWithBigLengthPulse := fmt.Sprintf("%v:%v", jetDropID, string(math.MaxInt64)+"1")
	randomNumbers := fmt.Sprintf("%v:%v",
		testutils.RandNumberOverRange(1, math.MaxInt32),
		testutils.RandNumberOverRange(1, math.MaxInt32))
	randomRecordRef := gen.RecordReference().String()

	tcs := []testCases{
		{"C5333 Get records by JetDropID as zero value, get error", "0", badRequest400, "zero value"},
		{"C5334 Get records by JetDropID as empty value, get error", "", badRequest400, "empty"},
		{"C5335 Get records by JetDropID as random reference, get error", id.String(), badRequest400, "reference"},
		{"C5336 Get records by JetDropID as jet_Id, get error", jetID, badRequest400, "jetID"},
		{"C5337 Get records by JetDropID as invalid value, get error", invalidValue, badRequest400, "invalid value"},
		{"C5338 Get records by JetDropID as value with 1k chars, get error", jetDropWithBigLengthPrefix, badRequest400, "big length jd pref"},
		{"C5339 Get records by JetDropID as invalid jetdrop_id with very big pulse number, get error", jetDropWithBigLengthPulse, badRequest400, "big length jd pulse"},
		{"C5340 Get records by JetDropID as very big number, get error", randomNumbers, badRequest400, "big number"},
		{"C5341 Get records by JetDropID as nonexisting record_ref, get error", randomRecordRef, badRequest400, "random record ref"},
	}

	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			t.Log(tc.trTestCaseName)
			c.JetDropRecordsWithError(t, tc.value, nil, tc.expResult)
		})
	}
}
