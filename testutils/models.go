// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"time"

	"github.com/insolar/insolar/insolar/gen"

	"github.com/insolar/block-explorer/etl/models"
)

func InitRecordDB() models.Record {
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
