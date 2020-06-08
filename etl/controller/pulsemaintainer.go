// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"time"

	"github.com/insolar/insolar/log"

	"github.com/insolar/block-explorer/etl/types"
)

func (c *Controller) pulseMaintainer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * time.Duration(c.cfg.PulsePeriod))
		}
		for p, d := range c.jetDropRegister {
			if pulseIsComplete(p, d) {
				c.jetDropRegisterLock.Lock()

				if err := c.storage.CompletePulse(p.PulseNo); err != nil {
					log.Error(err)
					c.jetDropRegisterLock.Unlock()
					continue
				}

				delete(c.jetDropRegister, p)
				c.jetDropRegisterLock.Unlock()
				log.Infof("Pulse %d completed and saved", p.PulseNo)
			}
		}
	}
}

func pulseIsComplete(p types.Pulse, d [][]byte) bool {
	// TODO implement me
	return true
}
