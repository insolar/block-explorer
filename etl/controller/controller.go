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
	jetDropRegister     map[types.Pulse]map[string]struct{}
	jetDropRegisterLock sync.RWMutex
	// missedDataManager stores pulses that were reloaded
	missedDataManager *MissedDataManager

	// sequentialPulse is greatest complete pulse after which all pulses complete too
	sequentialPulse     models.Pulse
	sequentialPulseLock sync.RWMutex
}

// NewController returns implementation of interfaces.Controller
func NewController(cfg configuration.Controller, extractor interfaces.JetDropsExtractor, storage interfaces.Storage) (*Controller, error) {
	c := &Controller{
		cfg:               cfg,
		extractor:         extractor,
		storage:           storage,
		jetDropRegister:   make(map[types.Pulse]map[string]struct{}),
		missedDataManager: NewMissedDataManager(time.Second*time.Duration(cfg.ReloadPeriod), time.Second*time.Duration(cfg.ReloadCleanPeriod)),
	}
	pulses, err := c.storage.GetIncompletePulses()
	if err != nil {
		return nil, errors.Wrap(err, "can't get not complete pulses from storage")
	}
	for _, p := range pulses {
		key := types.Pulse{PulseNo: p.PulseNumber, PrevPulseNumber: p.PrevPulseNumber, NextPulseNumber: p.NextPulseNumber}
		func() {
			c.jetDropRegisterLock.Lock()
			defer c.jetDropRegisterLock.Unlock()
			c.jetDropRegister[key] = map[string]struct{}{}
		}()
		jetDrops, err := c.storage.GetJetDrops(p)
		if err != nil {
			return nil, errors.Wrapf(err, "can't get jetDrops for pulse %d from storage", p.PulseNumber)
		}
		for _, jd := range jetDrops {
			c.SetJetDropData(key, jd.JetID)
		}
	}
	c.sequentialPulseLock.Lock()
	defer c.sequentialPulseLock.Unlock()
	c.sequentialPulse, err = c.storage.GetSequentialPulse()
	if err != nil {
		return nil, errors.Wrap(err, "can't get sequential pulse from storage")
	}
	emptyPulse := models.Pulse{}
	if c.sequentialPulse == emptyPulse {
		c.sequentialPulse = models.Pulse{
			PulseNumber: 0,
		}
	}
	return c, nil
}

// Start implements interfaces.Starter
func (c *Controller) Start(ctx context.Context) error {
	ctx, c.cancelFunc = context.WithCancel(ctx)
	c.missedDataManager.Start()
	go c.pulseMaintainer(ctx)
	go c.pulseSequence(ctx)
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
	if c.jetDropRegister[pulse] == nil {
		c.jetDropRegister[pulse] = map[string]struct{}{}
	}
	c.jetDropRegister[pulse][jetID] = struct{}{}
}
