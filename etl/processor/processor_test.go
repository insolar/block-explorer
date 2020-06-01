// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package processor

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/testutils"

	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/types"
)

func TestNewProcessor(t *testing.T) {

	ctx := belogger.TestContext(t)
	JDC := make(chan *types.JetDrop)
	trm := mock.NewTransformerMock(t)
	trm.GetJetDropsChannelMock.Set(func() (ch1 <-chan *types.JetDrop) {
		return JDC
	})

	wgStorage := sync.WaitGroup{}
	saves := int32(0)
	sm := mock.NewStorageSetterMock(t)
	sm.SaveJetDropDataMock.Set(func(jetDrop models.JetDrop, records []models.Record) (err error) {
		atomic.AddInt32(&saves, 1)
		wgStorage.Done()
		return nil
	})

	wgController := sync.WaitGroup{}
	controllerCalls := int32(0)
	contr := mock.NewControllerMock(t)
	contr.SetJetDropDataMock.Set(func(pulse types.Pulse, jetID []byte) {
		atomic.AddInt32(&controllerCalls, 1)
		wgController.Done()
	})

	p := NewProcessor(trm, sm, contr, 3)
	require.NotNil(t, p)

	require.NoError(t, p.Start(ctx))

	wgStorage.Add(5)
	wgController.Add(5)
	for i := 0; i < 5; i++ {
		JDC <- &types.JetDrop{
			MainSection: &types.MainSection{
				Start: types.DropStart{
					PulseData:           types.Pulse{},
					JetDropPrefix:       nil,
					JetDropPrefixLength: 0,
				},
				DropContinue: types.DropContinue{},
				Records:      nil,
			},
			Sections: nil,
			RawData:  nil,
		}
	}

	wgStorage.Wait()
	wgController.Wait()
	require.NoError(t, p.Stop(ctx))
	require.Equal(t, int32(5), atomic.LoadInt32(&saves))
	require.Equal(t, int32(5), atomic.LoadInt32(&controllerCalls))
}

func TestProcessor_process_EmptyPrev(t *testing.T) {
	ctx := belogger.TestContext(t)
	jd := testutils.CreateJetDropCanonical(
		[]types.Record{
			testutils.CreateRecordCanonical(), testutils.CreateRecordCanonical(), testutils.CreateRecordCanonical(),
		},
	)
	trm := mock.NewTransformerMock(t)
	trm.GetJetDropsChannelMock.Return(nil)

	sm := mock.NewStorageSetterMock(t)
	sm.SaveJetDropDataMock.Set(func(jetDrop models.JetDrop, records []models.Record) (err error) {
		require.Equal(t, jd.MainSection.Start.JetDropPrefix, jetDrop.JetID)
		require.Len(t, records, len(jd.MainSection.Records))
		return nil
	})

	contr := mock.NewControllerMock(t)
	contr.SetJetDropDataMock.Set(func(pulse types.Pulse, jetID []byte) {
		require.Equal(t, jd.MainSection.Start.PulseData, pulse)
		require.Equal(t, jd.MainSection.Start.JetDropPrefix, jetID)
	})

	p := NewProcessor(trm, sm, contr, 3)
	require.NotNil(t, p)

	p.process(ctx, &jd)

	require.Equal(t, uint64(1), sm.SaveJetDropDataAfterCounter())
	require.Equal(t, uint64(1), contr.SetJetDropDataAfterCounter())
}

func TestProcessor_process_SeveralPrev(t *testing.T) {
	ctx := belogger.TestContext(t)
	jd := testutils.CreateJetDropCanonical(
		[]types.Record{
			testutils.CreateRecordCanonical(), testutils.CreateRecordCanonical(), testutils.CreateRecordCanonical(),
		},
	)
	jd.MainSection.DropContinue.PrevDropHash = [][]byte{testutils.GenerateRandBytes(), testutils.GenerateRandBytes()}

	trm := mock.NewTransformerMock(t)
	trm.GetJetDropsChannelMock.Return(nil)

	sm := mock.NewStorageSetterMock(t)
	sm.SaveJetDropDataMock.Set(func(jetDrop models.JetDrop, records []models.Record) (err error) {
		require.Equal(t, jd.MainSection.Start.JetDropPrefix, jetDrop.JetID)
		require.Equal(t, jd.MainSection.DropContinue.PrevDropHash[0], jetDrop.FirstPrevHash)
		require.Equal(t, jd.MainSection.DropContinue.PrevDropHash[1], jetDrop.SecondPrevHash)
		require.Len(t, records, len(jd.MainSection.Records))
		return nil
	})

	contr := mock.NewControllerMock(t)
	contr.SetJetDropDataMock.Set(func(pulse types.Pulse, jetID []byte) {
		require.Equal(t, jd.MainSection.Start.PulseData, pulse)
		require.Equal(t, jd.MainSection.Start.JetDropPrefix, jetID)
	})

	p := NewProcessor(trm, sm, contr, 3)
	require.NotNil(t, p)

	p.process(ctx, &jd)

	require.Equal(t, uint64(1), sm.SaveJetDropDataAfterCounter())
	require.Equal(t, uint64(1), contr.SetJetDropDataAfterCounter())
}

func TestProcessor_process_StorageErr(t *testing.T) {
	ctx := belogger.TestContext(t)
	jd := testutils.CreateJetDropCanonical(
		[]types.Record{
			testutils.CreateRecordCanonical(), testutils.CreateRecordCanonical(), testutils.CreateRecordCanonical(),
		},
	)
	jd.MainSection.DropContinue.PrevDropHash = [][]byte{testutils.GenerateRandBytes(), testutils.GenerateRandBytes()}

	trm := mock.NewTransformerMock(t)
	trm.GetJetDropsChannelMock.Return(nil)

	sm := mock.NewStorageSetterMock(t)
	sm.SaveJetDropDataMock.Set(func(jetDrop models.JetDrop, records []models.Record) (err error) {
		return errors.New("test error")
	})

	contr := mock.NewControllerMock(t)

	p := NewProcessor(trm, sm, contr, 3)
	require.NotNil(t, p)

	p.process(ctx, &jd)

	require.Equal(t, uint64(1), sm.SaveJetDropDataAfterCounter())
}
