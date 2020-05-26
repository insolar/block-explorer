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

func TestStorage_GetNotCompletePulses(t *testing.T) {
	var err error
	testDB, dbCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	defer dbCleaner()
	s := NewStorage(testDB)

	completePulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	completePulse.IsComplete = true
	err = testutils.CreatePulse(testDB, completePulse)
	require.NoError(t, err)

	notCompletePulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, notCompletePulse)
	require.NoError(t, err)

	pulses, err := s.GetIncompletePulses()
	require.NoError(t, err)
	require.Equal(t, []models.Pulse{notCompletePulse}, pulses)
	require.False(t, pulses[0].IsComplete)
}

func TestStorage_GetJetDrops(t *testing.T) {
	var err error
	testDB, dbCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	defer dbCleaner()
	s := NewStorage(testDB)

	firstPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)
	jetDropForFirstPulse1 := testutils.InitJetDropDB(firstPulse)
	err = testutils.CreateJetDrop(testDB, jetDropForFirstPulse1)
	require.NoError(t, err)
	jetDropForFirstPulse2 := testutils.InitJetDropDB(firstPulse)
	err = testutils.CreateJetDrop(testDB, jetDropForFirstPulse2)
	require.NoError(t, err)

	secondPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)
	jetDropForSecondPulse := testutils.InitJetDropDB(secondPulse)
	err = testutils.CreateJetDrop(testDB, jetDropForSecondPulse)
	require.NoError(t, err)

	jetDrops, err := s.GetJetDrops(firstPulse)
	require.NoError(t, err)
	require.Equal(t, []models.JetDrop{jetDropForFirstPulse1, jetDropForFirstPulse2}, jetDrops)
}
