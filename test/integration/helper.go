package integration

import (
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	betest "github.com/insolar/block-explorer/testutils/betestsetup"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type BlockExplorerTestSuite struct {
	c  connectionmanager.ConnectionManager
	be betest.BlockExplorerTestSetUp
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
	a.c.Start(t)
	a.c.StartDB(t)

	a.be = betest.NewBlockExplorer(a.c.ExporterClient, a.c.DB)
	err := a.be.Start()
	require.NoError(t, err)
}

func (a *BlockExplorerTestSuite) Stop(t testing.TB) {
	err := a.be.Stop()
	require.NoError(t, err)
	// TODO remove sleep after resolving https://insolar.atlassian.net/browse/PENV-343
	time.Sleep(time.Second * 1)
	a.c.Stop()
}

func (a *BlockExplorerTestSuite) waitRecordsCount(t testing.TB, expCount int) {
	var c int
	for i := 0; i < 600; i++ {
		record := models.Record{}
		a.be.DB.Model(&record).Count(&c)
		t.Logf("Select from record, expected rows count=%v, actual=%v, attempt: %v", expCount, c, i)
		if c >= expCount {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("Found %v rows", c)
	require.Equal(t, expCount, c, "Records count in DB not as expected")
}

func (a *BlockExplorerTestSuite) importRecordsMultipleJetDrops(t testing.TB, jetDrops int, records int) {
	d := make([]*exporter.Record, 0)
	for i := 0; i < jetDrops; i++ {
		recs := testutils.GenerateRecordsFromOneJetSilence(1, records)
		d = append(d, recs...)
	}
	notFinalizedRecords := testutils.GenerateRecordsFromOneJetSilence(1, 1)
	d = append(d, notFinalizedRecords...)
	t.Logf("total records: %d", len(d))
	err := heavymock.ImportRecords(a.c.ImporterClient, d)
	require.NoError(t, err)
}
