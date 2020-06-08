// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package integration

import (
	"testing"

	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestHeavymockImporter_cleanAfterSend(t *testing.T) {
	cm := connectionmanager.ConnectionManager{}
	cm.Start(t)
	defer cm.Stop()

	recordsCount := 10
	expRecords := testutils.GenerateRecordsSilence(recordsCount)

	err := heavymock.ImportRecords(cm.ImporterClient, expRecords)
	require.NoError(t, err)
	require.Len(t, cm.Importer.GetUnsentRecords(), recordsCount)

	records, err := heavymock.ReceiveRecords(cm.ExporterClient, &exporter.GetRecords{})
	require.NoError(t, err)
	require.Len(t, records, recordsCount)

	require.Len(t, cm.Importer.GetUnsentRecords(), 0)

	records, err = heavymock.ReceiveRecords(cm.ExporterClient, &exporter.GetRecords{})
	require.NoError(t, err)
	require.Len(t, records, 0)
}
