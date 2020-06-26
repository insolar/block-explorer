// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type missedData struct {
	ts        time.Time
	fromPulse int
	toPulse   int
}

// MissedDataManager manages working with missed data pool
// It's thread safe
type MissedDataManager struct {
	mutex          sync.Mutex
	missedDataPool []missedData
	ttl            time.Duration
	stopped        chan struct{}
}

// NewMissedDataManager creates new missed data manager with custom params
func NewMissedDataManager(ttl time.Duration, cleanPeriod time.Duration) *MissedDataManager {
	sm := MissedDataManager{
		ttl:     ttl,
		stopped: make(chan struct{}),
	}

	ticker := time.NewTicker(cleanPeriod)

	go func() {
		var stop = false
		for !stop {
			select {
			case <-ticker.C:
				sm.deleteExpired()
			case <-sm.stopped:
				stop = true
				ticker.Stop()
			}
		}
		sm.stopped <- struct{}{}
	}()

	return &sm
}

func (sm *MissedDataManager) Stop() {
	sm.stopped <- struct{}{}
	<-sm.stopped
}

// Add adds missed data to pool
func (sm *MissedDataManager) Add(ctx context.Context, fromPulse, toPulse int) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for _, missed := range sm.missedDataPool {
		if missed.fromPulse <= fromPulse && missed.toPulse >= toPulse {
			belogger.FromContext(ctx).Infof("Data from pulse %d to %d was already reload", fromPulse, toPulse)
			return false
		}
	}

	sm.missedDataPool = append(sm.missedDataPool, missedData{
		ts:        time.Now(),
		fromPulse: fromPulse,
		toPulse:   toPulse,
	})
	return true
}

func (sm *MissedDataManager) isExpired(ts time.Time) bool {
	return time.Now().Sub(ts) > sm.ttl
}

func (sm *MissedDataManager) deleteExpired() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for i, missed := range sm.missedDataPool {
		if sm.isExpired(missed.ts) {
			sm.missedDataPool[i] = sm.missedDataPool[len(sm.missedDataPool)-1] // Copy last element to index i.
			sm.missedDataPool[len(sm.missedDataPool)-1] = missedData{}         // Erase last element (write zero value).
			sm.missedDataPool = sm.missedDataPool[:len(sm.missedDataPool)-1]
		}
	}
}
