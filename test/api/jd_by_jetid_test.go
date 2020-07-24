// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/antihax/optional"
	"github.com/gogo/protobuf/sortkeys"
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/jet"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

func TestGetJetDropsByJetID(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesInJet := 5
	recordsCount := 5
	recordsJetOne := testutils.GenerateRecordsFromOneJetSilence(pulsesInJet, recordsCount)
	recordsJetTwo := testutils.GenerateRecordsFromOneJetSilence(pulsesInJet, recordsCount)
	records := append(recordsJetOne, recordsJetTwo...)
	jetIDs := make(map[string][]string, 0)
	var maxPn insolar.PulseNumber = 0

	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	}

	for _, r := range records {
		pulse := r.Record.ID.GetPulseNumber()
		if maxPn < pulse {
			maxPn = pulse
		}
		jetID := converter.JetIDToString(r.Record.JetID)
		jetDropID := fmt.Sprintf("%v:%v", jetID, pulse.String())

		if jetDrops, ok := jetIDs[jetID]; ok {
			if !contains(jetDrops, jetDropID) {
				jetIDs[jetID] = append(jetIDs[jetID], jetDropID)
			}
		} else {
			jetIDs[jetID] = []string{jetDropID}
		}

	}
	require.Len(t, jetIDs, 2)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	recordInLastPulse := []*exporter.Record{testutils.GenerateRecordInNextPulse(maxPn)}
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, recordInLastPulse))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 5000)

	c := GetHTTPClient()
	t.Run("check jetDrops amount", func(t *testing.T) {
		t.Log("C5410 Get JetDrops by JetID if JetID contains JetDrops from different pulses")
		for expJetID := range jetIDs {
			response := c.JetDropsByJetID(t, expJetID, nil)
			require.Empty(t, response.Message)
			require.Empty(t, response.Code)
			require.Empty(t, response.Description)
			require.Empty(t, response.ValidationFailures)
			require.Equal(t, int64(pulsesInJet), response.Total)
			for _, jetDropResponse := range response.Result {
				if expJdIDs, ok := jetIDs[jetDropResponse.JetId]; ok {
					require.Len(t, response.Result, len(expJdIDs))
					require.Contains(t, expJdIDs, jetDropResponse.JetDropId)
					require.Equal(t, int64(recordsCount), jetDropResponse.RecordAmount)
					require.Equal(t, strings.Split(jetDropResponse.JetDropId, ":")[1], strconv.FormatInt(jetDropResponse.PulseNumber, 10))
					require.NotEmpty(t, jetDropResponse.Timestamp)
					require.NotEmpty(t, jetDropResponse.Hash)
				} else {
					t.Fatalf("Received unexpected JetID in response: %v", jetDropResponse.JetId)
				}
			}
		}
	})
	t.Run("check jetDrops amount", func(t *testing.T) {
		t.Log("C5421 Get JetDrops by JetID, if value is a starting numbers of existing JetID (get childs by parent)")
		var values []string
		for jetID := range jetIDs {
			values = append(values, jetID[:len(jetID)-int(math.Round(float64(len(jetID)/2)))])
		}
		for _, value := range values {
			response := c.JetDropsByJetID(t, value, nil)
			require.NotEmpty(t, response.Result)
			require.Greater(t, response.Total, int64(0))
			require.Empty(t, response.ValidationFailures)
			for _, res := range response.Result {
				require.True(t, strings.HasPrefix(res.JetId, value))
			}
		}
	})
	t.Run("check jetDrops amount", func(t *testing.T) {
		t.Log("C5422 Get JetDrops by nonexistent JetID")
		jetID := converter.JetIDToString(testutils.GenerateUniqueJetID())
		response := c.JetDropsByJetID(t, jetID, nil)
		require.Empty(t, response.Result)
		require.Equal(t, int64(0), response.Total)
		require.Empty(t, int64(0), response.ValidationFailures)
	})
}

func TestGetJetDropsByJetID_queryParams(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesInJet := 4
	recordsCount := 2
	records := testutils.GenerateRecordsFromOneJetSilence(pulsesInJet, recordsCount)

	contains := func(s []uint32, e uint32) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	}

	uniqPulses := make([]uint32, 0)
	var maxPn insolar.PulseNumber = 0
	for _, r := range records {
		pulse := r.Record.ID.GetPulseNumber()
		if pn := pulse.AsUint32(); !contains(uniqPulses, pn) {
			uniqPulses = append(uniqPulses, pn)
		}
		if maxPn < pulse {
			maxPn = pulse
		}
	}
	sortkeys.Uint32s(uniqPulses)
	jetID := converter.JetIDToString(records[0].Record.JetID)
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{testutils.GenerateRecordInNextPulse(maxPn)}))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 2000)

	c := GetHTTPClient()

	t.Run("limit", func(t *testing.T) {
		t.Log("C5423 Get JetDrops by JetID with limit")
		queryParams := client.JetDropsByJetIDOpts{
			Limit: optional.NewInt32(int32(pulsesInJet - 1)),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet), response.Total)
		require.Len(t, response.Result, pulsesInJet-1)
	})
	t.Run("SortBy pulse_number_asc,jet_id_desc", func(t *testing.T) {
		t.Log("C5424 Get JetDrops by JetID with SortBy=pulse_number_asc,jet_id_desc")
		queryParams := client.JetDropsByJetIDOpts{
			SortBy: optional.NewString("pulse_number_asc,jet_id_desc"),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet), response.Total)
		require.Len(t, response.Result, pulsesInJet)
		var pulses []int64
		for _, j := range response.Result {
			pulses = append(pulses, j.PulseNumber)
		}
		require.Len(t, pulses, pulsesInJet)
		for i := 0; i < pulsesInJet-1; i++ {
			require.Less(t, pulses[i], pulses[i+1])
		}
	})
	t.Run("SortBy pulse_number_desc,jet_id_asc", func(t *testing.T) {
		t.Log("C5425 Get JetDrops by JetID with SortBy=pulse_number_desc,jet_id_asc")
		queryParams := client.JetDropsByJetIDOpts{
			SortBy: optional.NewString("pulse_number_desc,jet_id_asc"),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet), response.Total)
		require.Len(t, response.Result, pulsesInJet)
		var pulses []int64
		for _, j := range response.Result {
			pulses = append(pulses, j.PulseNumber)
		}
		require.Len(t, pulses, pulsesInJet)
		for i := 0; i < pulsesInJet-1; i++ {
			require.Greater(t, pulses[i], pulses[i+1])
		}
	})
	t.Run("PulseNumberGte", func(t *testing.T) {
		t.Log("C5426 Get JetDrops by JetID with PulseNumberGte")
		pn := uniqPulses[1]
		queryParams := client.JetDropsByJetIDOpts{
			PulseNumberGte: optional.NewInt32(int32(pn)),
			SortBy:         optional.NewString("pulse_number_asc,jet_id_desc"),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet-1), response.Total)
		require.Len(t, response.Result, pulsesInJet-1)
		require.Equal(t, int64(pn), response.Result[0].PulseNumber)
		require.Less(t, response.Result[0].PulseNumber, response.Result[1].PulseNumber)
	})
	t.Run("PulseNumberGt", func(t *testing.T) {
		t.Log("C5428 Get JetDrops by JetID with PulseNumberGt")
		pn := uniqPulses[1]
		queryParams := client.JetDropsByJetIDOpts{
			PulseNumberGt: optional.NewInt32(int32(pn)),
			SortBy:        optional.NewString("pulse_number_asc,jet_id_desc"),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet-2), response.Total)
		require.Len(t, response.Result, pulsesInJet-2)
		require.Equal(t, int64(uniqPulses[2]), response.Result[0].PulseNumber)
		require.Less(t, response.Result[0].PulseNumber, response.Result[1].PulseNumber)
	})
	t.Run("PulseNumberLte", func(t *testing.T) {
		t.Log("C5427 Get JetDrops by JetID with PulseNumberLte")
		pn := uniqPulses[2]
		queryParams := client.JetDropsByJetIDOpts{
			PulseNumberLte: optional.NewInt32(int32(pn)),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet-1), response.Total)
		require.Len(t, response.Result, pulsesInJet-1)
		require.Equal(t, int64(pn), response.Result[0].PulseNumber)
		require.Greater(t, response.Result[0].PulseNumber, response.Result[1].PulseNumber)
	})
	t.Run("PulseNumberLt", func(t *testing.T) {
		t.Log("C5429 Get JetDrops by JetID with PulseNumberLt")
		pn := uniqPulses[2]
		queryParams := client.JetDropsByJetIDOpts{
			PulseNumberLt: optional.NewInt32(int32(pn)),
		}
		response := c.JetDropsByJetID(t, jetID, &queryParams)
		require.Equal(t, int64(pulsesInJet-2), response.Total)
		require.Len(t, response.Result, pulsesInJet-2)
		require.Equal(t, int64(uniqPulses[1]), response.Result[0].PulseNumber)
		require.Greater(t, response.Result[0].PulseNumber, response.Result[1].PulseNumber)
	})
}

func TestGetJetDropsByJetID_negative(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)
	c := GetHTTPClient()

	jetID := converter.JetIDToString(testutils.GenerateUniqueJetID())
	pn := gen.PulseNumber()
	jetDropID := fmt.Sprintf("%v:%v", jetID, pn.String())
	invalidValue := gen.RecordReference().String()
	jetIDBigLength := strings.Repeat(jetDropID, 50)
	randomNumbers := strconv.FormatInt(testutils.RandNumberOverRange(1, math.MaxInt32), 10)

	tcs := []testCases{
		{"C5430 Get JetDrops by JetID as empty value, get error", "", badRequest400, "empty"},
		{"C5431 Get JetDrops by JetID, get error if value is JetDropID", jetDropID, badRequest400, "JetDropID"},
		{"C5432 Get JetDrops by JetID as invalid value, get error", invalidValue, badRequest400, "invalid value"},
		{"C5433 Get JetDrops by JetID, get error if value is a big number", randomNumbers, badRequest400, "big length jd"},
		{"C5434 Get JetDrops by JetID as very big number of 1s and 0s, get error", jetIDBigLength, badRequest400, "big number"},
	}

	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			t.Log(tc.trTestCaseName)
			c.JetDropsByJetIDWithError(t, tc.value, nil, tc.expResult)
		})
	}
}

func TestGetJetDropsByJetID_emptyJetID(t *testing.T) {
	t.Log("C5457 Get JetDrops by JetID = '*', receive a list containing empty and not empty JetIDs")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount, recordsCount := 5, 2
	records := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsCount)

	pulses := make(map[int64]bool, pulsesCount)
	var maxPn pulse.Number = 0
	jetID := jet.NewIDFromString("")
	for _, r := range records {
		pn := r.Record.ID.Pulse()
		if maxPn < pn {
			maxPn = pn
		}
		pulses[int64(pn)] = false
		r.Record.JetID = jetID
	}
	recordWithNotEmptyJetID := testutils.GenerateRecordInNextPulse(maxPn)
	records = append(records, recordWithNotEmptyJetID)
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records), 5000)
	c := GetHTTPClient()

	res := c.JetDropsByJetID(t, "*", nil)
	jetDropsAmount := pulsesCount + 1
	require.Len(t, res.Result, jetDropsAmount)
	require.Equal(t, int64(jetDropsAmount), res.Total)
	for _, jd := range res.Result {
		if jd.JetId != "*" {
			require.Equal(t, converter.JetIDToString(recordWithNotEmptyJetID.Record.JetID), jd.JetId)
		} else {
			pulses[jd.PulseNumber] = true
			require.Equal(t, "*", jd.JetId)
			require.Equal(t, int64(recordsCount), jd.RecordAmount)
			require.Equal(t, fmt.Sprintf("*:%v", strconv.FormatInt(jd.PulseNumber, 10)), jd.JetDropId)
		}
	}
	for p := range pulses {
		require.True(t, pulses[p])
	}
}
