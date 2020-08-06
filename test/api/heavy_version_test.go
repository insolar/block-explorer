// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
)

func TestGBEVersion_Error(t *testing.T) {
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

	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)

	ts.StartBE(t)
	defer ts.StopBE(t)

	ts.WaitRecordsCount(t, len(records)+1, 5000)

	c := GetHTTPClient()
	pulsesResp := c.Pulses(t, nil)
	require.Len(t, pulsesResp.Result, pulsesCount+1)

	t.Run("check received data in jetdrops", func(t *testing.T) {
		t.Log("C5240 Get JetDrop by JetDropID")
		for jd := range jds {
			response := c.JetDropsByID(t, jd)
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
