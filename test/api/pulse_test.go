// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"testing"

	"github.com/gogo/protobuf/sortkeys"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestOnePulse(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	size := 3
	records := testutils.GenerateRecordsWithDifferencePulsesSilence(size, 1)
	for _, r := range records {
		println(r.Record.ID.GetPulseNumber())
	}

	err := heavymock.ImportRecords(ts.ConMngr.ImporterClient, records)
	require.NoError(t, err)

	ts.WaitRecordsCount(t, size-1, 5000)

	// time.Sleep(3*time.Second)
	c := GetHTTPClient()
	_, err = c.Pulses(t, nil)

	_, err = c.Pulse(t, int64(records[1].Record.ID.GetPulseNumber().AsUint32()))
	require.NoError(t, err)
}

func TestPulseAPI(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pulsesNumber := 10
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

	ts.WaitRecordsCount(t, len(records), 1000)

	c := GetHTTPClient()

	t.Run("default query params", func(t *testing.T) {
		t.Log("T9341 Get pulses, default limit and offset")
		response, err := c.Pulse(t, pulses[1])
		require.NoError(t, err)
		require.Equal(t, response.PulseNumber-10, response.PrevPulseNumber)
		require.Equal(t, response.PulseNumber+10, response.NextPulseNumber)
		require.Equal(t, recordsInPulse, int(response.JetDropAmount))
		require.Equal(t, recordsInPulse, int(response.RecordAmount))
		require.NotEmpty(t, response.Timestamp)
	})
}
