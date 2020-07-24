// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

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
	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(b)
	defer ts.Stop(b)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.ImportRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.WaitRecordsCount(b, jetDrops*records, 60000)
		b.StopTimer()
		testutils.TruncateTables(b, ts.BE.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse1kRecordsSingleJet(b *testing.B) {
	records := 1000
	jetDrops := 1
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(b)
	defer ts.Stop(b)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.ImportRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.WaitRecordsCount(b, jetDrops*records, 60000)
		b.StopTimer()
		testutils.TruncateTables(b, ts.BE.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse2kRecordsSingleJet(b *testing.B) {
	records := 2000
	jetDrops := 1
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(b)
	defer ts.Stop(b)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.ImportRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.WaitRecordsCount(b, jetDrops*records, 60000)
		b.StopTimer()
		testutils.TruncateTables(b, ts.BE.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse500Records5Jets(b *testing.B) {
	records := 100
	jetDrops := 5
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(b)
	defer ts.Stop(b)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.ImportRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.WaitRecordsCount(b, jetDrops*records, 60000)
		b.StopTimer()
		testutils.TruncateTables(b, ts.BE.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse500Records10Jets(b *testing.B) {
	records := 50
	jetDrops := 10
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(b)
	defer ts.Stop(b)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.ImportRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.WaitRecordsCount(b, jetDrops*records, 60000)
		b.StopTimer()
		testutils.TruncateTables(b, ts.BE.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}

func BenchmarkFetchPulse500Records20Jets(b *testing.B) {
	records := 25
	jetDrops := 20
	b.ResetTimer()
	ts := NewBlockExplorerTestSetup(b)
	ts.BE.PulseClient.SetNextFinalizedPulseFunc(ts.ConMngr.Importer)
	ts.StartBE(b)
	defer ts.Stop(b)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.ImportRecordsMultipleJetDrops(b, jetDrops, records)
		b.StartTimer()
		ts.WaitRecordsCount(b, jetDrops*records, 60000)
		b.StopTimer()
		testutils.TruncateTables(b, ts.BE.DB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	}
}
