// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"testing"

	"github.com/antihax/optional"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

func TestLifeline_onePulse(t *testing.T) {
	t.Log("C4993 Receive object lifeline, states belong to one pulse")
	ts := integration.NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)

	pulsesNumber := 1
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)

	lastPulseRecord := testutils.GenerateRecordsSilence(1)[0]
	lastPulseRecord.Record.ID = gen.IDWithPulse(lifeline.States[0].Pn + 10)
	lastPulseRecord.ShouldIterateFrom = nil

	lifeline.States[0].Records = append(lifeline.States[0].Records, lastPulseRecord)

	err := heavymock.ImportRecords(ts.C.ImporterClient, lifeline.States[0].Records)
	require.NoError(t, err)

	stateRecordsCount := pulsesNumber * recordsInPulse
	totalRecords := stateRecordsCount + 2
	ts.WaitRecordsCount(t, totalRecords, 600)

	c := NewBeApiClient(t, fmt.Sprintf("http://localhost%v", connectionmanager.DefaultApiPort))
	response, err := c.ObjectLifeline(lifeline.ObjID.String(), nil)
	require.NoError(t, err)
	require.Len(t, response.Result, stateRecordsCount)
	for _, res := range response.Result {
		require.Contains(t, lifeline.ObjID.String(), res.ObjectReference)
		require.Equal(t, int64(lifeline.States[0].Pn), res.PulseNumber)
	}
}

func TestLifeline_severalPulses(t *testing.T) {
	t.Log("C4994 Receive object lifeline, states belong to several pulses")
	ts := integration.NewBlockExplorerTestSetup(t)
	defer ts.Stop(t)

	pulsesNumber := 4
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)

	lastPulseRecord := testutils.GenerateRecordsSilence(1)[0]
	lastPulseRecord.Record.ID = gen.IDWithPulse(lifeline.States[pulsesNumber-1].Pn + 10)
	lastPulseRecord.ShouldIterateFrom = nil

	records := make([]*exporter.Record, 0)
	for _, state := range lifeline.States {
		records = append(records, state.Records...)
	}
	records = append(records, lastPulseRecord)
	err := heavymock.ImportRecords(ts.C.ImporterClient, records)
	require.NoError(t, err)

	stateRecordsCount := pulsesNumber * recordsInPulse
	totalRecords := stateRecordsCount + 2
	ts.WaitRecordsCount(t, totalRecords, 600)

	c := NewBeApiClient(t, fmt.Sprintf("http://localhost%v", connectionmanager.DefaultApiPort))
	response, err := c.ObjectLifeline(lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(t, err)
	require.Len(t, response.Result, stateRecordsCount)
	pulses := make([]int64, pulsesNumber)
	for i, s := range lifeline.States {
		pulses[i] = int64(s.Pn)
	}
	for _, res := range response.Result {
		require.Contains(t, lifeline.ObjID.String(), res.ObjectReference)
		require.Contains(t, pulses, res.PulseNumber)
	}
}
