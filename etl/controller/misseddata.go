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
	dm := MissedDataManager{
		ttl:     ttl,
		stopped: make(chan struct{}),
	}

	ticker := time.NewTicker(cleanPeriod)

	go func() {
		var stop = false
		for !stop {
			select {
			case <-ticker.C:
				dm.deleteExpired()
			case <-dm.stopped:
				stop = true
				ticker.Stop()
			}
		}
		dm.stopped <- struct{}{}
	}()

	return &dm
}

func (dm *MissedDataManager) Stop() {
	dm.stopped <- struct{}{}
	<-dm.stopped
}

// Add adds missed data to pool
func (dm *MissedDataManager) Add(ctx context.Context, fromPulse, toPulse int) bool {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for _, missed := range dm.missedDataPool {
		if missed.fromPulse <= fromPulse && missed.toPulse >= toPulse {
			belogger.FromContext(ctx).Infof("Data from pulse %d to %d was already reload", fromPulse, toPulse)
			return false
		}
	}

	dm.missedDataPool = append(dm.missedDataPool, missedData{
		ts:        time.Now(),
		fromPulse: fromPulse,
		toPulse:   toPulse,
	})
	return true
}

func (dm *MissedDataManager) isExpired(ts time.Time) bool {
	return time.Now().Sub(ts) > dm.ttl
}

func (dm *MissedDataManager) deleteExpired() {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for i, missed := range dm.missedDataPool {
		if dm.isExpired(missed.ts) {
			dm.missedDataPool[i] = dm.missedDataPool[len(dm.missedDataPool)-1] // Copy last element to index i.
			dm.missedDataPool[len(dm.missedDataPool)-1] = missedData{}         // Erase last element (write zero value).
			dm.missedDataPool = dm.missedDataPool[:len(dm.missedDataPool)-1]
		}
	}
}
