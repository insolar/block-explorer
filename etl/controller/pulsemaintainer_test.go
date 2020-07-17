// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/insolar/insolar/pulse"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/stretchr/testify/require"
)

func TestController_pulseMaintainer(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0, ReloadPeriod: 10, ReloadCleanPeriod: 1, SequentialPeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)
	sm.GetSequentialPulseMock.Return(models.Pulse{}, nil)
	sm.GetPulseByPrevMock.Return(models.Pulse{}, errors.New("test error"))

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, c.Start(ctx))
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

// sequential is 0, pulses in db: [1000110], expect loading data from 0 to 1000110
// sequential is 0, pulses in db: [1000110], expect don't load already loaded data
// sequential is 0, pulses in db: [MinTimePulse, 1000110], expect nothing happens
// sequential is 0, pulses in db: [MinTimePulse, 1000110], MinTimePulse is complete, expect sequential to change
// sequential is MinTimePulse, pulses in db: [MinTimePulse, 1000110], expect don't load already loaded data
func TestController_pulseSequence_StartFromNothing(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0, ReloadPeriod: 10, ReloadCleanPeriod: 1, SequentialPeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)
	sm.GetSequentialPulseMock.Return(models.Pulse{}, nil)
	wg := sync.WaitGroup{}
	wg.Add(5)
	sm.GetPulseByPrevMock.Set(func(prevPulse models.Pulse) (p1 models.Pulse, err error) {
		if sm.GetPulseByPrevBeforeCounter() <= 5 {
			wg.Done()
		}
		if sm.GetPulseByPrevBeforeCounter() == 1 || sm.GetPulseByPrevBeforeCounter() == 2 {
			require.Equal(t, int64(0), prevPulse.PulseNumber)
			return models.Pulse{}, nil
		}
		if sm.GetPulseByPrevBeforeCounter() == 3 {
			require.Equal(t, int64(0), prevPulse.PulseNumber)
			return models.Pulse{PrevPulseNumber: 0, PulseNumber: pulse.MinTimePulse, NextPulseNumber: 1000000, IsComplete: false}, nil
		}
		if sm.GetPulseByPrevBeforeCounter() == 4 {
			require.Equal(t, int64(0), prevPulse.PulseNumber)
			return models.Pulse{PrevPulseNumber: 0, PulseNumber: pulse.MinTimePulse, NextPulseNumber: 1000000, IsComplete: true}, nil
		}
		require.Equal(t, int64(pulse.MinTimePulse), prevPulse.PulseNumber)
		return models.Pulse{}, nil
	})
	sm.GetNextSavedPulseMock.Set(func(fromPulseNumber models.Pulse) (p1 models.Pulse, err error) {
		if sm.GetNextSavedPulseBeforeCounter() == 1 || sm.GetPulseByPrevBeforeCounter() == 2 {
			require.Equal(t, int64(0), fromPulseNumber.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000100, PulseNumber: 1000110, NextPulseNumber: 1000120}, nil
		}
		if sm.GetNextSavedPulseBeforeCounter() == 3 {
			require.Equal(t, int64(pulse.MinTimePulse), fromPulseNumber.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000100, PulseNumber: 1000110, NextPulseNumber: 1000120}, nil
		}
		require.Equal(t, int64(pulse.MinTimePulse), fromPulseNumber.PulseNumber)
		return models.Pulse{}, nil
	})
	extractor.LoadJetDropsMock.Set(func(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) (err error) {
		if extractor.LoadJetDropsBeforeCounter() == 1 {
			require.Equal(t, int64(0), fromPulseNumber)
			require.Equal(t, int64(1000110), toPulseNumber)
		} else {
			require.Fail(t, "LoadJetDrops was called more than once")
		}
		return nil
	})
	sm.SequencePulseMock.Set(func(pulseNumber int64) (err error) {
		if sm.SequencePulseBeforeCounter() == 1 {
			require.Equal(t, int64(pulse.MinTimePulse), pulseNumber)
		}
		return nil
	})

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	err = c.Start(ctx)
	require.NoError(t, err)
	wg.Wait()
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

// sequential is 1000000, pulses in db: [1000000, 1000020], expect loading data from 1000000 to 1000020
// sequential is 1000000, pulses in db: [1000000, 1000010, 1000020], expect don't load already loaded data
// sequential is 1000000, pulses in db: [1000000, 1000010, 1000020], 1000010 is complete, expect sequential to change
// sequential is 1000010, pulses in db: [1000000, 1000010, 1000020], 1000020 is complete, expect sequential to change
// sequential is 1000020, pulses in db: [1000000, 1000010, 1000020], expect nothing happens
func TestController_pulseSequence_StartFromSomething(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0, ReloadPeriod: 10, ReloadCleanPeriod: 1, SequentialPeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)
	sm.GetSequentialPulseMock.Return(models.Pulse{PulseNumber: 1000000}, nil)
	wg := sync.WaitGroup{}
	wg.Add(5)
	sm.GetPulseByPrevMock.Set(func(prevPulse models.Pulse) (p1 models.Pulse, err error) {
		if sm.GetPulseByPrevBeforeCounter() <= 5 {
			wg.Done()
		}
		if sm.GetPulseByPrevBeforeCounter() == 1 || sm.GetPulseByPrevBeforeCounter() == 2 {
			require.Equal(t, int64(1000000), prevPulse.PulseNumber)
			return models.Pulse{}, nil
		}
		if sm.GetPulseByPrevBeforeCounter() == 3 {
			require.Equal(t, int64(1000000), prevPulse.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000000, PulseNumber: 1000010, NextPulseNumber: 1000020, IsComplete: true}, nil
		}
		if sm.GetPulseByPrevBeforeCounter() == 4 {
			require.Equal(t, int64(1000010), prevPulse.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000010, PulseNumber: 1000020, NextPulseNumber: 1000030, IsComplete: true}, nil
		}
		require.Equal(t, int64(1000020), prevPulse.PulseNumber)
		return models.Pulse{PrevPulseNumber: 1000020, PulseNumber: 1000030, NextPulseNumber: 1000040, IsComplete: false}, nil
	})
	sm.GetNextSavedPulseMock.Set(func(fromPulseNumber models.Pulse) (p1 models.Pulse, err error) {
		if sm.GetNextSavedPulseBeforeCounter() == 1 || sm.GetPulseByPrevBeforeCounter() == 2 {
			require.Equal(t, int64(1000000), fromPulseNumber.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000010, PulseNumber: 1000020, NextPulseNumber: 1000030}, nil
		}
		require.Equal(t, int64(1000020), fromPulseNumber.PulseNumber)
		return models.Pulse{}, nil
	})
	extractor.LoadJetDropsMock.Set(func(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) (err error) {
		if extractor.LoadJetDropsBeforeCounter() == 1 {
			require.Equal(t, int64(1000000), fromPulseNumber)
			require.Equal(t, int64(1000020), toPulseNumber)
		} else {
			require.Fail(t, "LoadJetDrops was called more than once")
		}
		return nil
	})
	sm.SequencePulseMock.Set(func(pulseNumber int64) (err error) {
		if sm.SequencePulseBeforeCounter() == 1 {
			require.Equal(t, int64(1000010), pulseNumber)
		}
		if sm.SequencePulseBeforeCounter() == 2 {
			require.Equal(t, int64(1000020), pulseNumber)
		}
		return nil
	})

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	err = c.Start(ctx)
	require.NoError(t, err)
	wg.Wait()
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

// sequential is 1000000, pulses in db: [1000000, 1000010, 1000020], 1000010 is complete, expect sequential to change
// sequential is 1000010, pulses in db: [1000000, 1000010, 1000020], 1000020 is complete, expect sequential to change
// sequential is 1000020, pulses in db: [1000000, 1000010, 1000020], expect nothing happens
func TestController_pulseSequence_Start_NoMissedData(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0, ReloadPeriod: 10, ReloadCleanPeriod: 1, SequentialPeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)
	sm.GetSequentialPulseMock.Return(models.Pulse{PulseNumber: 1000000}, nil)
	wg := sync.WaitGroup{}
	wg.Add(3)
	sm.GetPulseByPrevMock.Set(func(prevPulse models.Pulse) (p1 models.Pulse, err error) {
		if sm.GetPulseByPrevBeforeCounter() <= 3 {
			wg.Done()
		}
		if sm.GetPulseByPrevBeforeCounter() == 1 {
			require.Equal(t, int64(1000000), prevPulse.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000000, PulseNumber: 1000010, NextPulseNumber: 1000020, IsComplete: true}, nil
		}
		if sm.GetPulseByPrevBeforeCounter() == 2 {
			require.Equal(t, int64(1000010), prevPulse.PulseNumber)
			return models.Pulse{PrevPulseNumber: 1000010, PulseNumber: 1000020, NextPulseNumber: 1000030, IsComplete: true}, nil
		}
		require.Equal(t, int64(1000020), prevPulse.PulseNumber)
		return models.Pulse{PrevPulseNumber: 1000020, PulseNumber: 1000030, NextPulseNumber: 1000040, IsComplete: false}, nil
	})
	sm.SequencePulseMock.Set(func(pulseNumber int64) (err error) {
		if sm.SequencePulseBeforeCounter() == 1 {
			require.Equal(t, int64(1000010), pulseNumber)
		}
		if sm.SequencePulseBeforeCounter() == 2 {
			require.Equal(t, int64(1000020), pulseNumber)
		}
		return nil
	})

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	err = c.Start(ctx)
	require.NoError(t, err)
	wg.Wait()
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

func TestController_pulseMaintainer_Start_PulsesCompleteAndNot(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0, ReloadPeriod: 10, ReloadCleanPeriod: 1, SequentialPeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetSequentialPulseMock.Return(models.Pulse{PulseNumber: 0}, nil)
	sm.GetPulseByPrevMock.Set(func(prevPulse models.Pulse) (p1 models.Pulse, err error) {
		return models.Pulse{}, errors.New("some test error")
	})

	sm.GetIncompletePulsesMock.Return([]models.Pulse{
		{PulseNumber: 1000000},
		{PulseNumber: 1000010},
	}, nil)
	sm.GetJetDropsMock.When(models.Pulse{PulseNumber: 1000000}).Then([]models.JetDrop{{JetID: "1000"}}, nil)
	sm.GetJetDropsMock.When(models.Pulse{PulseNumber: 1000010}).Then([]models.JetDrop{{JetID: ""}}, nil)

	wg := sync.WaitGroup{}
	wg.Add(2)
	sm.CompletePulseMock.Set(func(pulseNumber int64) (err error) {
		require.Equal(t, int64(1000010), pulseNumber)
		require.EqualValues(t, 1, sm.CompletePulseBeforeCounter())
		wg.Done()
		return nil
	})
	extractor.LoadJetDropsMock.Set(func(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) (err error) {
		require.Equal(t, int64(1000000), fromPulseNumber)
		require.Equal(t, int64(1000000), toPulseNumber)
		require.EqualValues(t, 1, extractor.LoadJetDropsBeforeCounter())
		wg.Done()
		return nil
	})

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	err = c.Start(ctx)
	require.NoError(t, err)
	wg.Wait()
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

// sequential is 1000000, pulses in db: [1000000, 1000020], expect loading data from 1000000 to 1000020
// sequential is 1000000, pulses in db: [1000000, 1000020], expect don't load already loaded data
// wait ReloadPeriod seconds
// sequential is 1000000, pulses in db: [1000000, 1000020], expect loading data from 1000000 to 1000020
func TestController_pulseSequence_ReloadPeriodExpired(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0, ReloadPeriod: 2, ReloadCleanPeriod: 1, SequentialPeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)
	sm.GetSequentialPulseMock.Return(models.Pulse{PulseNumber: 1000000}, nil)
	wg := sync.WaitGroup{}
	wg.Add(2)
	sm.GetPulseByPrevMock.Set(func(prevPulse models.Pulse) (p1 models.Pulse, err error) {
		return models.Pulse{}, nil
	})
	sm.GetNextSavedPulseMock.Set(func(fromPulseNumber models.Pulse) (p1 models.Pulse, err error) {
		return models.Pulse{PrevPulseNumber: 1000010, PulseNumber: 1000020, NextPulseNumber: 1000030}, nil
	})
	extractor.LoadJetDropsMock.Set(func(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) (err error) {
		require.Equal(t, int64(1000000), fromPulseNumber)
		require.Equal(t, int64(1000020), toPulseNumber)
		if extractor.LoadJetDropsBeforeCounter() > 2 {
			require.Fail(t, "LoadJetDrops was called more than once")
		}
		wg.Done()
		return nil
	})

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	err = c.Start(ctx)
	require.NoError(t, err)
	time.Sleep(time.Duration(cfg.ReloadPeriod) * time.Second)
	wg.Wait()
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

func Test_pulseIsComplete(t *testing.T) {
	type args struct {
		p types.Pulse
		d []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "root",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"",
				},
			},
			want: true,
		},
		{
			name: "complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"1",
					"000",
					"001",
					"010",
					"011",
				},
			},
			want: true,
		},
		{
			name: "not complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"1",
					"000",
					"001",
					"010",
				},
			},
			want: false,
		},
		{
			name: "not complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"1",
					"000",
					"001",
				},
			},
			want: false,
		},
		{
			name: "not complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"000",
					"001",
					"010",
					"011",
				},
			},
			want: false,
		},
		{
			name: "not complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"000",
					"001",
					"011",
				},
			},
			want: false,
		},
		{
			name: "complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"10",
					"11",
					"000",
					"001",
					"010",
					"011",
				},
			},
			want: true,
		},
		{
			name: "complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"10",
					"110",
					"111",
					"000",
					"001",
					"010",
					"011",
				},
			},
			want: true,
		},
		{
			name: "not complete",
			args: args{
				types.Pulse{PulseNo: 1000},
				[]string{
					"110",
					"111",
					"000",
					"001",
					"010",
					"011",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pulseIsComplete(tt.args.p, tt.args.d); got != tt.want {
				t.Errorf("pulseIsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}
