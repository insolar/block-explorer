// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"time"

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

func (c *Controller) pulseFinalizer(ctx context.Context) {
	log := belogger.FromContext(ctx)
	emptyPulse := models.Pulse{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * time.Duration(c.cfg.FinalizePeriod))
		}
		var err error
		var nextFinal models.Pulse
		func() {
			c.finalPulseLock.Lock()
			defer c.finalPulseLock.Unlock()
			log.WithField("final_pulse", c.finalPulse)

			nextFinal, err = c.storage.GetPulseByPrev(c.finalPulse)
			if err != nil {
				if !gorm.IsRecordNotFoundError(err) {
					log.Errorf("During loading next final pulse: %s", err.Error())
					return
				}
			}

			if nextFinal == emptyPulse {
				toPulse, err := c.storage.GetNextSavedPulse(c.finalPulse)
				if err != nil {
					log.Errorf("During loading next existing pulse: %s", err.Error())
					return
				}
				c.reloadData(ctx, c.finalPulse.PulseNumber, toPulse.PulseNumber)
				return
			}
			if nextFinal.IsComplete {
				err = c.storage.FinalizePulse(nextFinal.PulseNumber)
				if err != nil {
					log.Errorf("During finalizing next final pulse: %s", err.Error())
					return
				}
				c.finalPulse = nextFinal
				log.Infof("Pulse %d finalized", nextFinal.PulseNumber)
				return
			}
		}()
	}
}

func pulseIsComplete(p types.Pulse, d []string) bool {
	// TODO implement me
	// This if is here for test reason, delete it after implementation and update test data for expected behavior
	if p.PulseNo < 0 {
		return false
	}
	return true
}

func (c *Controller) reloadData(ctx context.Context, fromPulseNumber int, toPulseNumber int) {
	log := belogger.FromContext(ctx)
	if c.missedDataManager.Add(ctx, fromPulseNumber, toPulseNumber) {
		err := c.extractor.LoadJetDrops(ctx, fromPulseNumber, toPulseNumber)
		if err != nil {
			log.Errorf("During loading missing data from extractor: %s", err.Error())
			return
		}
		log.Infof("Reload data from %d to %d", fromPulseNumber, toPulseNumber)
	}
}
