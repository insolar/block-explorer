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
	ts.WaitRecordsCount(t, len(records), 2000)

	c := GetHTTPClient()

	t.Run("nonexistent JetDrop", func(t *testing.T) {
		t.Log("get records by jetdrops")
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
	})

	t.Run("nonexistent JetDrop", func(t *testing.T) {
		t.Log("")
		pn := gen.PulseNumber()
		jetID := converter.JetIDToString(testutils.GenerateUniqueJetID())
		val := fmt.Sprintf("%v:%v", jetID, pn)
		response, err := c.JetDropRecords(t, val, nil)
		require.NoError(t, err)
		require.Empty(t, response.Result)
		require.Empty(t, response.Total)
	})
	t.Run("value with star", func(t *testing.T) {
		t.Log("")
		val := "*:65538"
		response, err := c.JetDropRecords(t, val, nil)
		require.NoError(t, err)
		require.Empty(t, response.Result)
		require.Empty(t, response.Total)
	})
}

func TestGetRecordsByJetDropID_star(t *testing.T) {
	t.Log("")
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
	ts.WaitRecordsCount(t, recordsInJetDropCount*2, 5000)

	val := fmt.Sprintf("*:%v", pn.String())
	c := GetHTTPClient()
	response, err := c.JetDropRecords(t, val, nil)
	require.NoError(t, err)
	require.Len(t, response.Result, 2)
	require.Equal(t, int64(2), response.Total)
}

func TestGetRecordsByJetDropID_oneJdCheckFields(t *testing.T) {
	t.Log("")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 1
	recordsInJetDropCount := 9
	records := testutils.GenerateObjectLifeline(pulsesCount, recordsInJetDropCount).StateRecords[0].Records

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

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(records), func(i, j int) { records[i], records[j] = records[j], records[i] })

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
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
	require.Empty(t, response.Link)
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
		{"C5286 Search by zero value, get error", "0", badRequest400, "zero value"},
		{"C5287 Search by empty value, get error", "", badRequest400, "empty"},
		{"C5288 Search by random reference, get error", id.String(), badRequest400, "reference"},
		{"C5161 Search by jet_Id, get error", jetID, badRequest400, "jetID"},
		{"C5162 Search by invalid value, get error", invalidValue, badRequest400, "invalid value"},
		{"C5168 Search by value with 1k chars, get error", jetDropWithBigLengthPrefix, badRequest400, "big length jd pref"},
		{"C5289 Search by invalid jetdrop_id with very big pulse number, get error", jetDropWithBigLengthPulse, badRequest400, "big length jd pulse"},
		{"C5290 Search by very big number, get error", randomNumbers, badRequest400, "big number"},
		{"C5164 Search by nonexisting record_ref, get error", randomRecordRef, badRequest400, "random record ref"},
	}

	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			t.Log(tc.trTestCaseName)
			_, err := c.JetDropRecords(t, tc.value, nil)
			require.Error(t, err)
			require.Equal(t, tc.expResult, err.Error())
		})
	}
}
