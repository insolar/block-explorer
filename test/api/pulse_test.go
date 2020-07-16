// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"context"
	"errors"
	"testing"

	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/clients"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetPulse(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	size := 5
	recordsCount := 1
	records := testutils.GenerateRecordsWithDifferencePulsesSilence(size, recordsCount)
	pulses := make([]pulse.Number, size)
	for i, r := range records {
		pulses[i] = r.Record.ID.GetPulseNumber()
	}

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.BE.PulseClient.NextFinalizedPulseFunc = func(ctx context.Context, in *exporter.GetNextFinalizedPulse, opts ...grpc.CallOption) (*exporter.FullPulse, error) {
		p := uint32(ts.ConMngr.Importer.GetLowestUnsentPulse())
		if p == 1<<32-1 {
			return nil, errors.New("unready yet")
		}
		return clients.GetFullPulse(p), nil
	}

	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, size, 5000)

	c := GetHTTPClient()
	pulsesResp, err := c.Pulses(t, nil)
	require.Len(t, pulsesResp.Result, size)

	t.Run("existing pulses", func(t *testing.T) {
		t.Log("C5218 Get pulse data")
		for i, p := range pulses[:len(pulses)-1] {
			response, err := c.Pulse(t, int64(p))
			require.NoError(t, err)
			require.Equal(t, pulsesResp.Result[len(pulsesResp.Result)-1-i].PulseNumber, response.PulseNumber)
			require.Equal(t, int64(p), response.PulseNumber)
			require.Equal(t, response.PulseNumber-10, response.PrevPulseNumber)
			require.Equal(t, response.PulseNumber+10, response.NextPulseNumber)
			require.Equal(t, recordsCount, int(response.JetDropAmount))
			require.Equal(t, recordsCount, int(response.RecordAmount))
			require.NotEmpty(t, response.Timestamp)
			require.Empty(t, response.Message)
			require.Empty(t, response.ValidationFailures)
		}
	})
	t.Run("non existing pulse", func(t *testing.T) {
		t.Log("C5219 Get pulse, not found non existing pulse")
		_, err := c.Pulse(t, int64(pulses[len(pulses)-1]+1000))
		require.Error(t, err)
		require.Equal(t, "404 Not Found", err.Error())
	})
	t.Run("zero pulse", func(t *testing.T) {
		t.Log("C5221 Get pulse, pulse is zero value")
		_, err := c.Pulse(t, 0)
		require.Error(t, err)
		require.Equal(t, "404 Not Found", err.Error())
	})
	t.Run("empty pulse", func(t *testing.T) {
		t.Log("C5222 Get pulse, pulse is an empty pulse")
		t.Skip("waiting for PENV-347")
		newRecords := []*exporter.Record{testutils.GenerateRecordInNextPulse(pulses[size-1]),
			testutils.GenerateRecordInNextPulse(pulses[size-1] + 10),
			testutils.GenerateRecordInNextPulse(pulses[size-1] + 20)}

		err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, newRecords[1:])
		ts.WaitRecordsCount(t, size+1, 5000)

		emptyPulse := int64(newRecords[0].Record.ID.Pulse())
		// TODO check if emptyPulse exists or not in the pulses list
		_, err = c.Pulses(t, nil)
		require.NoError(t, err)

		r, err := c.Pulse(t, emptyPulse)
		require.Equal(t, emptyPulse, r.PulseNumber)
	})
}
