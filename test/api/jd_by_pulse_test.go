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
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

const defaultLimit = 20

func TestGetJetDropsByPulse(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 5
	jetDropsCount := 2
	records := testutils.GenerateRecordsWithDifferencePulsesSilence(pulsesCount, jetDropsCount)
	pulses := make(map[pulse.Number]bool, pulsesCount)
	jds := make(map[string]bool, pulsesCount*2)
	for _, r := range records {
		pulses[r.Record.ID.GetPulseNumber()] = false
		jds[converter.JetIDToString(r.Record.JetID)] = false
	}

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	lastPulse := records[len(records)-1].Record.ID.GetPulseNumber()
	lastRecordInPulse := []*exporter.Record{testutils.GenerateRecordInNextPulse(lastPulse)}
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, lastRecordInPulse))

	ts.WaitRecordsCount(t, len(records), 5000)

	c := GetHTTPClient()
	pulsesResp, err := c.Pulses(t, nil)
	require.NoError(t, err)
	require.Len(t, pulsesResp.Result, pulsesCount)
	var i int

	t.Run("check received data in jetdrops", func(t *testing.T) {
		t.Log("C5223 Get Jet drops by Pulse number")
		for p := range pulses {
			response, err := c.JetDropsByPulseNumber(t, int64(p), nil)
			require.NoError(t, err)
			require.Equal(t, int64(jetDropsCount), response.Total)
			require.Len(t, response.Result, jetDropsCount)
			require.Empty(t, response.Message)
			require.Empty(t, response.ValidationFailures)
			for _, jd := range response.Result {
				require.Equal(t, int64(p), jd.PulseNumber)
				require.Equal(t, int64(1), jd.RecordAmount)
				require.NotEmpty(t, jd.Timestamp)
				require.NotEmpty(t, jd.Hash)
				require.Equal(t, jd.JetDropId, fmt.Sprintf("%v:%v", jd.JetId, jd.PulseNumber))
				// fill expected map with received values, then check outside the loop
				jds[jd.JetId] = true
			}
			i++
		}
		result := make([]string, 0)
		for k, v := range jds {
			if !v {
				result = append(result, k)
			}
		}
		require.Empty(t, result, "Followed JetIDs not found in responses: %v", result)
	})
	t.Run("not found", func(t *testing.T) {
		t.Log("C5225 Get Jet drops by Pulse number, error if non existing pulse")
		response, err := c.JetDropsByPulseNumber(t, int64(lastPulse+10000), nil)
		require.NoError(t, err)
		require.Empty(t, response.Result)
		require.Equal(t, int64(0), response.Total)
	})
	t.Run("invalid pulse", func(t *testing.T) {
		t.Log("C5224 Get Jet drops by Pulse number, error if invalid pulse ")
		_, err := c.JetDropsByPulseNumber(t, math.MaxInt64, nil)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("pulse zero", func(t *testing.T) {
		t.Log("C5226 Get Jet drops by Pulse number, error if pulse is zero value")
		_, err := c.JetDropsByPulseNumber(t, 0, nil)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("empty pulse", func(t *testing.T) {
		t.Log("C5227 Get Jet drops by Pulse number, pulse is an empty pulse")
		newRecords := []*exporter.Record{testutils.GenerateRecordInNextPulse(lastPulse + 10),
			testutils.GenerateRecordInNextPulse(lastPulse + 20),
			testutils.GenerateRecordInNextPulse(lastPulse + 30)}

		require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, newRecords[1:]))
		ts.WaitRecordsCount(t, len(records)+2, 5000)

		emptyPulse := int64(newRecords[0].Record.ID.Pulse())

		resp, err := c.JetDropsByPulseNumber(t, emptyPulse, nil)
		require.NoError(t, err)
		require.Equal(t, int64(0), resp.Total)
		require.Empty(t, resp.Result)
	})
}

func TestGetJetDropsByPulse_severalRecordsInJD(t *testing.T) {
	t.Log("C5236 Get Jet drops by Pulse number, JetDrop contains several records")
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount, recordsCount := 2, 10
	records := testutils.GenerateRecordsFromOneJetSilence(pulsesCount, recordsCount)
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))

	ts.WaitRecordsCount(t, recordsCount*(pulsesCount-1), 1000)
	c := GetHTTPClient()
	response, err := c.JetDropsByPulseNumber(t, int64(records[0].Record.ID.Pulse()), nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), response.Total)
	require.Len(t, response.Result, 1)
	require.Equal(t, int64(recordsCount), response.Result[0].RecordAmount)
}

func TestGetJetDropsByPulse_queryParams(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesCount := 1
	jetDropsCount := 101
	records := testutils.GenerateRecordsWithDifferencePulsesSilence(pulsesCount, jetDropsCount)
	pn := records[0].Record.ID.GetPulseNumber()

	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	nextPulseRecords := []*exporter.Record{testutils.GenerateRecordInNextPulse(pn)}
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, nextPulseRecords))

	ts.WaitRecordsCount(t, len(records), 10000)
	c := GetHTTPClient()
	jdList, err := c.JetDropsByPulseNumber(t, int64(pn), &client.JetDropsByPulseNumberOpts{
		Limit: optional.NewInt32(int32(jetDropsCount)),
	})
	require.NoError(t, err)
	require.Len(t, jdList.Result, jetDropsCount)

	t.Run("default params", func(t *testing.T) {
		t.Log("C5228 Get Jet drops by Pulse number with default params")
		response, err := c.JetDropsByPulseNumber(t, int64(pn), nil)
		require.NoError(t, err)
		require.Len(t, response.Result, defaultLimit)
		require.Equal(t, int64(jetDropsCount), response.Total)
	})
	t.Run("all possible params", func(t *testing.T) {
		t.Log("C5229 Get Jet drops by Pulse number with all possible params")
		fromJdIdx := 1
		queryParams := client.JetDropsByPulseNumberOpts{
			Offset:        optional.NewInt32(int32(10)),
			Limit:         optional.NewInt32(int32(10)),
			FromJetDropId: optional.NewString(jdList.Result[fromJdIdx].JetDropId),
		}
		response, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.NoError(t, err)
		require.Len(t, response.Result, 10)
		require.Equal(t, int64(jetDropsCount-fromJdIdx), response.Total)
	})
	t.Run("offset 1", func(t *testing.T) {
		t.Log("C5230 Get Jet drops by Pulse number with offset = 1")
		offset := 1
		queryParams := client.JetDropsByPulseNumberOpts{
			Offset: optional.NewInt32(int32(offset)),
		}
		response, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.NoError(t, err)
		require.Len(t, response.Result, defaultLimit)
		require.Equal(t, int64(jetDropsCount), response.Total)
		require.Equal(t, jdList.Result[1], response.Result[0])
	})
	t.Run("offset out of range", func(t *testing.T) {
		t.Log("C5231 Get Jet drops by Pulse number with offset out of range")
		offset := jetDropsCount
		queryParams := client.JetDropsByPulseNumberOpts{
			Offset: optional.NewInt32(int32(offset)),
		}
		response, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.NoError(t, err)
		require.Len(t, response.Result, 0)
	})
	t.Run("with FromJetDropId and Offset", func(t *testing.T) {
		t.Log("C5232 Get Jet drops by Pulse number with FromJetDropId and Offset")
		fromJdIdx, offset := 10, 10
		queryParams := client.JetDropsByPulseNumberOpts{
			FromJetDropId: optional.NewString(jdList.Result[fromJdIdx].JetDropId),
			Offset:        optional.NewInt32(int32(offset)),
		}
		response, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.NoError(t, err)
		require.Equal(t, int64(jetDropsCount-fromJdIdx), response.Total)
		require.Equal(t, jdList.Result[fromJdIdx+offset], response.Result[0])
	})
	t.Run("FromJetDropId invalid", func(t *testing.T) {
		t.Log("C5233 Get Jet drops by Pulse number with invalid FromJetDropId")
		queryParams := client.JetDropsByPulseNumberOpts{
			FromJetDropId: optional.NewString("%^&Qwerty!@#$%123"),
		}
		_, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("FromJetDropId empty", func(t *testing.T) {
		t.Log("C5234 Get Jet drops by Pulse number with empty FromJetDropId")
		queryParams := client.JetDropsByPulseNumberOpts{
			FromJetDropId: optional.NewString(""),
		}
		_, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("FromJetDropId too big", func(t *testing.T) {
		t.Log("C5235 Get Jet drops by Pulse number with too big FromJetDropId")

		s := strconv.FormatInt(testutils.RandNumberOverRange(math.MaxInt32, math.MaxInt32+1), 10)
		s = strings.Repeat(s, 100)
		queryParams := client.JetDropsByPulseNumberOpts{
			FromJetDropId: optional.NewString(s),
		}
		_, err := c.JetDropsByPulseNumber(t, int64(pn), &queryParams)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
}
