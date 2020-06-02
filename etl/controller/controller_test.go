// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package controller

import (
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/types"
)

func TestNewController_NoPulses(t *testing.T) {
	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)

	c, err := NewController(extractor, sm)
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Empty(t, c.jetDropRegister)
	require.Empty(t, c.missedDataRequestsQueue)
	require.Equal(t, uint64(1), sm.GetIncompletePulsesAfterCounter())
}

func TestNewController_OneNotCompletePulse(t *testing.T) {
	extractor := mock.NewJetDropsExtractorMock(t)

	pulseNumber := 1
	firstJetID := []byte{1, 2, 3}
	secondJetID := []byte{3, 4, 5}
	expectedData := map[types.Pulse][][]byte{{PulseNo: pulseNumber}: {firstJetID, secondJetID}}

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return([]models.Pulse{{PulseNumber: pulseNumber}}, nil)
	sm.GetJetDropsMock.Return([]models.JetDrop{{JetID: firstJetID}, {JetID: secondJetID}}, nil)

	c, err := NewController(extractor, sm)
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Empty(t, c.missedDataRequestsQueue)

	require.Equal(t, expectedData, c.jetDropRegister)

	require.Equal(t, uint64(1), sm.GetIncompletePulsesAfterCounter())
	require.Equal(t, uint64(1), sm.GetJetDropsAfterCounter())
}

func TestNewController_SeveralNotCompletePulses(t *testing.T) {
	extractor := mock.NewJetDropsExtractorMock(t)

	firstPulseNumber := 1
	secondPulseNumber := 2
	firstJetID := []byte{1, 2, 3}
	secondJetID := []byte{3, 4, 5}
	firstPulse := types.Pulse{PulseNo: firstPulseNumber}
	secondPulse := types.Pulse{PulseNo: secondPulseNumber}
	expectedData := map[types.Pulse][][]byte{firstPulse: {firstJetID}, secondPulse: {secondJetID}}

	sm := mock.NewStorageMock(t)
	getJetDrops := func(pulse models.Pulse) (ja1 []models.JetDrop, err error) {
		jd := make([]models.JetDrop, 0)
		switch pulse.PulseNumber {
		case 1:
			jd = append(jd, models.JetDrop{JetID: firstJetID})
		case 2:
			jd = append(jd, models.JetDrop{JetID: secondJetID})
		}
		return jd, nil
	}
	sm.GetIncompletePulsesMock.Return([]models.Pulse{{PulseNumber: firstPulseNumber}, {PulseNumber: secondPulseNumber}}, nil)
	sm.GetJetDropsMock.Set(getJetDrops)

	c, err := NewController(extractor, sm)
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Empty(t, c.missedDataRequestsQueue)

	require.Equal(t, expectedData, c.jetDropRegister)

	require.Equal(t, uint64(1), sm.GetIncompletePulsesAfterCounter())
	require.Equal(t, uint64(2), sm.GetJetDropsAfterCounter())
}

func TestNewController_ErrorGetPulses(t *testing.T) {
	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, errors.New("test error"))

	c, err := NewController(extractor, sm)
	require.Error(t, err)
	require.Contains(t, err.Error(), "test error")
	require.Nil(t, c)
	require.Equal(t, uint64(1), sm.GetIncompletePulsesAfterCounter())
}

func TestNewController_ErrorGetJetDrops(t *testing.T) {
	extractor := mock.NewJetDropsExtractorMock(t)

	pulseNumber := 1

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return([]models.Pulse{{PulseNumber: pulseNumber}}, nil)
	sm.GetJetDropsMock.Return(nil, errors.New("test error"))

	c, err := NewController(extractor, sm)
	require.Error(t, err)
	require.Contains(t, err.Error(), "test error")
	require.Nil(t, c)
	require.Equal(t, uint64(1), sm.GetIncompletePulsesAfterCounter())
	require.Equal(t, uint64(1), sm.GetJetDropsAfterCounter())
}

func TestController_SetJetDropData(t *testing.T) {
	c := Controller{
		jetDropRegister: make(map[types.Pulse][][]byte),
	}

	pulse := types.Pulse{PulseNo: 12345}
	jetID := []byte{1, 1, 1, 1, 2, 2, 2, 2}
	expectedData := map[types.Pulse][][]byte{pulse: {jetID}}

	c.SetJetDropData(pulse, jetID)

	require.Equal(t, expectedData, c.jetDropRegister)
}

func TestController_SetJetDropData_Multiple(t *testing.T) {
	c := Controller{
		jetDropRegister: make(map[types.Pulse][][]byte),
	}

	pulse := types.Pulse{PulseNo: 12345}
	firstJetID := []byte{1, 2, 3}
	secondJetID := []byte{3, 4, 5}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		c.SetJetDropData(pulse, firstJetID)
		wg.Done()
	}()
	go func() {
		c.SetJetDropData(pulse, secondJetID)
		wg.Done()
	}()
	wg.Wait()

	require.Len(t, c.jetDropRegister, 1)
	require.Len(t, c.jetDropRegister[pulse], 2)
	require.Contains(t, c.jetDropRegister[pulse], firstJetID)
	require.Contains(t, c.jetDropRegister[pulse], secondJetID)
}
