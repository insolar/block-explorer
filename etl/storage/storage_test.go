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
	"testing"

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

	records, total, err := s.GetLifeline(objRef.Bytes(), nil, nil, nil, nil, nil, 20, 0, "-index")
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_ObjNotExist(t *testing.T) {
	s := NewStorage(testDB)

	records, total, err := s.GetLifeline(gen.Reference().Bytes(), nil, nil, nil, nil, nil, 20, 0, "-index")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, 0, "-index")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, 0, "-index")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, limit, 0, "-index")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, offset, "-index")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, limit, 0, "+index")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, nil, nil, 20, 2, "+index")
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
		nil, nil, &timestampLte, &timestampGte, 20, 0, "-index")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 20, 0, "-index")
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
		&pulses[2].pulse.PulseNumber, &pulses[2].pulse.PulseNumber, nil, nil, 20, 0, "-index")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 2, 0, "-index")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 2, 1, "-index")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 2, 0, "+index")
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
		&pulses[3].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 20, 0, "-index")
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
		&pulses[3].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, nil, nil, 3, 1, "-index")
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
		&pulses[0].pulse.PulseNumber, &pulses[3].pulse.PulseNumber, nil, nil, 20, 0, "-index")
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
	jetDropID := models.NewJetDropID(jetDropForFirstPulse1.JetID, int64(jetDropForFirstPulse1.PulseNumber))
	jetDrops, total, err := s.GetJetDropsByJetId(jID, jetDropID, nil, nil, -1, 0, true)
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
	fromJetID := models.NewJetDropID(jetDropForSecondPulse.JetID, int64(jetDropForFirstPulse1.PulseNumber))
	jetDrops, total, err := s.GetJetDropsByJetId(wrongJetID, fromJetID, nil, nil, -1, 0, true)
	require.NoError(t, err)
	require.Len(t, jetDrops, 0)
	require.Equal(t, 0, total)
}

func TestStorage_GetJetDropsByJetId2(t *testing.T) {
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

	someJetIDCount := 5
	someJetId, preparedJetDrops := generateJetDropsWithSomeJetID(t, someJetIDCount)
	for _, v := range preparedJetDrops {
		err := testutils.CreateJetDrop(testDB, v)
		require.NoError(t, err)
	}

	secondPulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, secondPulse)
	require.NoError(t, err)

	fromJetDropID := models.NewJetDropID(jetDropForFirstPulse1.JetID, int64(jetDropForFirstPulse1.PulseNumber))
	t.Run("limit", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsByJetId(someJetId, fromJetDropID, nil, nil, 1, 0, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, 1)
		require.Equal(t, someJetIDCount, total)
	})
	t.Run("offset", func(t *testing.T) {
		jetDrops, total, err := s.GetJetDropsByJetId(someJetId, fromJetDropID, nil, nil, -1, someJetIDCount-1, true)
		require.NoError(t, err)
		require.Len(t, jetDrops, 1)
		require.Equal(t, someJetIDCount, total)
	})
	t.Run("jetDropIDGte", func(t *testing.T) {

	})
	t.Run("jetDropIDLte", func(t *testing.T) {

	})
	t.Run("sortBy", func(t *testing.T) {

	})
}

func TestStorage_GetJetDropsByJetId_MultipleCounts(t *testing.T) {
	defer testutils.TruncateTables(t, testDB, []interface{}{models.Record{}, models.JetDrop{}, models.Pulse{}})
	s := NewStorage(testDB)

	tests := map[string]struct {
		jetDropCount int
		limit        int
		offset       int
	}{
		"one jetDrop by pulse": {jetDropCount: 1, limit: 1, offset: 0},
		"two jetDrop by pulse": {jetDropCount: 2, limit: 10, offset: 0},
		"10 jetDrop by pulse":  {jetDropCount: 10, limit: 10, offset: 0},
		"15 jetDrop by pulse":  {jetDropCount: 15, limit: 10, offset: 10},
	}

	for testName, data := range tests {
		t.Run(testName, func(t *testing.T) {
			jID, preparedJetDrops := generateJetDropsWithSomeJetID(t, data.jetDropCount)
			for _, v := range preparedJetDrops {
				err := testutils.CreateJetDrop(testDB, v)
				require.NoError(t, err)
			}

			fromJetDropID := models.NewJetDropID(jID, 1234)
			jetDropsFromDb, total, err := s.GetJetDropsByJetId(jID, fromJetDropID, nil, nil, data.limit, data.offset, true)
			require.NoError(t, err)
			expectedCount := data.jetDropCount - data.offset
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

func generateJetDropsWithSomeJetID(t *testing.T, jCount int) ([]byte, []models.JetDrop) {
	var jID *[]byte
	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)

	drops := make([]models.JetDrop, jCount)
	jDrop := testutils.InitJetDropDB(pulse)
	drops[0] = jDrop
	if jID == nil {
		jID = &jDrop.JetID
	}
	for i := 1; i < jCount; i++ {
		pulse, err := testutils.InitPulseDB()
		require.NoError(t, err)
		err = testutils.CreatePulse(testDB, pulse)
		require.NoError(t, err)
		jd := testutils.InitJetDropDB(pulse)
		jd.JetID = *jID
		drops[i] = jd
	}
	return *jID, drops
}
