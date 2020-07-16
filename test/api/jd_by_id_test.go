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
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"
)

func TestGetJetDropsByID(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 5
	jetDropsCount := 2
	records := testutils.GenerateRecordsWithDifferencePulsesSilence(pulsesCount, jetDropsCount)
	jds := make(map[string]pulse.Number, 0)
	for _, r := range records {
		pulse := r.Record.ID.GetPulseNumber()
		jetID := converter.JetIDToString(r.Record.JetID)
		jetDropID := fmt.Sprintf("%v:%v", jetID, pulse.String())
		jds[jetDropID] = pulse
	}
	require.Len(t, jds, pulsesCount*jetDropsCount)

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	lastPulse := records[len(records)-1].Record.ID.GetPulseNumber()
	recordInLastPulse := []*exporter.Record{testutils.GenerateRecordInNextPulse(lastPulse)}
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, recordInLastPulse))

	ts.WaitRecordsCount(t, len(records), 5000)

	c := GetHTTPClient()
	pulsesResp, err := c.Pulses(t, nil)
	require.NoError(t, err)
	require.Len(t, pulsesResp.Result, pulsesCount)

	t.Run("check received data in jetdrops", func(t *testing.T) {
		t.Log("C5240 Get JetDrop by JetDropID")
		for jd := range jds {
			response, err := c.JetDropsByID(t, jd)
			require.NoError(t, err)
			require.Equal(t, jd, response.JetDropId)
			require.Equal(t, int64(1), response.RecordAmount)
			require.Empty(t, response.Message)
			require.Empty(t, response.ValidationFailures)
			require.Equal(t, int64(jds[jd]), response.PulseNumber)
			require.Equal(t, strings.Split(jd, ":")[0], response.JetId)
			require.NotEmpty(t, response.Timestamp)
			require.NotEmpty(t, response.Hash)
		}
	})
}

func TestGetJetDropsByID_negativeCases(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 3
	jetDropsCount := 2
	records := testutils.GenerateRecordsWithDifferencePulsesSilence(pulsesCount, jetDropsCount)
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	ts.WaitRecordsCount(t, len(records)-jetDropsCount, 5000)
	c := GetHTTPClient()

	nonExistentJetID := fmt.Sprintf("%v:%v",
		converter.JetIDToString(testutils.GenerateUniqueJetID()),
		records[0].Record.ID.Pulse())
	jetID := strings.Split(nonExistentJetID, ":")[0]
	withWrongPulse := fmt.Sprintf("%v:%v",
		strings.Split(converter.JetIDToString(records[0].Record.JetID), ":")[0],
		records[2].Record.ID.Pulse())
	invalidJetDropID := "0qwerty123:!@#$%^"
	withBigLengthPrefix := fmt.Sprintf("%v:%v",
		strings.Repeat("01", 130),
		records[0].Record.ID.Pulse())
	withBigLengthPulse := fmt.Sprintf("%v:%v",
		strings.Split(converter.JetIDToString(records[0].Record.JetID), ":")[0],
		string(math.MaxInt64)+"1")
	randomNumbers := fmt.Sprintf("%v:%v",
		testutils.RandNumberOverRange(1, math.MaxInt32),
		testutils.RandNumberOverRange(1, math.MaxInt32))

	tcs := []testCases{
		{"C5242 Get JetDrop by JetDropID, not found non existing JetDropID ", nonExistentJetID, notFound404, "non existing JetDropID"},
		{"C5243 Get JetDrop by JetDropID, error if passed JetID", jetID, badRequest400, "JetDropID as JetID"},
		{"C5244 Get JetDrop by JetDropID, error if JetDropID format is [validJetDropID]:[wrongPulse]", withWrongPulse, notFound404, "wrong pulse"},
		{"C5245 Get JetDrop by JetDropID, error if JetDropID is invalid values separated by colon", invalidJetDropID, badRequest400, "invalid value"},
		{"C5246 Get JetDrop by JetDropID, error if JetID length > 217", withBigLengthPrefix, badRequest400, "too big prefix length"},
		{"C5247 Get JetDrop by JetDropID, error if pulse length > int64", withBigLengthPulse, badRequest400, "too big pulse length"},
		{"C5248 Get JetDrop by JetDropID, error if JetDropID = 0:0", "0:0", notFound404, "JetDropID = 0:0"},
		{"C5249 Get JetDrop by JetDropID, error if JetDropID = *", "*", badRequest400, "star"},
		{"C5251 Get JetDrop by JetDropID, if value is random numbers separated by colon", randomNumbers, badRequest400, "random number"},
	}

	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			t.Log(tc.trTestCaseName)
			_, err := c.JetDropsByID(t, tc.value)
			require.Error(t, err)
			require.Equal(t, tc.expResult, err.Error())
		})
	}
}
