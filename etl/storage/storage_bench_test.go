// +build bench

package storage

import (
	"testing"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/testutils"
	"github.com/stretchr/testify/require"
)

func BenchmarkSaveJetDropData(b *testing.B) {
	s := NewStorage(testDB)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		testutils.TruncateTables(b, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
		pulse, err := testutils.InitPulseDB()
		require.NoError(b, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(b, err)
		jetDrop := testutils.InitJetDropDB(pulse)
		firstRecord := testutils.InitRecordDB(jetDrop)
		secondRecord := testutils.InitRecordDB(jetDrop)

		b.StartTimer()
		err = s.SaveJetDropData(jetDrop, []models.Record{firstRecord, secondRecord}, pulse.PulseNumber)
		require.NoError(b, err)
		b.StopTimer()

		jetDropInDB := []models.JetDrop{}
		err = testDB.Find(&jetDropInDB).Error
		require.NoError(b, err)
	}
}
