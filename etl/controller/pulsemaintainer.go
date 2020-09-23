// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"strings"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/insolar/pulse"
	"github.com/jinzhu/gorm"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

// pulseMaintainer checks if we have not finished pulse data in db and reloads data
func (c *Controller) pulseMaintainer(ctx context.Context) {
	log := belogger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * time.Duration(c.cfg.PulsePeriod)):
			eraseJetDropRegister(ctx, c, log)
		}
	}
}

func eraseJetDropRegister(ctx context.Context, c *Controller, log log.Logger) {
	log.Debugf("pulseMaintainer(): eraseJetDropRegister start")
	jetDropRegisterCopy := map[types.Pulse]map[string]struct{}{}
	func() {
		c.jetDropRegisterLock.Lock()
		defer c.jetDropRegisterLock.Unlock()
		for k, v := range c.jetDropRegister {
			jetDropsCopy := map[string]struct{}{}
			for jetID := range v {
				jetDropsCopy[jetID] = struct{}{}
			}
			jetDropRegisterCopy[k] = jetDropsCopy
		}
	}()

	for p, d := range jetDropRegisterCopy {
		if pulseIsComplete(p, d) {
			PulseCompleteCounter.Inc()
			log.Infof("Pulse %d completed, update it in db", p.PulseNo)
			if func() bool {

				if err := c.storage.CompletePulse(p.PulseNo); err != nil {
					log.Errorf("During pulse saving: %s", err.Error())
					return false
				}

				func() {
					c.jetDropRegisterLock.Lock()
					defer c.jetDropRegisterLock.Unlock()
					delete(c.jetDropRegister, p)
					IncompletePulsesQueue.Dec()
				}()
				return true

			}() {
				log.Infof("Pulse %d completed and saved", p.PulseNo)
			}
		} else {
			PulseNotCompleteCounter.Inc()
			// commented for worker priority proof
			// log.Debugf("Pulse %d not completed, reloading", p.PulseNo)
			// c.reloadData(ctx, p.PrevPulseNumber, p.PulseNo, false)
		}
	}
}

// pulseSequence check if we have spaces between pulses and rerequests this pulses
func (c *Controller) pulseSequence(ctx context.Context) {
	emptyPulse := models.Pulse{}
	waitTime := time.Duration(c.cfg.SequentialPeriod)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * waitTime):
		}
		var err error
		var nextSequential models.Pulse
		func() {
			c.sequentialPulseLock.Lock()
			defer c.sequentialPulseLock.Unlock()
			waitTime = time.Duration(c.cfg.SequentialPeriod)
			log := belogger.FromContext(ctx)
			log = log.WithField("sequential_pulse", c.sequentialPulse)
			CurrentSeqPulse.Set(float64(c.sequentialPulse.PulseNumber))

			nextSequential, err = c.storage.GetPulseByPrev(c.sequentialPulse)
			if err != nil && !gorm.IsRecordNotFoundError(err) {
				log.Errorf("During loading next sequential pulse: %s", err.Error())
				return
			}

			if nextSequential.IsComplete {
				err = c.storage.SequencePulse(nextSequential.PulseNumber)
				if err != nil {
					log.Errorf("During sequence next sequential pulse: %s", err.Error())
					return
				}
				c.sequentialPulse = nextSequential
				log.Infof("Pulse %d sequenced", nextSequential.PulseNumber)
				waitTime = time.Duration(0)
				return
			}

			if !nextSequential.IsComplete || nextSequential == emptyPulse {
				toPulse, err := c.storage.GetNextSavedPulse(c.sequentialPulse)
				if err != nil && !gorm.IsRecordNotFoundError(err) {
					log.Errorf("During loading next existing pulse: %s", err.Error())
					return
				}
				if toPulse == emptyPulse {
					log.Info("no next saved pulse. skipping")
					return
				}
				log.Debugf("Reloading not seq pulses %d - %d", c.sequentialPulse.PulseNumber, toPulse.PrevPulseNumber)
				c.reloadData(ctx, c.sequentialPulse.PulseNumber, toPulse.PrevPulseNumber, true)
				return
			}
		}()
	}
}

func pulseIsComplete(p types.Pulse, d map[string]struct{}) bool { // nolint
	if len(d) == 0 {
		return false
	}

	// root
	if len(d) == 1 {
		if _, ok := d[""]; ok {
			return true
		}
	}

	// inverts last bool symbol in string
	invertLastSymbol := func(s string) string {
		last := s[len(s)-1:]
		var invertedSymbol string
		if last == "1" {
			invertedSymbol = "0"
		} else {
			invertedSymbol = "1"
		}

		return s[:len(s)-1] + invertedSymbol
	}

Main:
	for jetID := range d {
		jetIDInvertedLast := invertLastSymbol(jetID)
		if _, ok := d[jetIDInvertedLast]; ok {
			// found the opposite jet drop
			continue
		} else {
			// not found, let's find any siblings
			for jetID2 := range d {
				if strings.Index(jetID2, jetIDInvertedLast) == 0 {
					// found sibling
					continue Main
				}
			}
		}
		// not found anything
		return false
	}

	// let's search all possible opposite parents or their siblings
	checkedJetIDs := make(map[string]struct{})
	for jetID := range d {
	ParentIterator:
		for i := len(jetID) - 1; i >= 1; i-- {
			jetIDParentInverted := invertLastSymbol(jetID[:i])
			if _, ok := checkedJetIDs[jetIDParentInverted]; ok {
				// found in already checked
				continue
			} else {
				for jetID2 := range d {
					if strings.Index(jetID2, jetIDParentInverted) == 0 {
						// found sibling or opposite jetDropId
						checkedJetIDs[jetIDParentInverted] = struct{}{}
						continue ParentIterator
					}
				}
			}
			// not found anything
			return false
		}
	}
	return true
}

func (c *Controller) reloadData(ctx context.Context, fromPulseNumber int64, toPulseNumber int64, priority bool) {
	log := belogger.FromContext(ctx)
	if fromPulseNumber == 0 {
		fromPulseNumber = pulse.MinTimePulse - 1
	}
	if c.missedDataManager.Add(ctx, fromPulseNumber, toPulseNumber) {
		log.Infof("Reload data from %d to %d, prior=%v", fromPulseNumber, toPulseNumber, priority)
		err := c.extractor.LoadJetDrops(ctx, fromPulseNumber, toPulseNumber, priority)
		if err != nil {
			log.Errorf("During loading missing data from extractor: %s", err.Error())
			return
		}
	}
}
