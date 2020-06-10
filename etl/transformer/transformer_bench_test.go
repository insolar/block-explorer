// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build bench

package transformer

import (
	"bytes"
	"testing"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkTransformSort(b *testing.B) {
	for i := 0; i < b.N; i++ {
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

		b.StartTimer()
		result, err := sortRecords([]types.Record{record1, record2, record3, record4RequestType, record5, record6})
		require.NoError(b, err)
		require.True(b, assert.ObjectsAreEqual(expectedResult1, result) || assert.ObjectsAreEqual(expectedResult2, result))
		b.StopTimer()
	}
}
