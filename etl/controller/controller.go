// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"

	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
)

// Controller checks pulses completeness
// and sends signal to reload missing data from platform
type Controller struct {
	cfg       configuration.Controller
	extractor interfaces.JetDropsExtractor
	storage   interfaces.Storage

	cancelFunc context.CancelFunc

	// jetDropRegister stores processed jetDrops for not complete pulses
	jetDropRegister     map[types.Pulse][]string
	jetDropRegisterLock sync.RWMutex
	// missedDataManager stores pulses that were reloaded
	missedDataManager *MissedDataManager

	// finalPulse is greatest complete pulse after which all pulses complete too
	finalPulse     models.Pulse
	finalPulseLock sync.RWMutex
}

// NewController returns implementation of interfaces.Controller
func NewController(cfg configuration.Controller, extractor interfaces.JetDropsExtractor, storage interfaces.Storage) (*Controller, error) {
	c := &Controller{
		cfg:               cfg,
		extractor:         extractor,
		storage:           storage,
		jetDropRegister:   make(map[types.Pulse][]string),
		missedDataManager: NewMissedDataManager(time.Second*time.Duration(cfg.ReloadPeriod), time.Second*time.Duration(cfg.ReloadCleanPeriod)),
	}
	pulses, err := c.storage.GetIncompletePulses()
	if err != nil {
		return nil, errors.Wrap(err, "can't get not complete pulses from storage")
	}
	for _, p := range pulses {
		key := types.Pulse{PulseNo: p.PulseNumber}
		jetDrops, err := c.storage.GetJetDrops(p)
		if err != nil {
			return nil, errors.Wrapf(err, "can't get jetDrops for pulse %d from storage", p.PulseNumber)
		}
		for _, jd := range jetDrops {
			c.SetJetDropData(key, jd.JetID)
		}
	}
	c.finalPulseLock.Lock()
	defer c.finalPulseLock.Unlock()
	c.finalPulse, err = c.storage.GetFinalPulse()
	if err != nil {
		return nil, errors.Wrap(err, "can't get final pulse from storage")
	}
	emptyPulse := models.Pulse{}
	if c.finalPulse == emptyPulse {
		c.finalPulse = models.Pulse{
			PulseNumber: 0,
		}
	}
	return c, nil
}

// Start implements interfaces.Starter
func (c *Controller) Start(ctx context.Context) error {
	ctx, c.cancelFunc = context.WithCancel(ctx)
	go c.pulseMaintainer(ctx)
	go c.pulseFinalizer(ctx)
	return nil
}

// Stop implements interfaces.Stopper
func (c *Controller) Stop(ctx context.Context) error {
	c.cancelFunc()
	c.missedDataManager.Stop()
	return nil
}

// SetJetDropData stores jetID, processed at specific pulse
func (c *Controller) SetJetDropData(pulse types.Pulse, jetID string) {
	c.jetDropRegisterLock.Lock()
	defer c.jetDropRegisterLock.Unlock()
	c.jetDropRegister[pulse] = append(c.jetDropRegister[pulse], jetID)
}
