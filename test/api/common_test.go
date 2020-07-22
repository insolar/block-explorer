// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/test/integration"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/applicationbase/genesisrefs"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"
)

func TestRecordsWithoutObjectID(t *testing.T) {
	ts := integration.NewBlockExplorerTestSetup(t).WithHTTPServer(t)
	defer ts.Stop(t)

	pn, err := insolar.NewPulseNumberFromStr(strconv.FormatInt(pulse.MinTimePulse, 10))
	require.NoError(t, err)
	emptyObjID := *insolar.NewEmptyID()
	jetID := testutils.GenerateUniqueJetID()

	request := testutils.GenerateRequestRecord(pn, emptyObjID)
	request.Record.JetID = jetID
	virtualReq := request.Record.Virtual
	method := virtualReq.GetIncomingRequest().Method
	objRefReq := strings.Split(genesisrefs.GenesisRef(method).GetLocal().String(), ".")[0]

	result := testutils.GenerateVirtualResultRecord(pn, emptyObjID, gen.ID())
	result.Record.JetID = jetID
	virtualRes := result.Record.Virtual
	objRefRes := insolar.NewReference(*insolar.NewIDFromBytes(virtualRes.GetResult().GetObject().Bytes())).String()

	inNextPulse := testutils.GenerateRecordInNextPulse(pn)
	records := []*exporter.Record{request, result, inNextPulse}
	require.NoError(t, heavymock.ImportRecords(ts.ConMngr.ImporterClient, records))
	ts.WaitRecordsCount(t, len(records)-1, 5000)

	jetDropID := fmt.Sprintf("%v:%v", converter.JetIDToString(jetID), pn.String())
	c := GetHTTPClient()

	t.Run("request", func(t *testing.T) {
		t.Log("C5458 Receive request records with empty ObjectID")
		res, err := c.JetDropRecords(t, jetDropID, nil)
		require.NoError(t, err)
		require.NotEmpty(t, res.Result)
		for _, r := range res.Result {
			if request.Record.ID.String() == r.Reference {
				require.Equal(t, objRefReq, r.ObjectReference)
				return
			}
		}
		t.Fatalf("record with reference %v not found", request.Record.ID.String())
	})
	t.Run("result", func(t *testing.T) {
		t.Log("C5459 Receive result records with empty ObjectID")
		res, err := c.JetDropRecords(t, jetDropID, nil)
		require.NoError(t, err)
		require.NotEmpty(t, res.Result)
		for _, r := range res.Result {
			if result.Record.ID.String() == r.Reference {
				require.Equal(t, objRefRes, r.ObjectReference)
				return
			}
		}
		t.Fatalf("record with reference %v not found", result.Record.ID.String())
	})
}
