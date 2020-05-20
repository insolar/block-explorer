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
	JDC          <-chan types.JetDrop
	TaskC        chan Task
	TaskCCloseMu sync.Mutex
	Storage      interfaces.Storage
	Workers      int
	active       int32
}

func NewProcessor(jb interfaces.Transformer, storage interfaces.Storage, workers int) *Processor {
	if workers < 1 {
		workers = 1
	}
	return &Processor{
		JDC:          jb.GetJetDropsChannel(),
		Workers:      workers,
		TaskCCloseMu: sync.Mutex{},
		Storage:      storage,
	}

}

var ErrorAlreadyStarted = errors.New("Already started")

func (p *Processor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.active, 0, 1) {
		return ErrorAlreadyStarted
	}
	p.TaskC = make(chan Task)

	for i := 0; i < p.Workers; i++ {
		go func() {
			for {
				t, ok := <-p.TaskC
				if !ok {
					return
				}
				p.Process(t.JD)
			}
		}()
	}

	go func() {
		for {
			jd, ok := <-p.JDC
			if !ok {
				p.TaskCCloseMu.Lock()
				if atomic.CompareAndSwapInt32(&p.active, 1, 0) {
					close(p.TaskC)
				}
				p.TaskCCloseMu.Unlock()
				return
			}
			p.TaskCCloseMu.Lock()
			if atomic.LoadInt32(&p.active) == 1 {
				p.TaskC <- Task{&jd}
			}
			p.TaskCCloseMu.Unlock()
		}

	}()
	return nil
}

func (p *Processor) Stop(ctx context.Context) error {
	p.TaskCCloseMu.Lock()
	if atomic.CompareAndSwapInt32(&p.active, 1, 0) {
		close(p.TaskC)
	}
	p.TaskCCloseMu.Unlock()

	return nil
}

type Task struct {
	JD *types.JetDrop
}

func (p *Processor) Process(jd *types.JetDrop) {
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
	p.Storage.SaveJetDropData(mjd, mrs)
}
