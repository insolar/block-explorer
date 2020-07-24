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

func (c *Controller) pulseMaintainer(ctx context.Context) {
	log := belogger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * time.Duration(c.cfg.PulsePeriod))
		}
		eraseJetDropRegister(ctx, c, log)
	}
}

func eraseJetDropRegister(ctx context.Context, c *Controller, log log.Logger) {
	c.jetDropRegisterLock.Lock()
	defer c.jetDropRegisterLock.Unlock()

	for p, d := range c.jetDropRegister {
		if pulseIsComplete(p, d) {
			if func() bool {

				if err := c.storage.CompletePulse(p.PulseNo); err != nil {
					log.Errorf("During pulse saving: %s", err.Error())
					return false
				}

				delete(c.jetDropRegister, p)
				return true

			}() {
				log.Infof("Pulse %d completed and saved", p.PulseNo)
			}
		} else {
			c.reloadData(ctx, p.PrevPulseNumber, p.PulseNo)
		}
	}
}

func (c *Controller) pulseSequence(ctx context.Context) {
	log := belogger.FromContext(ctx)
	emptyPulse := models.Pulse{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * time.Duration(c.cfg.SequentialPeriod))
		}
		var err error
		var nextSequential models.Pulse
		func() {
			c.sequentialPulseLock.Lock()
			defer c.sequentialPulseLock.Unlock()
			log.WithField("sequential_pulse", c.sequentialPulse)

			nextSequential, err = c.storage.GetPulseByPrev(c.sequentialPulse)
			if err != nil && !gorm.IsRecordNotFoundError(err) {
				log.Errorf("During loading next sequential pulse: %s", err.Error())
				return
			}

			if nextSequential == emptyPulse {
				toPulse, err := c.storage.GetNextSavedPulse(c.sequentialPulse)
				if err != nil && !gorm.IsRecordNotFoundError(err) {
					log.Errorf("During loading next existing pulse: %s", err.Error())
					return
				}
				if toPulse == emptyPulse {
					log.Info("no next saved pulse. skipping")
					return
				}
				c.reloadData(ctx, c.sequentialPulse.PulseNumber, toPulse.PrevPulseNumber)
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

func (c *Controller) reloadData(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) {
	log := belogger.FromContext(ctx)
	if fromPulseNumber == 0 {
		fromPulseNumber = pulse.MinTimePulse - 1
	}
	if c.missedDataManager.Add(ctx, fromPulseNumber, toPulseNumber) {
		log.Infof("Reload data from %d to %d", fromPulseNumber, toPulseNumber)
		err := c.extractor.LoadJetDrops(ctx, fromPulseNumber, toPulseNumber)
		if err != nil {
			log.Errorf("During loading missing data from extractor: %s", err.Error())
			return
		}
	}
}
