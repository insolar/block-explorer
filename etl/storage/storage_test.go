// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration bench

package storage

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/insolar/insolar/gen"
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
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
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
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	jetDrop := testutils.InitJetDropDB(pulse)
	record := testutils.InitRecordDB(jetDrop)
	err = s.SaveJetDropData(jetDrop, []models.Record{record})
	require.NoError(t, err)
	newPayload := []byte{0, 1, 0, 1}
	require.NotEqual(t, record.Payload, newPayload)
	record.Payload = newPayload

	err = s.SaveJetDropData(
		models.JetDrop{PulseNumber: pulse.PulseNumber, JetID: converter.JetIDToString(testutils.GenerateUniqueJetID())},
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
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
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
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
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
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	jetDrop := testutils.InitJetDropDB(pulse)
	jetDrop.JetID = ""
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
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	jetDrop := testutils.InitJetDropDB(pulse)
	firstRecord := testutils.InitRecordDB(jetDrop)
	secondRecord := testutils.InitRecordDB(jetDrop)
	secondRecord.Reference = nil

	err = s.SaveJetDropData(jetDrop, []models.Record{firstRecord, secondRecord})
	require.Error(t, err)
	require.Contains(t, err.Error(), "error while saving record")

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
	expected := []models.JetDrop{jetDropForFirstPulse1, jetDropForFirstPulse2}
	require.Len(t, jetDrops, 2)
	require.Contains(t, expected, jetDrops[0])
	require.Contains(t, expected, jetDrops[1])
}

func TestStorage_GetJetDropsWithParams(t *testing.T) {
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

	t.Run("happy common", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsWithParams(firstPulse, nil, 20, 0)
		require.NoError(t, err)
		expected := []models.JetDrop{jetDropForFirstPulse1, jetDropForFirstPulse2}
		require.Len(t, jetDrops, 2)
		require.Contains(t, expected, jetDrops[0])
		require.Contains(t, expected, jetDrops[1])
		require.Equal(t, 2, total)
	})
	t.Run("happy limit", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsWithParams(firstPulse, nil, 1, 0)
		require.NoError(t, err)
		expected := []models.JetDrop{jetDropForFirstPulse1, jetDropForFirstPulse2}
		require.Len(t, jetDrops, 1)
		require.Contains(t, expected, jetDrops[0])
		require.Equal(t, 2, total)
	})
	t.Run("happy limit/offset", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsWithParams(firstPulse, nil, 1, 1)
		require.NoError(t, err)
		expected := []models.JetDrop{jetDropForFirstPulse1, jetDropForFirstPulse2}
		require.Len(t, jetDrops, 1)
		require.Contains(t, expected, jetDrops[0])
		require.Equal(t, 2, total)
	})
}

func TestStorage_GetJetDropsByID(t *testing.T) {
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

	jetDrop, err := s.GetJetDropByID(*models.NewJetDropID(jetDropForSecondPulse.JetID, int64(jetDropForSecondPulse.PulseNumber)))
	require.NoError(t, err)
	require.EqualValues(t, jetDropForSecondPulse, jetDrop)
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

func TestStorage_SequencePulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	err = s.SequencePulse(pulse.PulseNumber)
	require.NoError(t, err)

	pulse.IsSequential = true
	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 1)
	require.EqualValues(t, pulse, pulseInDB[0])
}

func TestStorage_SequencePulse_ErrorUpdateSeveralRows(t *testing.T) {
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

	err = s.SequencePulse(0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "several rows were affected")

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 2)
	require.EqualValues(t, false, pulseInDB[0].IsComplete)
	require.EqualValues(t, false, pulseInDB[1].IsComplete)

}

func TestStorage_SequencePulse_AlreadyCompleted(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	pulse.IsSequential = true
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	err = s.SequencePulse(pulse.PulseNumber)
	require.NoError(t, err)

	pulseInDB := []models.Pulse{}
	err = testDB.Find(&pulseInDB).Error
	require.NoError(t, err)
	require.Len(t, pulseInDB, 1)
	require.EqualValues(t, pulse, pulseInDB[0])
}

func TestStorage_SequencePulse_ErrorNotExist(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)

	err = s.SequencePulse(pulse.PulseNumber)
	require.Error(t, err)
	require.Contains(t, err.Error(), "try to sequence not existing pulse")

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

type pulseRecords struct {
	records []models.Record
	pulse   models.Pulse
}

func pulseSequenceOneObject(t *testing.T, pulseAmount, recordsAmount int) []pulseRecords {
	var result []pulseRecords
	objRef := gen.ID()
	pulseNumber := gen.PulseNumber()
	for i := 0; i < pulseAmount; i++ {
		timestamp, err := pulseNumber.AsApproximateTime()
		require.NoError(t, err)
		pulse := models.Pulse{PulseNumber: int(pulseNumber), Timestamp: timestamp.Unix()}
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)
		jetDrop := testutils.InitJetDropDB(pulse)
		err = testutils.CreateJetDrop(testDB, jetDrop)
		require.NoError(t, err)
		records := testutils.OrderedRecords(t, testDB, jetDrop, objRef, recordsAmount)
		result = append(result, pulseRecords{records: records, pulse: pulse})
		pulseNumber = pulseNumber.Next(10)
	}
	return result
}

func TestStorage_GetLifeline(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.ID()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, objRef, 3)
	testutils.OrderedRecords(t, testDB, jetDrop, gen.ID(), 3)

	expectedRecords := []models.Record{genRecords[2], genRecords[1], genRecords[0]}

	records, total, err := s.GetLifeline(objRef.Bytes(), nil, nil, nil, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_ObjNotExist(t *testing.T) {
	s := NewStorage(testDB)

	records, total, err := s.GetLifeline(gen.Reference().Bytes(), nil, nil, nil, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, records)
}

func TestStorage_GetLifeline_Index(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.ID()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, objRef, 3)

	expectedRecords := []models.Record{genRecords[1], genRecords[0]}

	index := fmt.Sprintf("%d:%d", pulse.PulseNumber, genRecords[1].Order)
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_Index_NotExist(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulseNumber := int(gen.PulseNumber().AsUint32())
	objRef := gen.Reference()

	index := fmt.Sprintf("%d:%d", pulseNumber, 10)
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, records)
}

func TestStorage_GetLifeline_IndexLimit(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.ID()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, objRef, 5)

	expectedRecords := []models.Record{genRecords[3], genRecords[2]}

	limit := 2
	index := fmt.Sprintf("%d:%d", pulse.PulseNumber, genRecords[3].Order)
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, limit, 0, false)
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_IndexOffset(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.ID()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, objRef, 5)

	expectedRecords := []models.Record{genRecords[1], genRecords[0]}

	offset := 2
	index := fmt.Sprintf("%d:%d", pulse.PulseNumber, genRecords[3].Order)
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, offset, false)
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_IndexLimit_Asc(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.ID()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, objRef, 5)

	expectedRecords := []models.Record{genRecords[2], genRecords[3]}

	limit := 2
	index := fmt.Sprintf("%d:%d", pulse.PulseNumber, genRecords[2].Order)
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, limit, 0, true)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_IndexOffset_Asc(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop)
	require.NoError(t, err)

	objRef := gen.ID()

	genRecords := testutils.OrderedRecords(t, testDB, jetDrop, objRef, 5)

	expectedRecords := []models.Record{genRecords[3], genRecords[4]}

	index := fmt.Sprintf("%d:%d", pulse.PulseNumber, genRecords[1].Order)
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, 2, true)
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_TimestampRange(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 3, 2)

	expectedRecords := []models.Record{pulses[1].records[1], pulses[1].records[0], pulses[0].records[1], pulses[0].records[0]}

	timestampLte := int(pulses[1].pulse.Timestamp)
	timestampGte := int(pulses[0].pulse.Timestamp)
	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		nil, nil, &timestampLte, &timestampGte, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_PulseRange(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 3, 3)

	expectedRecords := []models.Record{pulses[1].records[2], pulses[1].records[1], pulses[1].records[0]}

	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_PulseRange_SamePulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 3, 3)

	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		&pulses[2].pulse.PulseNumber, &pulses[2].pulse.PulseNumber, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, records)
}

func TestStorage_GetLifeline_PulseRange_Limit(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 3, 3)

	expectedRecords := []models.Record{pulses[1].records[2], pulses[1].records[1]}

	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 2, 0, false)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_PulseRange_LimitOffset(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 3, 4)

	expectedRecords := []models.Record{pulses[1].records[2], pulses[1].records[1]}

	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 2, 1, false)
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_PulseRange_Limit_Asc(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 3, 3)

	expectedRecords := []models.Record{pulses[1].records[0], pulses[1].records[1]}

	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 2, 0, true)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_Index_PulseRange(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 4, 3)

	expectedRecords := []models.Record{pulses[2].records[1], pulses[2].records[0], pulses[1].records[2], pulses[1].records[1], pulses[1].records[0]}

	index := fmt.Sprintf("%d:%d", pulses[2].pulse.PulseNumber, pulses[2].records[1].Order)
	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, &index,
		&pulses[3].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_AllParams(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 4, 3)

	expectedRecords := []models.Record{pulses[2].records[0], pulses[1].records[2], pulses[1].records[1]}

	index := fmt.Sprintf("%d:%d", pulses[2].pulse.PulseNumber, pulses[2].records[1].Order)
	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, &index,
		&pulses[3].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 3, 1, false)
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_PulseRange_Empty(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulses := pulseSequenceOneObject(t, 4, 3)

	records, total, err := s.GetLifeline(
		pulses[1].records[0].ObjectReference, nil,
		&pulses[0].pulse.PulseNumber, &pulses[3].pulse.PulseNumber, nil, nil, 20, 0, false)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, records)
}

func TestStorage_GetPulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	expectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, expectedPulse)
	require.NoError(t, err)
	notExpectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, notExpectedPulse)
	require.NoError(t, err)

	pulse, jetDropAmount, recordAmount, err := s.GetPulse(expectedPulse.PulseNumber)
	require.NoError(t, err)
	require.Equal(t, expectedPulse, pulse)
	require.EqualValues(t, 0, jetDropAmount)
	require.EqualValues(t, 0, recordAmount)
}

func TestStorage_GetPulse_PulseWithDifferentNext(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	expectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, expectedPulse)
	require.NoError(t, err)
	nextPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	nextPulse.PulseNumber = expectedPulse.PulseNumber + 200
	nextPulse.PrevPulseNumber = expectedPulse.PulseNumber
	err = testutils.CreatePulse(testDB, nextPulse)
	require.NoError(t, err)

	pulse, jetDropAmount, recordAmount, err := s.GetPulse(expectedPulse.PulseNumber)
	require.NoError(t, err)
	expectedPulse.NextPulseNumber = nextPulse.PulseNumber
	require.Equal(t, expectedPulse, pulse)
	require.EqualValues(t, 0, jetDropAmount)
	require.EqualValues(t, 0, recordAmount)
}

func TestStorage_GetPulse_PulseWithRecords(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	expectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, expectedPulse)
	require.NoError(t, err)
	jetDrop1 := testutils.InitJetDropDB(expectedPulse)
	jetDrop1.RecordAmount = 10
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	jetDrop2 := testutils.InitJetDropDB(expectedPulse)
	jetDrop2.RecordAmount = 25
	err = testutils.CreateJetDrop(testDB, jetDrop2)
	require.NoError(t, err)

	pulse, jetDropAmount, recordAmount, err := s.GetPulse(expectedPulse.PulseNumber)
	require.NoError(t, err)
	require.Equal(t, expectedPulse, pulse)
	require.EqualValues(t, 2, jetDropAmount)
	require.EqualValues(t, jetDrop1.RecordAmount+jetDrop2.RecordAmount, recordAmount)
}

func TestStorage_GetPulse_NotExist(t *testing.T) {
	s := NewStorage(testDB)

	_, _, _, err := s.GetPulse(int(gen.PulseNumber()))
	require.Error(t, err)
	require.True(t, gorm.IsRecordNotFoundError(err))
}

func TestStorage_GetPulses(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, nil, nil, 100, 0)
	require.NoError(t, err)
	require.Len(t, pulses, 2)
	require.Contains(t, pulses, firstPulse)
	require.Contains(t, pulses, secondPulse)
	require.EqualValues(t, 2, total)
}

func TestStorage_GetPulses_Limit(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PulseNumber: 66666666,
		IsComplete:  false,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: 66666667,
		IsComplete:  false,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: 66666668,
		IsComplete:  false,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, nil, nil, 2, 0)
	require.NoError(t, err)
	require.Equal(t, []models.Pulse{thirdPulse, secondPulse}, pulses)
	require.EqualValues(t, 3, total)
}

func TestStorage_GetPulses_Offset(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PulseNumber: 66666666,
		IsComplete:  false,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: 66666667,
		IsComplete:  false,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: 66666668,
		IsComplete:  false,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, nil, nil, 100, 1)
	require.NoError(t, err)
	require.Equal(t, []models.Pulse{secondPulse, firstPulse}, pulses)
	require.EqualValues(t, 3, total)
}

func TestStorage_GetPulses_TimestampRange(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PulseNumber: 66666666,
		IsComplete:  false,
		Timestamp:   66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: 66666667,
		IsComplete:  false,
		Timestamp:   66666667,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: 66666668,
		IsComplete:  false,
		Timestamp:   66666668,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	fourthPulse := models.Pulse{
		PulseNumber: 66666669,
		IsComplete:  false,
		Timestamp:   66666669,
	}
	err = testutils.CreatePulse(testDB, fourthPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, &thirdPulse.PulseNumber, &secondPulse.PulseNumber, 100, 0)
	require.NoError(t, err)
	require.Equal(t, []models.Pulse{thirdPulse, secondPulse}, pulses)
	require.EqualValues(t, 2, total)
}

func TestStorage_GetPulses_FromPulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PulseNumber: 66666666,
		IsComplete:  false,
		Timestamp:   66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: 66666667,
		IsComplete:  false,
		Timestamp:   66666667,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: 66666668,
		IsComplete:  false,
		Timestamp:   66666668,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	fromPulse := int64(secondPulse.PulseNumber)
	pulses, total, err := s.GetPulses(&fromPulse, nil, nil, 100, 0)
	require.NoError(t, err)
	require.Equal(t, []models.Pulse{secondPulse, firstPulse}, pulses)
	require.EqualValues(t, 2, total)
}

func TestStorage_GetPulses_AllParams(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PulseNumber: 66666666,
		IsComplete:  false,
		Timestamp:   66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PulseNumber: 66666667,
		IsComplete:  false,
		Timestamp:   66666667,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PulseNumber: 66666668,
		IsComplete:  false,
		Timestamp:   66666668,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	fourthPulse := models.Pulse{
		PulseNumber: 66666669,
		IsComplete:  false,
		Timestamp:   66666669,
	}
	err = testutils.CreatePulse(testDB, fourthPulse)
	require.NoError(t, err)

	fromPulse := int64(thirdPulse.PulseNumber)
	pulses, total, err := s.GetPulses(&fromPulse, &fourthPulse.PulseNumber, &secondPulse.PulseNumber, 1, 1)
	require.NoError(t, err)
	require.Equal(t, []models.Pulse{secondPulse}, pulses)
	require.EqualValues(t, 2, total)
}

func TestStorage_GetPulses_DifferentNextAtLastPulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PrevPulseNumber: 66666665,
		PulseNumber:     66666666,
		NextPulseNumber: 66666667,
		IsComplete:      false,
		Timestamp:       66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	// pulsar was down, next pulse is not as expected
	secondPulse := models.Pulse{
		PrevPulseNumber: 66666666,
		PulseNumber:     66666670,
		NextPulseNumber: 66666671,
		IsComplete:      false,
		Timestamp:       66666670,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	thirdPulse := models.Pulse{
		PrevPulseNumber: 66666670,
		PulseNumber:     66666671,
		NextPulseNumber: 66666672,
		IsComplete:      false,
		Timestamp:       66666671,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, nil, nil, 100, 0)
	require.NoError(t, err)
	require.Len(t, pulses, 3)

	// check pulses chain% 3<-2<-1
	require.Equal(t, thirdPulse.PulseNumber, pulses[0].PulseNumber)
	require.Equal(t, thirdPulse.NextPulseNumber, pulses[0].NextPulseNumber)

	require.Equal(t, secondPulse.PulseNumber, pulses[1].PulseNumber)
	require.Equal(t, thirdPulse.PulseNumber, pulses[1].NextPulseNumber)
	require.Equal(t, firstPulse.PulseNumber, pulses[1].PrevPulseNumber)

	require.Equal(t, firstPulse.PulseNumber, pulses[2].PulseNumber)
	require.Equal(t, secondPulse.PulseNumber, pulses[2].NextPulseNumber)

	require.EqualValues(t, 3, total)
}

func TestStorage_GetPulses_MissingData_DifferentNext(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PrevPulseNumber: 66666665,
		PulseNumber:     66666666,
		NextPulseNumber: 66666667,
		IsComplete:      false,
		Timestamp:       66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PrevPulseNumber: 66666666,
		PulseNumber:     66666667,
		NextPulseNumber: 66666668,
		IsComplete:      false,
		Timestamp:       66666667,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	// pulsar was down, next pulse is not as expected
	thirdPulse := models.Pulse{
		PrevPulseNumber: 66666667,
		PulseNumber:     66666680,
		NextPulseNumber: 66666681,
		IsComplete:      false,
		Timestamp:       66666680,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, nil, nil, 100, 0)
	require.NoError(t, err)
	require.Len(t, pulses, 3)

	// check pulses chain: 3<-2<-1
	require.Equal(t, thirdPulse.PulseNumber, pulses[0].PulseNumber)
	require.Equal(t, thirdPulse.NextPulseNumber, pulses[0].NextPulseNumber)

	require.Equal(t, secondPulse.PulseNumber, pulses[1].PulseNumber)
	require.Equal(t, thirdPulse.PulseNumber, pulses[1].NextPulseNumber)
	require.Equal(t, firstPulse.PulseNumber, pulses[1].PrevPulseNumber)

	require.Equal(t, firstPulse.PulseNumber, pulses[2].PulseNumber)
	require.Equal(t, secondPulse.PulseNumber, pulses[2].NextPulseNumber)

	require.EqualValues(t, 3, total)
}

func TestStorage_GetPulses_MissingData_DifferentNextInTop(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	firstPulse := models.Pulse{
		PrevPulseNumber: 66666665,
		PulseNumber:     66666666,
		NextPulseNumber: 66666667,
		IsComplete:      false,
		Timestamp:       66666666,
	}
	err := testutils.CreatePulse(testDB, firstPulse)
	require.NoError(t, err)

	secondPulse := models.Pulse{
		PrevPulseNumber: 66666666,
		PulseNumber:     66666667,
		NextPulseNumber: 66666668,
		IsComplete:      false,
		Timestamp:       66666667,
	}
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	// pulsar was down, next pulse is not as expected
	thirdPulse := models.Pulse{
		PrevPulseNumber: 66666667,
		PulseNumber:     66666680,
		NextPulseNumber: 66666681,
		IsComplete:      false,
		Timestamp:       66666680,
	}
	err = testutils.CreatePulse(testDB, thirdPulse)
	require.NoError(t, err)

	pulses, total, err := s.GetPulses(nil, nil, nil, 100, 1)
	require.NoError(t, err)
	require.Len(t, pulses, 2)

	// check pulses chain: 3<-2<-1, but we get only 2 and 1 in result
	require.Equal(t, secondPulse.PulseNumber, pulses[0].PulseNumber)
	require.Equal(t, thirdPulse.PulseNumber, pulses[0].NextPulseNumber)
	require.Equal(t, firstPulse.PulseNumber, pulses[0].PrevPulseNumber)

	require.Equal(t, firstPulse.PulseNumber, pulses[1].PulseNumber)
	require.Equal(t, secondPulse.PulseNumber, pulses[1].NextPulseNumber)

	require.EqualValues(t, 3, total)
}

func TestStorage_GetRecordsByJetDrop(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop1 := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	require.NoError(t, err)
	recordResult := testutils.InitRecordDB(jetDrop1)
	recordResult.Type = models.Result
	recordResult.Order = 1
	err = testutils.CreateRecord(testDB, recordResult)
	require.NoError(t, err)
	recordState1 := testutils.InitRecordDB(jetDrop1)
	recordState1.Order = 2
	err = testutils.CreateRecord(testDB, recordState1)
	require.NoError(t, err)
	recordState2 := testutils.InitRecordDB(jetDrop1)
	recordState2.Order = 3
	err = testutils.CreateRecord(testDB, recordState2)
	require.NoError(t, err)

	jetDrop2 := testutils.InitJetDropDB(pulse)
	err = testutils.CreateJetDrop(testDB, jetDrop2)
	require.NoError(t, err)
	err = testutils.CreateRecord(testDB, testutils.InitRecordDB(jetDrop2))
	require.NoError(t, err)

	jetDropID := *models.NewJetDropID(jetDrop1.JetID, int64(pulse.PulseNumber))
	t.Run("happy", func(t *testing.T) {
		records, total, err := s.GetRecordsByJetDrop(jetDropID, nil, nil, 1000, 0)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, records, 3)
		require.Contains(t, records, recordResult)
		require.Contains(t, records, recordState1)
		require.Contains(t, records, recordState2)
	})

	t.Run("type", func(t *testing.T) {
		recType := string(models.Result)
		records, total, err := s.GetRecordsByJetDrop(jetDropID, nil, &recType, 1000, 0)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Equal(t, []models.Record{recordResult}, records)
	})

	t.Run("limit", func(t *testing.T) {
		records, total, err := s.GetRecordsByJetDrop(jetDropID, nil, nil, 2, 0)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, records, 2)
		require.Contains(t, records, recordResult)
		require.Contains(t, records, recordState1)
	})

	t.Run("offset", func(t *testing.T) {
		records, total, err := s.GetRecordsByJetDrop(jetDropID, nil, nil, 1000, 1)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, records, 2)
		require.Contains(t, records, recordState1)
		require.Contains(t, records, recordState2)
	})

	t.Run("from_index", func(t *testing.T) {
		index := fmt.Sprintf("%d:%d", pulse.PulseNumber, recordState1.Order)
		records, total, err := s.GetRecordsByJetDrop(jetDropID, &index, nil, 1000, 0)
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Len(t, records, 2)
		require.Contains(t, records, recordState1)
		require.Contains(t, records, recordState2)
	})

	t.Run("empty", func(t *testing.T) {
		jetDropEmpty := testutils.InitJetDropDB(pulse)
		jetDropIDEmpty := *models.NewJetDropID(jetDropEmpty.JetID, int64(pulse.PulseNumber))
		records, total, err := s.GetRecordsByJetDrop(jetDropIDEmpty, nil, nil, 1000, 0)
		require.NoError(t, err)
		require.Equal(t, 0, total)
		require.Empty(t, records)
	})
}

func TestStorage_GetPulseByPrev(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	prevPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, prevPulse)
	require.NoError(t, err)
	expectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	expectedPulse.PrevPulseNumber = prevPulse.PulseNumber
	err = testutils.CreatePulse(testDB, expectedPulse)
	require.NoError(t, err)
	notExpectedPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, notExpectedPulse)
	require.NoError(t, err)

	pulse, err := s.GetPulseByPrev(prevPulse)
	require.NoError(t, err)
	require.Equal(t, expectedPulse, pulse)
}

func TestStorage_GetPulseByPrev_NotExistError(t *testing.T) {
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	_, err = s.GetPulseByPrev(models.Pulse{PrevPulseNumber: pulse.PulseNumber})
	require.Error(t, err)
}

func TestStorage_GetSequentialPulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	sequentialPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	sequentialPulse.IsSequential = true
	err = testutils.CreatePulse(testDB, sequentialPulse)
	require.NoError(t, err)

	lessSequentialPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	lessSequentialPulse.IsSequential = true
	lessSequentialPulse.PulseNumber = sequentialPulse.PulseNumber - 10
	err = testutils.CreatePulse(testDB, lessSequentialPulse)
	require.NoError(t, err)

	notSequentialPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, notSequentialPulse)
	require.NoError(t, err)

	pulse, err := s.GetSequentialPulse()
	require.NoError(t, err)
	require.Equal(t, sequentialPulse, pulse)
}

func TestStorage_GetSequentialPulse_Empty(t *testing.T) {
	s := NewStorage(testDB)

	sequentialPulse, err := s.GetSequentialPulse()
	require.NoError(t, err)
	require.Equal(t, models.Pulse{}, sequentialPulse)
}

func TestStorage_GetNextSavedPulse(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	pulse := models.Pulse{
		PulseNumber: int(gen.PulseNumber().AsUint32()),
	}
	expectedPulse := models.Pulse{
		PulseNumber: pulse.PulseNumber + 10,
	}
	notExpectedPulse := models.Pulse{
		PulseNumber: pulse.PulseNumber + 20,
	}

	err := testutils.CreatePulse(testDB, notExpectedPulse)
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, expectedPulse)
	require.NoError(t, err)

	res, err := s.GetNextSavedPulse(pulse)
	require.NoError(t, err)
	require.Equal(t, expectedPulse, res)
}

func TestStorage_GetNextSavedPulse_Empty(t *testing.T) {
	s := NewStorage(testDB)

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)

	sequentialPulse, err := s.GetNextSavedPulse(pulse)
	require.NoError(t, err)
	require.Equal(t, models.Pulse{}, sequentialPulse)
}

func TestStorage_GetJetDropsByJetId_Success(t *testing.T) {
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

	jID := jetDropForFirstPulse1.JetID
	jetDrops, total, err := s.GetJetDropsByJetID(jID, nil, nil, nil, nil, -1, true)
	require.NoError(t, err)
	require.Len(t, jetDrops, 1)
	require.EqualValues(t, jetDropForFirstPulse1, jetDrops[0])
	require.Equal(t, 1, total)
}

func TestStorage_GetJetDropsByJetId_Fail(t *testing.T) {
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

	wrongJetID := jetDropForSecondPulse.JetID
	jetDrops, total, err := s.GetJetDropsByJetID(wrongJetID, nil, nil, nil, nil, -1, true)
	require.NoError(t, err)
	require.Len(t, jetDrops, 0)
	require.Equal(t, 0, total)
}

func TestStorage_GetJetDropsByJetId(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	someJetIDCount := 5
	someJetId, preparedJetDrops, preparedPulses := testutils.GenerateJetDropsWithSomeJetID(t, someJetIDCount)
	err := testutils.CreatePulses(testDB, preparedPulses)
	require.NoError(t, err)
	err = testutils.CreateJetDrops(testDB, preparedJetDrops)
	require.NoError(t, err)

	t.Run("limit", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, nil, nil, 1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, 1)
		require.Equal(t, someJetIDCount, total)
	})
	t.Run("pulseNumberLte", func(t *testing.T) {
		expectedCount := 2
		pulseNumberLte := preparedPulses[1].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, &pulseNumberLte, nil, nil, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i := 0; i < expectedCount; i++ {
			expected := preparedJetDrops[i]
			received := jetDrops[i]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberLte-all", func(t *testing.T) {
		expectedCount := someJetIDCount
		pulseNumberLte := preparedPulses[someJetIDCount-1].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, &pulseNumberLte, nil, nil, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i := 0; i < expectedCount; i++ {
			expected := preparedJetDrops[i]
			received := jetDrops[i]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberLt", func(t *testing.T) {
		expectedCount := 1
		pulseNumberLt := preparedPulses[1].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, &pulseNumberLt, nil, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i := 0; i < expectedCount; i++ {
			expected := preparedJetDrops[i]
			received := jetDrops[i]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberLt-no-one", func(t *testing.T) {
		expectedCount := 0
		pulseNumberLt := preparedPulses[0].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, &pulseNumberLt, nil, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
	})
	t.Run("pulseNumberGte", func(t *testing.T) {
		expectedCount := someJetIDCount - 1
		pulseNumberGte := preparedPulses[1].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, &pulseNumberGte, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := preparedJetDrops[i]
			received := jetDrops[j]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberGte-all", func(t *testing.T) {
		expectedCount := someJetIDCount
		pulseNumberGt := preparedPulses[0].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, &pulseNumberGt, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i := 0; i < expectedCount; i++ {
			expected := preparedJetDrops[i]
			received := jetDrops[i]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberGt", func(t *testing.T) {
		expectedCount := someJetIDCount - 2
		pulseNumberGt := preparedPulses[1].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, nil, &pulseNumberGt, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i, j := 2, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := preparedJetDrops[i]
			received := jetDrops[j]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberGt-no-one", func(t *testing.T) {
		expectedCount := 0
		pulseNumberGt := preparedPulses[someJetIDCount-1].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, nil, &pulseNumberGt, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
	})
	t.Run("pulseNumberGte and pulseNumberLte", func(t *testing.T) {
		expectedCount := someJetIDCount - 2
		pulseNumberGte := preparedPulses[1].PulseNumber
		pulseNumberLte := preparedPulses[someJetIDCount-2].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, &pulseNumberLte, nil, &pulseNumberGte, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := preparedJetDrops[i]
			received := jetDrops[j]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberGte and pulseNumberLt", func(t *testing.T) {
		expectedCount := someJetIDCount - 3
		pulseNumberGte := preparedPulses[1].PulseNumber
		pulseNumberLt := preparedPulses[someJetIDCount-2].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, &pulseNumberLt, &pulseNumberGte, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := preparedJetDrops[i]
			received := jetDrops[j]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberGt and pulseNumberLt", func(t *testing.T) {
		expectedCount := someJetIDCount - 4
		pulseNumberGt := preparedPulses[1].PulseNumber
		pulseNumberLt := preparedPulses[someJetIDCount-2].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, &pulseNumberLt, nil, &pulseNumberGt, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i, j := 1, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := preparedJetDrops[i]
			received := jetDrops[j]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("pulseNumberGt and pulseNumberLte", func(t *testing.T) {
		expectedCount := someJetIDCount - 3
		pulseNumberGt := preparedPulses[1].PulseNumber
		pulseNumberLte := preparedPulses[someJetIDCount-2].PulseNumber
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, &pulseNumberLte, nil, nil, &pulseNumberGt, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, expectedCount)
		require.Equal(t, expectedCount, total)
		for i, j := 2, 0; i < expectedCount; i, j = i+1, j+1 {
			expected := preparedJetDrops[i]
			received := jetDrops[j]
			require.EqualValues(t, expected, received)
		}
	})
	t.Run("sortBy asc", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, nil, nil, -1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, total)
		require.Equal(t, someJetIDCount, total)
		for i, drop := range jetDrops {
			require.EqualValues(t, preparedJetDrops[i], drop)
		}
	})
	t.Run("sortBy desc", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, nil, nil, -1, false)
		require.NoError(t, err)
		require.Len(t, jetDrops, total)
		require.Equal(t, someJetIDCount, total)
		for i, drop := range jetDrops {
			require.EqualValues(t, preparedJetDrops[total-i-1], drop)
		}
	})
}

func TestStorage_GetJetDropsByJetId_Splites(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	// fnGetJetIDs returns the jetid with middle depth and all possible values fot that
	// if we pass the jetid, then we expect that the db must contains all possible values
	fnGetJetIDs := func(drops []models.JetDrop) (string, []string) {
		deepest := drops[0].JetID
		depthless := drops[0].JetID
		middle := drops[0].JetID

		for i := 1; i < len(drops); i++ {
			id := drops[i].JetID
			if len(id) > len(deepest) {
				deepest = id
			}
			if len(id) < len(depthless) {
				depthless = id
			}
		}
		start := len(depthless)
		end := len(deepest)
		median := start + (end-start)/2

		// try to find the middle
		for i := 0; i < len(drops); i++ {
			if len(drops[i].JetID) == median {
				middle = drops[i].JetID
			}
		}

		// find parents of middle
		parents := GetJetIDParents(middle)
		childrenRegexp := regexp.MustCompile(fmt.Sprintf("^%s.*", middle))
		for i := 0; i < len(drops); i++ {
			id := drops[i].JetID
			if childrenRegexp.MatchString(id) && id != middle { // if it's child
				parents = append(parents, id)
			}
		}

		// true if array contains value
		contains := func(data []models.JetDrop, find string) bool {
			for _, v := range data {
				if v.JetID == find {
					return true
				}
			}
			return false
		}

		// try to calculate all possible values
		allPossible := make([]string, 0)
		// delete non existing jetid
		for _, id := range parents {
			// if incoming jet drops contains generated values
			if contains(drops, id) {
				allPossible = append(allPossible, id)
			}
		}

		return middle, allPossible
	}

	tests := map[string]struct {
		pulseCount int
		jDCount    int
		depth      int
		total      int
	}{
		"pc=1, jdc=1, depth=0, total=1":     {1, 1, 0, 1},
		"pc=1, jdc=1, depth=1, total=3":     {1, 1, 1, 3},
		"pc=2, jdc=1, depth=1, total=6":     {2, 1, 1, 6},
		"pc=1, jdc=2, depth=2, total=14":    {1, 2, 2, 14},
		"pc=2, jdc=2, depth=2, total=28":    {2, 2, 2, 28},
		"pc=2, jdc=2, depth=4, total=124":   {2, 2, 4, 124},
		"pc=4, jdc=10, depth=5, total=2520": {4, 10, 5, 2520},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
			pulseCount := test.pulseCount
			jetDropCount := test.jDCount
			depth := test.depth
			total := test.total

			preparedJetDrops, preparedPulses := testutils.GenerateJetDropsWithSplit(t, pulseCount, jetDropCount, depth)
			require.Equal(t, total, len(preparedJetDrops))
			err := testutils.CreatePulses(testDB, preparedPulses)
			require.NoError(t, err)
			err = testutils.CreateJetDrops(testDB, preparedJetDrops)
			require.NoError(t, err)

			// try to calculate all possible Jet IDs
			middle, allPossible := fnGetJetIDs(preparedJetDrops)

			jetDropsFromDb, totalFromDb, err := s.GetJetDropsByJetID(middle, nil, nil, nil, nil, -1, true)
			require.NoError(t, err)

			require.Equal(t, len(allPossible), totalFromDb)
			for _, v := range jetDropsFromDb {
				require.Contains(t, allPossible, v.JetID)
			}

		})
	}

}

func TestStorage_GetJetDropsByJetId_MultipleCounts(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	tests := map[string]struct {
		jetDropCount int
		limit        int
	}{
		"no jetDrop with limit 0":   {jetDropCount: 1, limit: 0},
		"one jetDrop with limit 1":  {jetDropCount: 1, limit: 1},
		"two jetDrop with limit 10": {jetDropCount: 2, limit: 10},
		"10 jetDrop with limit 10":  {jetDropCount: 10, limit: 10},
		"15 jetDrop with limit 10":  {jetDropCount: 15, limit: 10},
	}

	for testName, data := range tests {
		t.Run(testName, func(t *testing.T) {
			someJetId, preparedJetDrops, preparedPulses := testutils.GenerateJetDropsWithSomeJetID(t, data.jetDropCount)
			err := testutils.CreatePulses(testDB, preparedPulses)
			require.NoError(t, err)
			err = testutils.CreateJetDrops(testDB, preparedJetDrops)
			require.NoError(t, err)

			jetDropsFromDb, total, err := s.GetJetDropsByJetID(someJetId, nil, nil, nil, nil, data.limit, true)
			require.NoError(t, err)
			expectedCount := data.jetDropCount
			if expectedCount > data.limit {
				expectedCount = data.limit
			}
			require.Len(t, jetDropsFromDb, expectedCount)
			for i := 0; i < expectedCount; i++ {
				require.Contains(t, preparedJetDrops, jetDropsFromDb[i])
			}
			require.Equal(t, data.jetDropCount, total)
		})
	}
}
