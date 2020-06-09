// +build bench_integration

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
)

func BenchmarkFetchPulse2kRecords(b *testing.B) {
	b.ResetTimer()
	c := new(connectionmanager.ConnectionManager)
	c.Start(b)
	c.StartDB(b)
	be := betest.NewBlockExplorer(c.ExporterClient, c.DB)
	err := be.Start()
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// generating 2 pulses, previous will be fetched, current is not finalized for now
		recordsCount := 2000
		pulses := 2
		// current pulse is not finalized
		expRecordsJet1 := testutils.GenerateRecordsFromOneJetSilence(pulses, 1)
		expRecordsJet2 := testutils.GenerateRecordsFromOneJetSilence(pulses, recordsCount)
		expRecords := make([]*exporter.Record, 0)
		expRecords = append(expRecords, expRecordsJet1...)
		expRecords = append(expRecords, expRecordsJet2...)
		err := heavymock.ImportRecords(c.ImporterClient, expRecords)
		require.NoError(b, err)

		b.StartTimer()
		// last records with the biggest pulse number won't be processed, so we do not expect this record in DB
		waitRecordsCount(b, be.DB, len(expRecords)-recordsCount)
		testutils.TruncateTables(b, be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}
