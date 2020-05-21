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
)

type Processor struct {
	jdC          <-chan *types.JetDrop
	taskC        chan Task
	taskCCloseMu sync.Mutex
	storage      interfaces.StorageSetter
	workers      int
	active       int32
}

func NewProcessor(jb interfaces.Transformer, storage interfaces.StorageSetter, workers int) *Processor {
	if workers < 1 {
		workers = 1
	}
	return &Processor{
		jdC:          jb.GetJetDropsChannel(),
		workers:      workers,
		taskCCloseMu: sync.Mutex{},
		storage:      storage,
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
				p.process(t.JD)
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

func (p *Processor) process(jd *types.JetDrop) {
	ms := jd.MainSection
	pd := ms.Start.PulseData
	mjd := models.JetDrop{
		JetID:          nil,
		PulseNumber:    pd.PulseNo,
		FirstPrevHash:  nil,
		SecondPrevHash: nil,
		Hash:           nil,
		RawData:        nil,
		Timestamp:      pd.PulseTimestamp,
	}

	mrs := []models.Record{}
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
	p.storage.SaveJetDropData(mjd, mrs)
}
