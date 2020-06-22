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

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/insolar/insolar"
	ins_record "github.com/insolar/insolar/insolar/record"
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
			lastPn := uint32(0)
			for i := 0; i < test.differentPulseSize*test.recordCount+1; i++ {
				record, _ := fn()
				require.NotNil(t, record)
				pn := record.Record.ID.Pulse().AsUint32()
				require.GreaterOrEqual(t, pn, lastPn)
				lastPn = pn
			}

			_, err := fn()
			require.EqualError(t, err, io.EOF.Error())
		})
	}
}

func TestGenerateObjectLifeline(t *testing.T) {
	pulsesNumber := 5
	recordsNumber := 10
	lifeline := GenerateObjectLifeline(pulsesNumber, recordsNumber)
	require.Len(t, lifeline.StateRecords, pulsesNumber)
	require.Len(t, lifeline.SideRecords, 2)

	objID := lifeline.ObjID
	allRecords := make([]*exporter.Record, 0)
	var prevPn insolar.PulseNumber
	prevPn = 0
	for i := 0; i < pulsesNumber; i++ {
		pn := lifeline.StateRecords[i].Pn
		require.Greater(t, pn.AsUint32(), prevPn.AsUint32())
		prevPn = pn

		records := lifeline.StateRecords[i].Records
		require.Len(t, records, recordsNumber)
		allRecords = append(allRecords, records...)
	}

	var amendCount int
	var unknown int
	for _, r := range allRecords {
		require.Equal(t, objID, r.Record.ObjectID)

		virtual := r.Record.Virtual
		switch virtual.Union.(type) {
		case *ins_record.Virtual_Amend:
			amendCount++
		default:
			unknown++
		}
	}
	require.Equal(t, 0, unknown)
	require.Equal(t, pulsesNumber*recordsNumber, amendCount)

	sideRecords := make([]*exporter.Record, 0)
	sideRecords = append(sideRecords, lifeline.SideRecords[0].Records...)
	sideRecords = append(sideRecords, lifeline.SideRecords[1].Records...)
	var activateCount int
	var incomingCount int
	for _, r := range sideRecords {
		require.Equal(t, objID, r.Record.ObjectID)

		virtual := r.Record.Virtual
		switch virtual.Union.(type) {
		case *ins_record.Virtual_Activate:
			activateCount++
		case *ins_record.Virtual_IncomingRequest:
			incomingCount++
		default:
			unknown++
		}
	}
	require.Equal(t, 1, activateCount)
	require.Equal(t, 1, incomingCount)
	require.Equal(t, 0, unknown)

	all := lifeline.GetAllRecords()
	require.Len(t, all, pulsesNumber*recordsNumber+2)
	sr := lifeline.GetStateRecords()
	require.Len(t, sr, pulsesNumber*recordsNumber)
}

func TestExporterJetIDToString(t *testing.T) {
	jetID := GenerateUniqueJetID().Prefix()
	toString := models.ExporterJetIDToString(jetID)
	fromString, err := models.NewJetIDFromString(toString)
	require.NoError(t, err)
	strFromString := models.ExporterJetIDToString(fromString)
	require.Equal(t, toString, strFromString)
	require.Equal(t, jetID, fromString)
}
