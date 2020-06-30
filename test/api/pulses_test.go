// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"testing"

	"github.com/antihax/optional"
	"github.com/gogo/protobuf/sortkeys"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

func TestPulsesAPI(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 100
	recordsInPulse := 1
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)
	lastPulseRecord := testutils.GenerateRecordInNextPulse(lifeline.StateRecords[0].Pn)
	records := lifeline.GetAllRecords()
	pulses := make([]int64, pulsesNumber)
	for i, l := range lifeline.StateRecords {
		pulses[i] = int64(l.Pn.AsUint32())
	}
	sortkeys.Int64s(pulses)

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)
	err = heavymock.ImportRecords(ts.ConMngr.ImporterClient, []*exporter.Record{lastPulseRecord})
	require.NoError(t, err)

	ts.WaitRecordsCount(t, len(records), 10000)

	c := GetHTTPClient()

	t.Run("default query params", func(t *testing.T) {
		t.Log("T9341 Get pulses, default limit and offset")
		response, err := c.Pulses(t, nil)
		require.NoError(t, err)
		require.Len(t, response.Result, 20)
		require.Equal(t, pulsesNumber, int(response.Total))
		for _, res := range response.Result {
			require.Contains(t, pulses, res.PulseNumber)
			require.Equal(t, res.PulseNumber-10, res.PrevPulseNumber)
			require.Equal(t, res.PulseNumber+10, res.NextPulseNumber)
			require.Equal(t, recordsInPulse, int(res.JetDropAmount))
			require.Equal(t, recordsInPulse, int(res.RecordAmount))
			require.NotEmpty(t, res.Timestamp)
		}
	})
	t.Run("valid limit and offset", func(t *testing.T) {
		t.Log("C5209 Get pulses with limit and offset")
		offset, limit := 3, 3
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Len(t, response.Result, limit)
		require.Equal(t, pulses[len(pulses)-4], response.Result[0].PulseNumber)
		require.Equal(t, pulses[len(pulses)-5], response.Result[1].PulseNumber)
		require.Equal(t, pulses[len(pulses)-6], response.Result[2].PulseNumber)
	})
	t.Run("out of offset", func(t *testing.T) {
		t.Log("C5178 Get pulses, offset > max pulses")
		limit, offset := 10, pulsesNumber+1
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit)), Offset: optional.NewInt32(int32(offset))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, response.Total, int64(pulsesNumber))
		require.Len(t, response.Result, 0)
	})
	t.Run("limit min", func(t *testing.T) {
		t.Log("C5170 Success limit validation, limit = 1")
		limit := 1
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, response.Total, int64(pulsesNumber))
		require.Equal(t, pulses[len(pulses)-1], response.Result[0].PulseNumber)
		require.Len(t, response.Result, limit)
	})
	t.Run("limit max", func(t *testing.T) {
		t.Log("C5174 Success limit validation, limit = 100")
		limit := 100
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, response.Total, int64(pulsesNumber))
		require.Equal(t, pulses[len(pulses)-1], response.Result[0].PulseNumber)
		require.Len(t, response.Result, limit)
	})
	t.Run("zero limit", func(t *testing.T) {
		t.Log("C5172 Error limit validation, limit = 0")
		limit, offset := 0, 10
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		_, err := c.Pulses(t, &opts)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("too big limit", func(t *testing.T) {
		t.Log("C5173 Error limit validation, limit = 101")
		limit, offset := 101, 10
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		_, err := c.Pulses(t, &opts)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("negative limit", func(t *testing.T) {
		t.Log("C5210 Error limit validation, limit = -1")
		limit, offset := 10, -1
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		_, err := c.Pulses(t, &opts)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("offset min", func(t *testing.T) {
		t.Log("C5175 Success offset validation, offset = 1")
		limit, offset := 10, 1
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, response.Total, int64(pulsesNumber))
		require.Equal(t, pulses[len(pulses)-2], response.Result[0].PulseNumber)
		require.Len(t, response.Result, limit)
	})
	t.Run("offset zero", func(t *testing.T) {
		t.Log("C5212 Success offset validation, offset = 0")
		limit, offset := 10, 0
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, pulses[len(pulses)-1], response.Result[0].PulseNumber)
	})
	t.Run("offset negative", func(t *testing.T) {
		t.Log("C5177 Error offset validation, offset = -1")
		limit, offset := 10, -1
		opts := client.PulsesOpts{Offset: optional.NewInt32(int32(offset)), Limit: optional.NewInt32(int32(limit))}
		_, err := c.Pulses(t, &opts)
		require.Error(t, err)
		require.Equal(t, "400 Bad Request", err.Error())
	})
	t.Run("with FromPulseNumber", func(t *testing.T) {
		t.Log("")
		limit, fromPulse := 20, int(pulses[2])
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit)), FromPulseNumber: optional.NewInt64(int64(fromPulse))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, int64(3), response.Total)
		require.Len(t, response.Result, 3)
		require.Equal(t, int64(fromPulse), response.Result[0].PulseNumber)
	})
	t.Run("with FromPulseNumber reduced total", func(t *testing.T) {
		t.Log("C5213 Get pulses with parameter FromPulseNumber")
		l := len(pulses) - 2
		limit, fromPulse := 20, int(pulses[l])
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit)), FromPulseNumber: optional.NewInt64(int64(fromPulse))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, int64(l+1), response.Total)
		require.Len(t, response.Result, limit)
		require.Equal(t, int64(fromPulse), response.Result[0].PulseNumber)
	})
	t.Run("non existing FromPulseNumber", func(t *testing.T) {
		t.Log("C5214 Get pulses, non existing FromPulseNumber")
		limit, fromPulse := 20, pulses[0]-100
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit)), FromPulseNumber: optional.NewInt64(fromPulse)}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Len(t, response.Result, 0)
	})
	t.Run("FromPulseNumber value between existing pulses", func(t *testing.T) {
		t.Log("C5215 Get pulses, FromPulseNumber value between existing pulses")
		fromPulse := int(pulses[2])
		limit := 20
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit)), FromPulseNumber: optional.NewInt64(int64(fromPulse + 5))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, int64(3), response.Total)
		require.Len(t, response.Result, 3)
		require.Equal(t, int64(fromPulse), response.Result[0].PulseNumber)
	})
	t.Run("from TimestampGte", func(t *testing.T) {
		t.Log("C5216 Get pulses with parameter TimestampGte")
		limit, offset := 20, len(pulses)-10
		opts := client.PulsesOpts{Limit: optional.NewInt32(int32(limit)), Offset: optional.NewInt32(int32(offset))}
		response, err := c.Pulses(t, &opts)
		require.NoError(t, err)
		ts := response.Result[0].Timestamp

		opts = client.PulsesOpts{TimestampLte: optional.NewInt64(ts)}
		response, err = c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, int64(10), response.Total)
		require.Len(t, response.Result, 10)
		require.Equal(t, ts, response.Result[0].Timestamp)
	})
	t.Run("until TimestampLte", func(t *testing.T) {
		t.Log("C5217 Get pulses with parameter TimestampLte")
		response, err := c.Pulses(t, nil)
		require.NoError(t, err)
		ts := response.Result[10].Timestamp

		opts := client.PulsesOpts{TimestampGte: optional.NewInt64(ts)}
		response, err = c.Pulses(t, &opts)
		require.NoError(t, err)
		require.Equal(t, int64(11), response.Total)
		require.Len(t, response.Result, 11)
		require.Equal(t, ts, response.Result[10].Timestamp)
	})
}
