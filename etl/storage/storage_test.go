// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package storage

import (
	"context"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/testutils"
)

var testDB *gorm.DB

func TestMain(t *testing.M) {
	var dbCleaner func()
	var err error
	testDB, dbCleaner, err = testutils.SetupDB()
	if err != nil {
		belogger.FromContext(context.Background()).Fatal(err)
	}
	retCode := t.Run()
	dbCleaner()
	os.Exit(retCode)
}

func TestStorage_SaveJetDropData(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	pulse.PulseNumber = 1
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	firstRecord := testutils.InitRecordDB(jetDrop)
	secondRecord := testutils.InitRecordDB(jetDrop)

	err = s.SaveJetDropData(jetDrop, []models.Record{firstRecord, secondRecord})
	require.NoError(t, err)

	jetDropInDB := []models.JetDrop{}
	err = testDB.Find(&jetDropInDB).Error
	require.NoError(t, err)
	require.Len(t, jetDropInDB, 1)
	require.EqualValues(t, jetDrop, jetDropInDB[0])

	recordInDB := []models.Record{}
	err = testDB.Find(&recordInDB).Error
	require.NoError(t, err)
	require.Len(t, recordInDB, 2)
	require.EqualValues(t, firstRecord, recordInDB[0])
	require.EqualValues(t, secondRecord, recordInDB[1])
}

func TestStorage_SaveJetDropData_UpdateExistedRecord(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	pulse.PulseNumber = 1
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	record := testutils.InitRecordDB(jetDrop)
	err = s.SaveJetDropData(jetDrop, []models.Record{record})
	require.NoError(t, err)
	newPayload := []byte{0, 1, 0, 1}
	require.NotEqual(t, record.Payload, newPayload)
	record.Payload = newPayload

	err = s.SaveJetDropData(
		models.JetDrop{PulseNumber: pulse.PulseNumber, JetID: testutils.GenerateUniqueJetID().Prefix()},
		[]models.Record{record},
	)
	require.NoError(t, err)

	recordInDB, err := s.GetRecord(record.Reference)
	require.NoError(t, err)
	require.EqualValues(t, record, recordInDB)
}

func TestStorage_SaveJetDropData_UpdateExistedJetDrop(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	pulse.PulseNumber = 1
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	record := testutils.InitRecordDB(jetDrop)
	err = s.SaveJetDropData(jetDrop, []models.Record{record})
	require.NoError(t, err)
	newPayload := []byte{0, 1, 0, 1}
	require.NotEqual(t, record.Payload, newPayload)
	record.Payload = newPayload

	err = s.SaveJetDropData(jetDrop, []models.Record{record})
	require.NoError(t, err)

	jetDropInDB := []models.JetDrop{}
	err = testDB.Find(&jetDropInDB).Error
	require.NoError(t, err)
	require.Len(t, jetDropInDB, 1)
	require.EqualValues(t, jetDrop, jetDropInDB[0])
}

func TestStorage_SaveJetDropData_RecordError_NilPK(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	pulse.PulseNumber = 1
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	record := testutils.InitRecordDB(jetDrop)
	record.Reference = nil

	err = s.SaveJetDropData(jetDrop, []models.Record{record})
	require.Error(t, err)
	require.Contains(t, err.Error(), "violates not-null constraint")
	require.Contains(t, err.Error(), "error while saving record")
}

func TestStorage_SaveJetDropData_JetDropError_NilPK(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	pulse.PulseNumber = 1
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	jetDrop.JetID = nil
	record := testutils.InitRecordDB(jetDrop)

	err = s.SaveJetDropData(jetDrop, []models.Record{record})
	require.Error(t, err)
	require.Contains(t, err.Error(), "violates not-null constraint")
	require.Contains(t, err.Error(), "error while saving jetDrop")
}

func TestStorage_SaveJetDropData_ErrorAtTransaction(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	pulse.PulseNumber = 1
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	firstRecord := testutils.InitRecordDB(jetDrop)
	secondRecord := testutils.InitRecordDB(jetDrop)
	secondRecord.Reference = nil

	err = s.SaveJetDropData(jetDrop, []models.Record{firstRecord, secondRecord})
	require.Error(t, err)

	jetDropInDB := []models.JetDrop{}
	err = testDB.Find(&jetDropInDB).Error
	require.NoError(t, err)
	require.Empty(t, jetDropInDB)

	recordInDB := []models.Record{}
	err = testDB.Find(&recordInDB).Error
	require.NoError(t, err)
	require.Empty(t, recordInDB)
}

func TestStorage_GetNotCompletePulses(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
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
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
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

func TestStorage_CompletePulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	err = s.CompletePulse(pulse.PulseNumber)
	require.NoError(t, err)

	pulse.IsComplete = true
	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 1)
	require.EqualValues(t, pulse, pulseInDB[0])
}

func TestStorage_CompletePulse_ErrorUpdateSeveralRows(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	pulse, err = testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	err = s.CompletePulse(0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "several rows were affected")

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 2)
	require.EqualValues(t, false, pulseInDB[0].IsComplete)
	require.EqualValues(t, false, pulseInDB[1].IsComplete)

}

func TestStorage_CompletePulse_AlreadyCompleted(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	pulse.IsComplete = true
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	err = s.CompletePulse(pulse.PulseNumber)
	require.NoError(t, err)

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 1)
	require.EqualValues(t, pulse, pulseInDB[0])
}

func TestStorage_CompletePulse_ErrorNotExist(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)

	err = s.CompletePulse(pulse.PulseNumber)
	require.Error(t, err)
	require.Contains(t, err.Error(), "try to complete not existing pulse")

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Empty(t, pulseInDB)
}

func TestStorage_SavePulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)

	err = s.SavePulse(pulse)
	require.NoError(t, err)

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 1)
	require.EqualValues(t, pulse, pulseInDB[0])
}

func TestStorage_SavePulse_Existed(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	err = s.SavePulse(pulse)
	require.NoError(t, err)

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 1)
	require.EqualValues(t, pulse, pulseInDB[0])
}

func TestStorage_SavePulse_Error(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	pulse.PulseNumber = 0

	err = s.SavePulse(pulse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "violates not-null constraint")
	require.Contains(t, err.Error(), "error while saving pulse")

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Empty(t, pulseInDB)
}
