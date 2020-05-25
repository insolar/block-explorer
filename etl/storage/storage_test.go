// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/testutils"
)

func TestStorage_SaveJetDropData(t *testing.T) {
	var err error
	testDB, dbCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	defer dbCleaner()
	s := NewStorage(testDB)

	firstRecord := testutils.InitRecordDB()
	secondRecord := testutils.InitRecordDB()

	err = s.SaveJetDropData(models.JetDrop{}, []models.Record{firstRecord, secondRecord})
	require.NoError(t, err)

	recordInDB := []models.Record{}
	err = testDB.Find(&recordInDB).Error
	require.NoError(t, err)
	require.Len(t, recordInDB, 2)
	require.EqualValues(t, firstRecord, recordInDB[0])
	require.EqualValues(t, secondRecord, recordInDB[1])
}

func TestStorage_SaveJetDropData_UpdateExistedRecord(t *testing.T) {
	var err error
	testDB, dbCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	defer dbCleaner()
	s := NewStorage(testDB)

	record := testutils.InitRecordDB()
	err = s.SaveJetDropData(models.JetDrop{}, []models.Record{record})
	require.NoError(t, err)
	newPayload := []byte{0, 1, 0, 1}
	require.NotEqual(t, record.Payload, newPayload)
	record.Payload = newPayload

	err = s.SaveJetDropData(models.JetDrop{}, []models.Record{record})
	require.NoError(t, err)

	recordInDB, err := s.GetRecord(record.Reference)
	require.NoError(t, err)
	require.EqualValues(t, record, recordInDB)
}

func TestStorage_SaveJetDropData_Error_NilPK(t *testing.T) {
	var err error
	testDB, dbCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	defer dbCleaner()
	s := NewStorage(testDB)

	record := testutils.InitRecordDB()
	record.Reference = nil

	err = s.SaveJetDropData(models.JetDrop{}, []models.Record{record})
	require.Error(t, err)
	require.Contains(t, err.Error(), "violates not-null constraint")
}

func TestStorage_SaveJetDropData_ErrorAtTransaction(t *testing.T) {
	var err error
	testDB, dbCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	defer dbCleaner()
	s := NewStorage(testDB)

	firstRecord := testutils.InitRecordDB()
	secondRecord := testutils.InitRecordDB()
	secondRecord.Reference = nil

	err = s.SaveJetDropData(models.JetDrop{}, []models.Record{firstRecord, secondRecord})
	require.Error(t, err)

	recordInDB := []models.Record{}
	err = testDB.Find(&recordInDB).Error
	require.NoError(t, err)
	require.Empty(t, recordInDB)
}
