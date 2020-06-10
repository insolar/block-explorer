// +build bench_integration

package integration

import (
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/testutils"
	"testing"
)

func BenchmarkFetchPulse500RecordsSingleJet(b *testing.B) {
	records := 500
	jetDrops := 1
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	defer ts.Stop(b)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.importRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.waitRecordsCount(b, jetDrops*records)
		b.StopTimer()
		testutils.TruncateTables(b, ts.be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse1kRecordsSingleJet(b *testing.B) {
	records := 1000
	jetDrops := 1
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	defer ts.Stop(b)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.importRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.waitRecordsCount(b, jetDrops*records)
		b.StopTimer()
		testutils.TruncateTables(b, ts.be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse2kRecordsSingleJet(b *testing.B) {
	records := 2000
	jetDrops := 1
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	defer ts.Stop(b)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.importRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.waitRecordsCount(b, jetDrops*records)
		b.StopTimer()
		testutils.TruncateTables(b, ts.be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse500Records5Jets(b *testing.B) {
	records := 100
	jetDrops := 5
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	defer ts.Stop(b)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.importRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.waitRecordsCount(b, jetDrops*records)
		b.StopTimer()
		testutils.TruncateTables(b, ts.be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse500Records10Jets(b *testing.B) {
	records := 50
	jetDrops := 10
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	defer ts.Stop(b)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.importRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.waitRecordsCount(b, jetDrops*records)
		b.StopTimer()
		testutils.TruncateTables(b, ts.be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse500Records20Jets(b *testing.B) {
	records := 25
	jetDrops := 20
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	defer ts.Stop(b)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.importRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.waitRecordsCount(b, jetDrops*records)
		b.StopTimer()
		testutils.TruncateTables(b, ts.be.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}
