// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"context"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type MainNetTransformer struct {
	stopSignal      chan bool
	extractorChan   <-chan *types.PlatformPulseData
	transformerChan chan *types.JetDrop
}

func NewMainNetTransformer(ch <-chan *types.PlatformPulseData, queueLen uint32) *MainNetTransformer {
	return &MainNetTransformer{
		stopSignal:      make(chan bool, 1),
		extractorChan:   ch,
		transformerChan: make(chan *types.JetDrop, queueLen),
	}
}

func (m *MainNetTransformer) Start(ctx context.Context) error {
	belogger.FromContext(ctx).Info("MainNetTransformer is starting")
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
		TransformedPulses.Inc()
		if err != nil {
			belogger.FromContext(ctx).Errorf("cannot transform jet drop %v, error: %s", jd, err.Error())
			Errors.Inc()
			return
		}
		if len(transform) == 0 {
			belogger.FromContext(ctx).Warn("no transformed data to logging")
		} else {
			belogger.FromContext(ctx).
				Infof("transformed jet drop to canonical for pulse: %d", transform[0].MainSection.Start.PulseData.PulseNo)
			for _, jetDrop := range transform {
				m.transformerChan <- jetDrop
				FromTransformerDataQueue.Set(float64(len(m.transformerChan)))
			}
		}
	case <-m.stopSignal:
		m.stopSignal <- true
		return
	}
}
