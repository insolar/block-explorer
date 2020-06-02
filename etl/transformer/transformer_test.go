// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package transformer

import (
	"bytes"
	"testing"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/testutils"
)

func TestRestoreInsolarID(t *testing.T) {
	id := gen.ID()
	restored := restoreInsolarID(id.Bytes())
	require.Equal(t, id.String(), restored)
}

func TestInitRecordsMapsByObj(t *testing.T) {
	expectedNotState := testutils.CreateRecordCanonical()
	expectedNotState.Type = types.REQUEST
	activateRecord := testutils.CreateRecordCanonical()
	activateRecord.PrevRecordReference = make([]byte, 0)
	input := []types.Record{
		testutils.CreateRecordCanonical(),
		activateRecord,
		testutils.CreateRecordCanonical(),
		expectedNotState,
		testutils.CreateRecordCanonical(),
	}
	byPrevRef, byRef, notState := initRecordsMapsByObj(input)
	require.Equal(t, []types.Record{expectedNotState}, notState)
	require.Len(t, byRef, 4)
	require.Len(t, byPrevRef, 3)
}

func TestTransform_sortRecords(t *testing.T) {
	firstObj := gen.Reference().Bytes()
	secondObj := gen.Reference().Bytes()
	for bytes.Equal(firstObj, secondObj) {
		secondObj = gen.Reference().Bytes()
	}
	record1 := testutils.CreateRecordCanonical()
	record1.ObjectReference = firstObj
	record2 := testutils.CreateRecordCanonical()
	record2.ObjectReference = secondObj
	record3 := testutils.CreateRecordCanonical()
	record3.ObjectReference = firstObj
	record4RequestType := testutils.CreateRecordCanonical()
	record4RequestType.Type = types.REQUEST
	record4RequestType.ObjectReference = firstObj
	record5 := testutils.CreateRecordCanonical()
	record5.ObjectReference = firstObj
	record6 := testutils.CreateRecordCanonical()
	record6.ObjectReference = firstObj

	// make lifeline: 5 <- 3 <- 6 <- 1
	record1.PrevRecordReference = record6.Ref
	record6.PrevRecordReference = record3.Ref
	record3.PrevRecordReference = record5.Ref

	// result can be (4, 5, 3, 6, 1, 2) or (4, 2, 5, 3, 6, 1)
	expectedResult1 := []types.Record{record4RequestType, record5, record3, record6, record1, record2}
	expectedResult2 := []types.Record{record4RequestType, record2, record5, record3, record6, record1}

	result, err := sortRecords([]types.Record{record1, record2, record3, record4RequestType, record5, record6})
	require.NoError(t, err)

	// result can be (4, 5, 3, 6, 1, 2) or (4, 2, 5, 3, 6, 1)
	require.True(t, assert.ObjectsAreEqual(expectedResult1, result) || assert.ObjectsAreEqual(expectedResult2, result))
}

func TestTransform_sortRecords_HeadPrevIsEmpty(t *testing.T) {
	firstObj := gen.Reference().Bytes()
	record1 := testutils.CreateRecordCanonical()
	record1.ObjectReference = firstObj
	record3 := testutils.CreateRecordCanonical()
	record3.ObjectReference = firstObj
	record4RequestType := testutils.CreateRecordCanonical()
	record4RequestType.Type = types.REQUEST
	record4RequestType.ObjectReference = firstObj
	record5 := testutils.CreateRecordCanonical()
	record5.ObjectReference = firstObj
	record6 := testutils.CreateRecordCanonical()
	record6.ObjectReference = firstObj

	// make lifeline: 5 <- 3 <- 6 <- 1
	record5.PrevRecordReference = make([]byte, 0)
	record1.PrevRecordReference = record6.Ref
	record6.PrevRecordReference = record3.Ref
	record3.PrevRecordReference = record5.Ref

	expectedResult := []types.Record{record4RequestType, record5, record3, record6, record1}

	result, err := sortRecords([]types.Record{record1, record3, record4RequestType, record5, record6})
	require.NoError(t, err)
	require.Equal(t, expectedResult, result)
}

func TestTransform_sortRecords_ErrorNoHead(t *testing.T) {
	firstObj := gen.Reference().Bytes()
	record1 := testutils.CreateRecordCanonical()
	record1.ObjectReference = firstObj
	record2 := testutils.CreateRecordCanonical()
	record2.ObjectReference = firstObj

	// make lifeline with cycle: 2 <- 1 <- 2
	record1.PrevRecordReference = record2.Ref
	record2.PrevRecordReference = record1.Ref
	result, err := sortRecords([]types.Record{record1, record2})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot find head record for object")
	require.Nil(t, result)
}

func TestTransform_sortRecords_ErrorNoPrevRecord(t *testing.T) {
	firstObj := gen.Reference().Bytes()
	record1 := testutils.CreateRecordCanonical()
	record1.ObjectReference = firstObj
	record3 := testutils.CreateRecordCanonical()
	record3.ObjectReference = firstObj
	record5 := testutils.CreateRecordCanonical()
	record5.ObjectReference = firstObj
	record6 := testutils.CreateRecordCanonical()
	record6.ObjectReference = firstObj

	// make lifeline: 5 <- 3 <- 6 <- 1
	record1.PrevRecordReference = record6.Ref
	record6.PrevRecordReference = record3.Ref
	record3.PrevRecordReference = record5.Ref

	// don't provide record3
	result, err := sortRecords([]types.Record{record1, record5, record6})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot find record with prev record")
	require.Nil(t, result)
}
