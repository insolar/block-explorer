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
	mdm := MissedDataManager{
		ttl:     ttl,
		stopped: make(chan struct{}),
	}

	ticker := time.NewTicker(cleanPeriod)

	go func() {
		var stop = false
		for !stop {
			select {
			case <-ticker.C:
				mdm.deleteExpired()
			case <-mdm.stopped:
				stop = true
				ticker.Stop()
			}
		}
		mdm.stopped <- struct{}{}
	}()

	return &mdm
}

func (mdm *MissedDataManager) Stop() {
	mdm.stopped <- struct{}{}
	<-mdm.stopped
}

// Add adds missed data to pool
func (mdm *MissedDataManager) Add(ctx context.Context, fromPulse, toPulse int) bool {
	mdm.mutex.Lock()
	defer mdm.mutex.Unlock()

	for _, missed := range mdm.missedDataPool {
		if missed.fromPulse <= fromPulse && missed.toPulse >= toPulse {
			belogger.FromContext(ctx).Infof("Data from pulse %d to %d was already reload", fromPulse, toPulse)
			return false
		}
	}

	mdm.missedDataPool = append(mdm.missedDataPool, missedData{
		ts:        time.Now(),
		fromPulse: fromPulse,
		toPulse:   toPulse,
	})
	return true
}

func (mdm *MissedDataManager) isExpired(ts time.Time) bool {
	return time.Since(ts) > mdm.ttl
}

func (mdm *MissedDataManager) deleteExpired() {
	mdm.mutex.Lock()
	defer mdm.mutex.Unlock()

	for i, missed := range mdm.missedDataPool {
		length := len(mdm.missedDataPool)
		if i < length && mdm.isExpired(missed.ts) {
			mdm.missedDataPool[i] = mdm.missedDataPool[length-1] // Copy last element to index i.
			mdm.missedDataPool[length-1] = missedData{}          // Erase last element (write zero value).
			mdm.missedDataPool = mdm.missedDataPool[:length-1]
		}
	}
}
