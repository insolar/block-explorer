package integration

import (
	"github.com/insolar/block-explorer/etl/models"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func waitRecordsCount(t testing.TB, db *gorm.DB, expCount int) {
	var c int
	for i := 0; i < 600; i++ {
		record := models.Record{}
		db.Model(&record).Count(&c)
		t.Logf("Select from record, expected rows count=%v, actual=%v, attempt: %v", expCount, c, i)
		if c >= expCount {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("Found %v rows", c)
	require.Equal(t, expCount, c, "Records count in DB not as expected")
}
