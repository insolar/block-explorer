// +build unit

package integration

import (
	"testing"

	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestHeavymockImporter_cleanAfterSend(t *testing.T) {
	cm := connectionmanager.ConnectionManager{}
	cm.Start(t)
	defer cm.Stop()

	recordsCount := 10
	expRecords := testutils.GenerateRecordsSilence(recordsCount)
	pu := gen.PulseNumber()
	for i, _ := range expRecords {
		expRecords[i].Record.ID = *insolar.NewID(pu, nil)
	}

	err := heavymock.ImportRecords(cm.ImporterClient, expRecords)
	require.NoError(t, err)
	require.Len(t, cm.Importer.GetUnsentRecords(), recordsCount)

	records, err := heavymock.ReceiveRecords(cm.ExporterClient, &exporter.GetRecords{PulseNumber: pu})
	require.NoError(t, err)
	require.Len(t, records, recordsCount+1)

	require.Len(t, cm.Importer.GetUnsentRecords(), 0)

	records, err = heavymock.ReceiveRecords(cm.ExporterClient, &exporter.GetRecords{PulseNumber: pu})
	require.NoError(t, err)
	require.Len(t, records, 1)
}
