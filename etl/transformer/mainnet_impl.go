// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"context"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
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
		if err != nil {
			belogger.FromContext(ctx).Errorf("cannot transform jet drop %v, error: %s", jd, err.Error())
			return
		}
		go func() {
			go log(ctx, transform)
			for _, t := range transform {
				m.transformerChan <- t
			}
		}()
	case <-m.stopSignal:
		m.stopSignal <- true
		return
	}
}

func log(ctx context.Context, transform []*types.JetDrop) {
	if len(transform) == 0 {
		belogger.FromContext(ctx).Warn("no transformed data to log")
		return
	}

	type customRecord struct {
		Type                string
		Ref                 string
		ObjectReference     string
		PrototypeReference  string
		PrevRecordReference string
		Order               uint32
	}
	type customJetDrop struct {
		Start   types.DropStart
		records []customRecord
	}

	data := customJetDrop{}
	for _, t := range transform {
		data.Start = t.MainSection.Start
		for _, r := range t.MainSection.Records {
			data.records = append(data.records, customRecord{
				Type:                string(models.RecordTypeFromTypes(r.Type)),
				Ref:                 restoreInsolarID(r.Ref),
				ObjectReference:     restoreInsolarID(r.ObjectReference),
				PrototypeReference:  restoreInsolarID(r.PrototypeReference),
				PrevRecordReference: restoreInsolarID(r.PrevRecordReference),
				Order:               r.Order,
			})
		}
	}
	pn := transform[0].MainSection.Start.PulseData.PulseNo
	logger := belogger.FromContext(ctx).WithField("pulse_number", pn)
	logger.Infof("transformed jet drop to canonical for pulse: %d", pn)
	logger.Debugf("transformed data: %+v", data)
}
