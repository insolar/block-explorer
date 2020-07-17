// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

const (
	typePulse   = "pulse"
	typeJetDrop = "jet-drop"
	typeRef     = "lifeline"
	typeRecord  = "record"
)

func TestSearchApi(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount, recordsCount := 3, 2
	lifeline := testutils.GenerateObjectLifeline(pulsesCount, recordsCount)
	records := lifeline.GetAllRecords()

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	ts.WaitRecordsCount(t, len(records), 5000)
	c := GetHTTPClient()

	record := lifeline.GetStateRecords()[0]
	pn := record.Record.ID.Pulse()
	jetID := converter.JetIDToString(record.Record.JetID)
	jetDropID := fmt.Sprintf("%v:%v", jetID, pn.String())
	objRef := insolar.NewReference(lifeline.ObjID).String()

	t.Run("get pulse", func(t *testing.T) {
		t.Log("C5157 Search by existing pulse_number")
		response, err := c.Search(t, pn.String())
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typePulse,
			Meta: client.SearchResponse200Meta{
				PulseNumber: int64(pn.AsUint32()),
			},
		}
		require.Equal(t, exp, response)
	})
	t.Run("get random pulse", func(t *testing.T) {
		t.Log("C5163 Search by nonexistent pulse_number")
		wrongPulse := lifeline.GetStateRecords()[0].Record.ID.Pulse() + 1000
		response, err := c.Search(t, wrongPulse.String())
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typePulse,
			Meta: client.SearchResponse200Meta{
				PulseNumber: int64(wrongPulse.AsUint32()),
			},
		}
		require.Equal(t, exp, response)
	})
	t.Run("get jetDrop", func(t *testing.T) {
		t.Log("C5159 Search by existing jetdrop_id")
		response, err := c.Search(t, jetDropID)
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typeJetDrop,
			Meta: client.SearchResponse200Meta{
				JetDropId: jetDropID,
			},
		}
		require.Equal(t, exp, response)
	})
	t.Run("get nonexisting jetDrop", func(t *testing.T) {
		t.Log("C5165 Search by nonexisting jetdrop_id")
		jetDrop := fmt.Sprintf("%v:%v",
			strings.Split(converter.JetIDToString(records[0].Record.JetID), ":")[0],
			records[2].Record.ID.Pulse())
		response, err := c.Search(t, jetDrop)
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typeJetDrop,
			Meta: client.SearchResponse200Meta{
				JetDropId: jetDrop,
			},
		}
		require.Equal(t, exp, response)
	})
	t.Run("get object ref", func(t *testing.T) {
		t.Log("C5160 Search by existing object_reference")
		response, err := c.Search(t, objRef)
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typeRef,
			Meta: client.SearchResponse200Meta{
				ObjectReference: objRef,
			},
		}
		require.Equal(t, exp, response)
	})
	t.Run("get random object ref", func(t *testing.T) {
		t.Log("C5166 Search by nonexisting object_reference")
		randomRef := gen.Reference().String()
		response, err := c.Search(t, randomRef)
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typeRef,
			Meta: client.SearchResponse200Meta{
				ObjectReference: randomRef,
			},
		}
		require.Equal(t, exp, response)
	})
	t.Run("get record", func(t *testing.T) {
		t.Log("C5158 Search by existing record_ref")
		objRef := insolar.NewReference(lifeline.ObjID).String()
		r := lifeline.GetStateRecords()[0]
		id := r.Record.ID.Bytes()
		ref := insolar.NewRecordReference(*insolar.NewIDFromBytes(id)).String()
		pn := r.Record.ID.Pulse()

		response, err := c.Search(t, ref)
		require.NoError(t, err)
		exp := client.SearchResponse200{
			Type: typeRecord,
			Meta: client.SearchResponse200Meta{
				ObjectReference: objRef,
				Index:           fmt.Sprintf("%v:%v", pn, "0"),
			},
		}
		require.Equal(t, exp, response)
	})

	id := lifeline.ObjID
	invalidValue := "0qwerty123:!@:#$%^"
	jetDropWithBigLengthPrefix := fmt.Sprintf("%v:%v",
		strings.Repeat(jetDropID, 20),
		records[0].Record.ID.Pulse())
	jetDropWithBigLengthPulse := fmt.Sprintf("%v:%v",
		strings.Split(converter.JetIDToString(records[0].Record.JetID), ":")[0],
		string(math.MaxInt64)+"1")
	randomNumbers := fmt.Sprintf("%v:%v",
		testutils.RandNumberOverRange(1, math.MaxInt32),
		testutils.RandNumberOverRange(1, math.MaxInt32))
	randomRecordRef := gen.RecordReference().String()

	tcs := []testCases{
		{"C5286 Search by zero value", "0", badRequest400, "zero value"},
		{"C5287 Search by empty value", "", badRequest400, "empty"},
		{"C5288 Search by random reference", id.String(), badRequest400, "reference"},
		{"C5161 Search by existing jet_Id, get error", jetID, badRequest400, "jetID"},
		{"C5162 Search by invalid value", invalidValue, badRequest400, "invalid value"},
		{"C5168 Search by value with 1k chars", jetDropWithBigLengthPrefix, badRequest400, "big length jd pref"},
		{"C5289 Search by invalid jetdrop_id with very big pulse number, get error", jetDropWithBigLengthPulse, badRequest400, "big length jd pulse"},
		{"C5290 Search by very big number, get error", randomNumbers, badRequest400, "big number"},
		{"C5164 Search by nonexisting record_ref, get error", randomRecordRef, badRequest400, "random record ref"},
	}

	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			t.Log(tc.trTestCaseName)
			_, err := c.Search(t, tc.value)
			require.Error(t, err)
			require.Equal(t, tc.expResult, err.Error())
		})
	}
}
