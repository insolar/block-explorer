// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package processor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type Processor struct {
	jdC          <-chan *types.JetDrop
	taskC        chan Task
	taskCCloseMu sync.Mutex
	storage      interfaces.StorageSetter
	controller   interfaces.Controller
	workers      int
	active       int32
}

func NewProcessor(jb interfaces.Transformer, storage interfaces.StorageSetter, controller interfaces.Controller, workers int) *Processor {
	if workers < 1 {
		workers = 1
	}
	return &Processor{
		jdC:          jb.GetJetDropsChannel(),
		workers:      workers,
		taskCCloseMu: sync.Mutex{},
		storage:      storage,
		controller:   controller,
	}

}

var ErrorAlreadyStarted = errors.New("Already started")

func (p *Processor) Start(ctx context.Context) error {
	p.taskCCloseMu.Lock()
	if !atomic.CompareAndSwapInt32(&p.active, 0, 1) {
		p.taskCCloseMu.Unlock()
		return ErrorAlreadyStarted
	}
	p.taskC = make(chan Task)
	p.taskCCloseMu.Unlock()

	for i := 0; i < p.workers; i++ {
		go func() {
			for {
				t, ok := <-p.taskC
				if !ok {
					return
				}
				p.process(ctx, t.JD)
			}
		}()
	}

	go func() {
		for {
			jd, ok := <-p.jdC
			if !ok {
				p.taskCCloseMu.Lock()
				if atomic.CompareAndSwapInt32(&p.active, 1, 0) {
					close(p.taskC)
				}
				p.taskCCloseMu.Unlock()
				return
			}
			p.taskCCloseMu.Lock()
			if atomic.LoadInt32(&p.active) == 1 {
				p.taskC <- Task{jd}
			}
			p.taskCCloseMu.Unlock()
		}

	}()
	return nil
}

func (p *Processor) Stop(ctx context.Context) error {
	p.taskCCloseMu.Lock()
	if atomic.CompareAndSwapInt32(&p.active, 1, 0) {
		close(p.taskC)
	}
	p.taskCCloseMu.Unlock()

	return nil
}

type Task struct {
	JD *types.JetDrop
}

func (p *Processor) process(ctx context.Context, jd *types.JetDrop) {
	ms := jd.MainSection
	pd := ms.Start.PulseData

	logger := belogger.FromContext(ctx)
	logger.Infof("pulse = %d, jetDrop = %v, record amount = %d", pd.PulseNo, ms.Start.JetDropPrefix, len(jd.MainSection.Records))

	mp := models.Pulse{
		PulseNumber:     pd.PulseNo, // TODO PulseNumber must be int64
		PrevPulseNumber: pd.PrevPulseNumber,
		NextPulseNumber: pd.NextPulseNumber,
		IsComplete:      false,
		Timestamp:       pd.PulseTimestamp,
	}
	err := p.storage.SavePulse(mp)
	if err != nil {
		logger.Errorf("cannot save pulse data: %s. pulse = %+v", err.Error(), mp)
		return
	}

	var firstPrevHash []byte
	var secondPrevHash []byte
	if len(ms.DropContinue.PrevDropHash) > 0 {
		firstPrevHash = ms.DropContinue.PrevDropHash[0]
	}
	if len(ms.DropContinue.PrevDropHash) > 1 {
		secondPrevHash = ms.DropContinue.PrevDropHash[1]
	}

	mjd := models.JetDrop{
		JetID:          ms.Start.JetDropPrefix, // FIXME
		PulseNumber:    pd.PulseNo, //FIXME
		FirstPrevHash:  firstPrevHash,
		SecondPrevHash: secondPrevHash,
		Hash:           jd.Hash,
		RawData:        jd.RawData,
		Timestamp:      pd.PulseTimestamp,
		RecordAmount:   len(ms.Records),
	}

	var mrs []models.Record
	for i, r := range ms.Records {
		mrs = append(mrs, models.Record{
			Reference:           models.ReferenceFromTypes(r.Ref),
			Type:                models.RecordTypeFromTypes(r.Type),
			ObjectReference:     models.ReferenceFromTypes(r.ObjectReference),
			PrototypeReference:  models.ReferenceFromTypes(r.PrototypeReference),
			Payload:             r.RecordPayload,
			PrevRecordReference: models.ReferenceFromTypes(r.PrevRecordReference),
			Hash:                r.Hash,
			RawData:             r.RawData,
			JetID:               mjd.JetID,
			PulseNumber:         mjd.PulseNumber,
			Order:               i,
			Timestamp:           mjd.Timestamp,
		})
	}
	err = p.storage.SaveJetDropData(mjd, mrs)
	if err != nil {
		logger.Errorf("cannot save jetDrop data: %s. jetDrop:{jetID: %s, pulseNumber: %d}, record amount = %d\n",
			err.Error(), mjd.JetID, mjd.PulseNumber, len(mrs))
		return
	}
	p.controller.SetJetDropData(pd, mjd.JetID)
	logger.Infof("Processed: pulseNumber = %d, jetID = %v\n", pd.PulseNo, mjd.JetID)
}
