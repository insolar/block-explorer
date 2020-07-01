// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"time"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/jinzhu/gorm"
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
		for p, d := range c.jetDropRegister {
			if pulseIsComplete(p, d) {
				if func() bool {
					c.jetDropRegisterLock.Lock()
					defer c.jetDropRegisterLock.Unlock()

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
				c.reloadData(ctx, p.PulseNo, p.PulseNo)
			}
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
				c.reloadData(ctx, c.sequentialPulse.PulseNumber, toPulse.PulseNumber)
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

func pulseIsComplete(p types.Pulse, d []string) bool { // nolint
	// TODO implement me
	// This if is here for test reason, delete it after implementation and update test data for expected behavior
	return p.PulseNo >= 0
}

func (c *Controller) reloadData(ctx context.Context, fromPulseNumber int, toPulseNumber int) {
	log := belogger.FromContext(ctx)
	if fromPulseNumber < 0 {
		fromPulseNumber = 0
	}

	if toPulseNumber < 1 {
		toPulseNumber = 1
	}
	if c.missedDataManager.Add(ctx, fromPulseNumber, toPulseNumber) {
		err := c.extractor.LoadJetDrops(ctx, fromPulseNumber, toPulseNumber)
		if err != nil {
			log.Errorf("During loading missing data from extractor: %s", err.Error())
			return
		}
		log.Infof("Reload data from %d to %d", fromPulseNumber, toPulseNumber)
	}
}
