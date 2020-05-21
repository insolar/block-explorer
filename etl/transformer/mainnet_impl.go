// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"context"
	"fmt"

	"github.com/insolar/block-explorer/etl/types"
)

type MainNetTransformer struct {
	stopSignal      chan bool
	extractorChan   <-chan *types.PlatformJetDrops
	transformerChan chan *types.JetDrop
}

func NewMainNetTransformer(ch <-chan *types.PlatformJetDrops) *MainNetTransformer {
	return &MainNetTransformer{
		stopSignal:      make(chan bool, 1),
		extractorChan:   ch,
		transformerChan: make(chan *types.JetDrop),
	}
}

func (m *MainNetTransformer) Start(ctx context.Context) error {
	// belogger.FromContext(ctx).Info("MainNetTransformer is starting")
	go func() {
		for {
			m.run(ctx)
			if m.needStop() {
				return
			}
		}
	}()

	return nil
}

func (m *MainNetTransformer) Stop(ctx context.Context) error {
	ctx.Done()
	m.stopSignal <- true
	return nil
}

func (m *MainNetTransformer) GetJetDropsChannel() <-chan *types.JetDrop {
	return m.transformerChan
}

func (m *MainNetTransformer) needStop() bool {
	select {
	case <-m.stopSignal:
		return true
	default:
		// continue
	}
	return false
}

func (m *MainNetTransformer) run(ctx context.Context) {
	select {
	case jd := <-m.extractorChan:
		transform, err := Transform(ctx, jd)
		if err != nil {
			//todo: add logging here
			fmt.Print(err.Error())
			return
		}
		go func() {
			//todo: add logging here
			for _, t := range transform {
				m.transformerChan <- t
			}
		}()
	case <-m.stopSignal:
		m.stopSignal <- true
		return
	}
}
