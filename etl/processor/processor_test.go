// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package processor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/insolar/block-explorer/etl/models"

	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/types"
)

func TestNewProcessor(t *testing.T) {

	ctx := context.Background()
	JDC := make(chan types.JetDrop)
	trm := mock.NewTransformerMock(t)
	trm.GetJetDropsChannelMock.Set(func() (ch1 <-chan types.JetDrop) {
		return JDC
	})

	wg := sync.WaitGroup{}
	saves := int32(0)
	sm := mock.NewStorageSetterMock(t)
	sm.SaveJetDropDataMock.Set(func(jetDrop models.JetDrop, records []models.Record) (err error) {
		atomic.AddInt32(&saves, 1)
		wg.Done()
		return nil
	})

	p := NewProcessor(trm, sm, 3)
	require.NotNil(t, p)

	require.NoError(t, p.Start(ctx))

	wg.Add(5)
	for i := 0; i < 5; i++ {
		JDC <- types.JetDrop{
			MainSection: &types.MainSection{
				Start: types.DropStart{
					PulseData:           types.Pulse{},
					JetDropPrefix:       nil,
					JetDropPrefixLength: 0,
				},
				DropContinue: types.DropContinue{},
				Sections:     nil,
				Records:      nil,
			},
			Sections: nil,
			RawData:  nil,
		}
	}

	wg.Wait()
	require.NoError(t, p.Stop(ctx))
	require.Equal(t, int32(5), atomic.LoadInt32(&saves))
}
