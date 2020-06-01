// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package testutils

import (
	"fmt"
	"io"
	"testing"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestGenerateRecords_CanReturnEOF(t *testing.T) {
	batchSize := 5
	f := GenerateRecords(batchSize)

	n := uint32(1)
	for i := 0; i < batchSize; i++ {
		record, err := f()
		require.NoError(t, err)
		require.Equal(t, n, record.RecordNumber)
		n++
	}
	res, err := f()
	require.Error(t, err)
	require.Equal(t, io.EOF, err)
	require.Equal(t, &exporter.Record{}, res)
}

func TestGenerateRecordsSilence_recordsAreUnique(t *testing.T) {
	count := 5
	records := GenerateRecordsSilence(count)
	require.Len(t, records, count)
	for i, r := range records {
		require.Equal(t, uint32(i+1), r.RecordNumber)
	}
}

func TestGenerateUniqueJetIDFunction(t *testing.T) {
	ids := len(uniqueJetID)
	idFirst := GenerateUniqueJetID()
	require.NotEmpty(t, idFirst)
	require.Len(t, uniqueJetID, ids+1)

	idSecond := GenerateUniqueJetID()
	require.NotEqual(t, idFirst, idSecond)
	require.NotEmpty(t, idSecond)
	require.Len(t, uniqueJetID, ids+2)
}

func TestGenerateRecordsWithDifferencePulses(t *testing.T) {
	tests := []struct {
		differentPulseSize int
		recordCount        int
	}{
		{
			differentPulseSize: 1,
			recordCount:        1,
		}, {
			differentPulseSize: 1,
			recordCount:        2,
		}, {
			differentPulseSize: 2,
			recordCount:        1,
		}, {
			differentPulseSize: 2,
			recordCount:        2,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("pulse-size=%d,record-count=%d", test.differentPulseSize, test.recordCount), func(t *testing.T) {
			fn := GenerateRecordsWithDifferencePulses(test.differentPulseSize, test.recordCount)

			for i := 0; i < test.differentPulseSize*test.recordCount+1; i++ {
				record, _ := fn()
				require.NotNil(t, record)
			}

			_, err := fn()
			require.EqualError(t, err, io.EOF.Error())
		})
	}
}
