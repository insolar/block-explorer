// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package storage

import (
	"testing"
	"time"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/testutils"
)

func initRecord() models.Record {
	return models.Record{
		Reference:           gen.Reference().Bytes(),
		Type:                "",
		ObjectReference:     gen.Reference().Bytes(),
		PrototypeReference:  gen.Reference().Bytes(),
		Payload:             []byte{1, 2, 3},
		PrevRecordReference: gen.Reference().Bytes(),
		Hash:                []byte{1, 2, 3, 4},
		RawData:             []byte{1, 2, 3, 4, 5},
		JetID:               []byte{1},
		PulseNumber:         1,
		Order:               1,
		Timestamp:           time.Now().Unix(),
	}
}

func TestStorage_SaveJetDropData(t *testing.T) {
	testDB, dbCleaner := testutils.SetupDB()
	defer dbCleaner()
	s := NewStorage(testDB)

	firstRecord := initRecord()
	secondRecord := initRecord()

	err := s.SaveJetDropData(models.JetDrop{}, []models.Record{firstRecord, secondRecord})
	require.NoError(t, err)

	recordInDB := []models.Record{}
	err = testDB.Find(&recordInDB).Error
	require.NoError(t, err)
	require.Len(t, recordInDB, 2)
	require.EqualValues(t, firstRecord, recordInDB[0])
	require.EqualValues(t, secondRecord, recordInDB[1])
}

func TestStorage_SaveJetDropData_UpdateExistedRecord(t *testing.T) {
	testDB, dbCleaner := testutils.SetupDB()
	defer dbCleaner()
	s := NewStorage(testDB)

	record := initRecord()
	err := s.SaveJetDropData(models.JetDrop{}, []models.Record{record})
	require.NoError(t, err)
	newPayload := []byte{0,1,0,1}
	require.NotEqual(t, record.Payload, newPayload)
	record.Payload = newPayload

	err = s.SaveJetDropData(models.JetDrop{}, []models.Record{record})
	require.NoError(t, err)

	recordInDB, err := s.GetRecord(record.Reference)
	require.NoError(t, err)
	require.EqualValues(t, record, recordInDB)
}

func TestStorage_SaveJetDropData_Error_NilPK(t *testing.T) {
	testDB, dbCleaner := testutils.SetupDB()
	defer dbCleaner()
	s := NewStorage(testDB)

	record := initRecord()
	record.Reference = nil

	err := s.SaveJetDropData(models.JetDrop{}, []models.Record{record})
	require.Error(t, err)
	require.Contains(t, err.Error(), "violates not-null constraint")
}

func TestStorage_SaveJetDropData_ErrorAtTransaction(t *testing.T) {
	testDB, dbCleaner := testutils.SetupDB()
	defer dbCleaner()
	s := NewStorage(testDB)

	firstRecord := initRecord()
	secondRecord := initRecord()
	secondRecord.Reference = nil

	err := s.SaveJetDropData(models.JetDrop{}, []models.Record{firstRecord, secondRecord})
	require.Error(t, err)

	recordInDB := []models.Record{}
	err = testDB.Find(&recordInDB).Error
	require.NoError(t, err)
	require.Empty(t, recordInDB)
}
