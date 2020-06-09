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

type pulseRecords struct {
	records []models.Record
	pulse   models.Pulse
}

func pulseSequenceOneObject(t *testing.T, pulseAmount, recordsAmount int) []pulseRecords {
	var result []pulseRecords
	objRef := gen.ID()
	pulseNumber := gen.PulseNumber()
	for i := 0; i < pulseAmount; i++ {
		pulse := models.Pulse{PulseNumber: int(pulseNumber)}
		err := testutils.CreatePulse(testDB, pulse)
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

	records, total, err := s.GetLifeline(objRef.Bytes(), nil, nil, nil, 20, 0, "desc")
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, expectedRecords, records)
}

func TestStorage_GetLifeline_ObjNotExist(t *testing.T) {
	s := NewStorage(testDB)

	records, total, err := s.GetLifeline(gen.Reference().Bytes(), nil, nil, nil, 20, 0, "desc")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, 20, 0, "desc")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, 20, 0, "desc")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, limit, 0, "desc")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, 20, offset, "desc")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, limit, 0, "asc")
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
	records, total, err := s.GetLifeline(objRef.Bytes(), &index, nil, nil, 20, 2, "asc")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, 20, 0, "desc")
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
		&pulses[2].pulse.PulseNumber, &pulses[2].pulse.PulseNumber, 20, 0, "desc")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, 2, 0, "desc")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, 2, 1, "desc")
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
		&pulses[2].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, 2, 0, "asc")
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
		&pulses[3].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, 20, 0, "desc")
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
		&pulses[3].pulse.PulseNumber, &pulses[0].pulse.PulseNumber, 3, 1, "desc")
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
		&pulses[0].pulse.PulseNumber, &pulses[3].pulse.PulseNumber, 20, 0, "desc")
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, records)
}
