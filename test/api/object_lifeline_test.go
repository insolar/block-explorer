// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/antihax/optional"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	betest "github.com/insolar/block-explorer/testutils/betestsetup"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type apiLifelineSuite struct {
	suite.Suite
	c  connectionmanager.ConnectionManager
	be betest.BlockExplorerTestSetUp
}

func (a *apiLifelineSuite) SetupTest() {
	a.c.Start(a.T())
	a.c.StartDB(a.T())
	a.c.StartAPIServer(a.T())

	a.be = betest.NewBlockExplorer(a.c.ExporterClient, a.c.DB)
	err := a.be.Start()
	require.NoError(a.T(), err)
}

func (a *apiLifelineSuite) TearDownTest() {
	err := a.be.Stop()
	require.NoError(a.T(), err)
	// TODO remove sleep after resolving https://insolar.atlassian.net/browse/PENV-343
	time.Sleep(time.Second * 1)
	a.c.Stop()
}

func (a *apiLifelineSuite) TestLifeline_onePulse() {
	pulsesNumber := 1
	recordsInPulse := 10
	lifeline := testutils.GenerateObjectLifeline(pulsesNumber, recordsInPulse)

	lastPulseRecord := testutils.GenerateRecordsSilence(1)[0]
	lastPulseRecord.Record.ID = gen.IDWithPulse(lifeline.States[0].Pn + 10)
	lastPulseRecord.ShouldIterateFrom = nil

	lifeline.States[0].Records = append(lifeline.States[0].Records, lastPulseRecord)

	err := heavymock.ImportRecords(a.c.ImporterClient, lifeline.States[0].Records)
	require.NoError(a.T(), err)

	stateRecordsCount := pulsesNumber * recordsInPulse
	totalRecords := stateRecordsCount + 2
	a.waitRecordsCount(totalRecords)

	c := NewBeApiClient(a.T(), fmt.Sprintf("http://localhost%v", connectionmanager.DefaultApiPort))
	response, err := c.ObjectLifeline(lifeline.ObjID.String(), nil)
	require.NoError(a.T(), err)
	require.Len(a.T(), response.Result, stateRecordsCount)
	for _, res := range response.Result {
		require.Contains(a.T(), lifeline.ObjID.String(), res.ObjectReference)
		// TODO: change it to 'lifeline.States[0].Pn' at PENV-212
		require.Equal(a.T(), int64(1), res.PulseNumber)
	}
}

func (a *apiLifelineSuite) TestLifeline_severalPulses() {
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
	err := heavymock.ImportRecords(a.c.ImporterClient, records)
	require.NoError(a.T(), err)

	stateRecordsCount := pulsesNumber * recordsInPulse
	totalRecords := stateRecordsCount + 2
	a.waitRecordsCount(totalRecords)

	c := NewBeApiClient(a.T(), fmt.Sprintf("http://localhost%v", connectionmanager.DefaultApiPort))
	response, err := c.ObjectLifeline(lifeline.ObjID.String(), &client.ObjectLifelineOpts{Limit: optional.NewInt32(100)})
	require.NoError(a.T(), err)
	require.Len(a.T(), response.Result, stateRecordsCount)
	pulses := make([]int64, pulsesNumber)
	for i, s := range lifeline.States {
		pulses[i] = int64(s.Pn)
	}
	for _, res := range response.Result {
		require.Contains(a.T(), lifeline.ObjID.String(), res.ObjectReference)
		require.Contains(a.T(), pulses, res.PulseNumber)
	}
}

func (a *apiLifelineSuite) waitRecordsCount(expCount int) {
	var c int
	for i := 0; i < 60; i++ {
		record := models.Record{}
		a.c.DB.Model(&record).Count(&c)
		a.T().Logf("Select from record, expected rows count=%v, actual=%v, attempt: %v", expCount, c, i)
		if c >= expCount {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	a.T().Logf("Found %v rows", c)
	require.Equal(a.T(), expCount, c, "Records count in DB not as expected")
}

func TestAll(t *testing.T) {
	suite.Run(t, new(apiLifelineSuite))
}
