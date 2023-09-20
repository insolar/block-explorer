package integration

import (
	"bytes"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	betest "github.com/insolar/block-explorer/testutils/betestsetup"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
)

type BlockExplorerTestSuite struct {
	ConMngr *connectionmanager.ConnectionManager
	BE      betest.BlockExplorerTestSetUp
}

func NewBlockExplorerTestSetup(t testing.TB) *BlockExplorerTestSuite {
	c := &connectionmanager.ConnectionManager{}
	c.Start(t)
	c.StartDB(t)
	be := betest.NewBlockExplorer(c.ExporterClient, c.DB)
	be.PulseClient.SetNextFinalizedPulseFunc(c.Importer)
	return &BlockExplorerTestSuite{
		ConMngr: c,
		BE:      be,
	}
}

func (a *BlockExplorerTestSuite) Start(t testing.TB) {
	a.ConMngr.Start(t)
	a.ConMngr.StartDB(t)

	a.BE = betest.NewBlockExplorer(a.ConMngr.ExporterClient, a.ConMngr.DB)
	err := a.BE.Start()
	require.NoError(t, err)
}

func (a *BlockExplorerTestSuite) Stop(t testing.TB) {
	a.ConMngr.Stop()
}

func (a *BlockExplorerTestSuite) StartBE(t testing.TB) {
	err := a.BE.Start()
	require.NoError(t, err)
}

func (a *BlockExplorerTestSuite) StopBE(t *testing.T) {
	err := a.BE.Stop()
	require.NoError(t, err)
}

func (a *BlockExplorerTestSuite) WithHTTPServer(t testing.TB) *BlockExplorerTestSuite {
	a.ConMngr.StartAPIServer(t)
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

func (a *BlockExplorerTestSuite) WaitJetDropHash(t testing.TB, id models.JetDropID, expectedHash []byte, timeoutMs int) {
	interval := 100
	jetDrop := models.JetDrop{}
	for i := 0; i < timeoutMs/interval; i++ {
		a.BE.DB.Where("pulse_number = ? AND jet_id = ?", id.PulseNumber, id.JetID).Find(&jetDrop)
		t.Logf("Select from jetDrops, expected jetDrop hash %v, actual=%v, attempt: %v", expectedHash, jetDrop.Hash, i)
		if bytes.Equal(expectedHash, jetDrop.Hash) {
			break
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	t.Logf("Found jetDrop with hash %v", jetDrop.Hash)
	require.Equal(t, expectedHash, jetDrop.Hash, "JetDrop hash in DB not as expected")
}

func (a *BlockExplorerTestSuite) CheckForRecordsNotChanged(t testing.TB, expCount int, timeoutMs int) {
	var c int
	interval := 100
	for i := 0; i < timeoutMs/interval; i++ {
		record := models.Record{}
		a.BE.DB.Model(&record).Count(&c)
		t.Logf("Wait records in DB count unchanged: expected rows count=%v, actual=%v, attempt: %v", expCount, c, i)
		require.Equal(t, expCount, c)
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

// nolint
func (a *BlockExplorerTestSuite) ImportRecordsMultipleJetDrops(t testing.TB, jetDrops int, records int, pulse insolar.PulseNumber) {
	d := make([]*exporter.Record, 0)
	for i := 0; i < jetDrops; i++ {
		recs := testutils.GenerateRecordsFromOneJetSilence(1, records)
		d = append(d, recs...)
	}
	for _, r := range d {
		r.Record.ID = gen.IDWithPulse(pulse)
	}
	t.Logf("total records: %d", len(d))
	err := heavymock.ImportRecords(a.ConMngr.ImporterClient, d)
	require.NoError(t, err)
}
