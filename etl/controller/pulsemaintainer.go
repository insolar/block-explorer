// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"time"

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
			}
		}
	}
}

func pulseIsComplete(p types.Pulse, d []string) bool {
	// TODO implement me
	return true
}
