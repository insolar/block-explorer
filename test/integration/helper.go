// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	betest "github.com/insolar/block-explorer/testutils/betestsetup"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

type BlockExplorerTestSuite struct {
	C  connectionmanager.ConnectionManager
	BE betest.BlockExplorerTestSetUp
}

func NewBlockExplorerTestSetup(t testing.TB) *BlockExplorerTestSuite {
	c := connectionmanager.ConnectionManager{}
	c.Start(t)
	c.StartDB(t)
	be := betest.NewBlockExplorer(c.ExporterClient, c.DB)
	err := be.Start()
	require.NoError(t, err)
	return &BlockExplorerTestSuite{
		c,
		be,
	}
}

func (a *BlockExplorerTestSuite) Start(t testing.TB) {
	a.C.Start(t)
	a.C.StartDB(t)

	a.BE = betest.NewBlockExplorer(a.C.ExporterClient, a.C.DB)
	err := a.BE.Start()
	require.NoError(t, err)
}

func (a *BlockExplorerTestSuite) Stop(t testing.TB) {
	err := a.BE.Stop()
	require.NoError(t, err)
	// TODO remove sleep after resolving https://insolar.atlassian.net/browse/PENV-343
	time.Sleep(time.Second * 1)
	a.C.Stop()
}

func (a *BlockExplorerTestSuite) WithHTTPServer(t testing.TB) *BlockExplorerTestSuite {
	a.C.StartAPIServer(t)
	return a
}

// nolint
func (a *BlockExplorerTestSuite) WaitRecordsCount(t testing.TB, expCount int, timeoutMs int) {
	var c int
	interval := 100
	for i := 0; i < timeoutMs/interval; i++ {
		record := models.Record{}
		a.BE.DB.Model(&record).Count(&c)
		t.Logf("Select from record, expected rows count=%v, actual=%v, attempt: %v", expCount, c, i)
		if c >= expCount {
			break
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	t.Logf("Found %v rows", c)
	require.Equal(t, expCount, c, "Records count in DB not as expected")
}

// nolint
func (a *BlockExplorerTestSuite) ImportRecordsMultipleJetDrops(t testing.TB, jetDrops int, records int) {
	d := make([]*exporter.Record, 0)
	for i := 0; i < jetDrops; i++ {
		recs := testutils.GenerateRecordsFromOneJetSilence(1, records)
		d = append(d, recs...)
	}
	notFinalizedRecords := testutils.GenerateRecordsFromOneJetSilence(1, 1)
	d = append(d, notFinalizedRecords...)
	t.Logf("total records: %d", len(d))
	err := heavymock.ImportRecords(a.C.ImporterClient, d)
	require.NoError(t, err)
}
