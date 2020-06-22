// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"context"
	"sync"

	"github.com/insolar/block-explorer/configuration"

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
	// missedDataRequestsQueue stores pulses, that were reloaded
	missedDataRequestsQueue map[types.Pulse]bool
}

// NewController returns implementation of interfaces.Controller
func NewController(cfg configuration.Controller, extractor interfaces.JetDropsExtractor, storage interfaces.Storage) (*Controller, error) {
	c := &Controller{
		cfg:                     cfg,
		extractor:               extractor,
		storage:                 storage,
		jetDropRegister:         make(map[types.Pulse][]string),
		missedDataRequestsQueue: make(map[types.Pulse]bool),
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
	return c, nil
}

// Start implements interfaces.Starter
func (c *Controller) Start(ctx context.Context) error {
	ctx, c.cancelFunc = context.WithCancel(ctx)
	go c.pulseMaintainer(ctx)
	return nil
}

// Stop implements interfaces.Stopper
func (c *Controller) Stop(ctx context.Context) error {
	c.cancelFunc()
	return nil
}

// SetJetDropData stores jetID, processed at specific pulse
func (c *Controller) SetJetDropData(pulse types.Pulse, jetID string) {
	c.jetDropRegisterLock.Lock()
	defer c.jetDropRegisterLock.Unlock()
	c.jetDropRegister[pulse] = append(c.jetDropRegister[pulse], jetID)
}
