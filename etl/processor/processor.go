// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package processor

import (
	"context"
	"errors"
	"fmt"
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
	log := belogger.FromContext(ctx)
	startOnce := func() error {
		p.taskCCloseMu.Lock()
		defer p.taskCCloseMu.Unlock()
		if !atomic.CompareAndSwapInt32(&p.active, 0, 1) {
			return ErrorAlreadyStarted
		}
		p.taskC = make(chan Task)
		return nil
	}

	err := startOnce()
	if err != nil {
		return err
	}

	for i := 0; i < p.workers; i++ {
		go func() {
			for {
				t, ok := <-p.taskC
				if !ok {
					return
				}
				err := p.process(ctx, t.JD)
				// todo remove this in penv-667
				if err != nil {
					log.Error(err)
					p.taskC <- t
				}
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

func (p *Processor) process(ctx context.Context, jd *types.JetDrop) error {
	ms := jd.MainSection
	pd := ms.Start.PulseData

	logger := belogger.FromContext(ctx)
	logger.Infof("Process start, pulse = %d, jetDrop = %v, record amount = %d", pd.PulseNo, ms.Start.JetDropPrefix, len(jd.MainSection.Records))

	mp := models.Pulse{
		PulseNumber:     pd.PulseNo,
		PrevPulseNumber: pd.PrevPulseNumber,
		NextPulseNumber: pd.NextPulseNumber,
		IsComplete:      false,
		Timestamp:       pd.PulseTimestamp,
	}
	err := p.storage.SavePulse(mp)
	if err != nil {
		return fmt.Errorf("cannot save pulse data: %s. pulse = %+v", err.Error(), mp)
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
		JetID:          ms.Start.JetDropPrefix,
		PulseNumber:    pd.PulseNo,
		FirstPrevHash:  firstPrevHash,
		SecondPrevHash: secondPrevHash,
		Hash:           jd.Hash,
		RawData:        jd.RawData,
		Timestamp:      pd.PulseTimestamp,
		RecordAmount:   len(ms.Records),
	}

	var mrs []models.IRecord
	for i, r := range ms.Records {
		switch r.TypeOf() {
		case types.REQUEST:
			_, ok := r.(types.Request)
			if !ok {
				break
			}
			mrs = append(mrs, models.Request{
				RecordReference:        models.ReferenceFromTypes(r.Reference()),
				Type:                   models.RequestTypeFromTypes(r.(types.Request).Type),
				CallType:               r.(types.Request).CallType,
				ObjectReference:        models.ReferenceFromTypes(r.(types.Request).ObjectReference),
				CallerObjectReference:  models.ReferenceFromTypes(r.(types.Request).Caller),
				APIRequestID:           r.(types.Request).APIRequestID,
				ReasonRequestReference: models.ReferenceFromTypes(r.(types.Request).CallReason),
				Method:                 r.(types.Request).CallSiteMethod,
				Arguments:              r.(types.Request).Arguments,
				Immutable:              r.(types.Request).Immutable,
				IsOriginalRequest:      r.(types.Request).IsOriginalRequest,
				JetID:                  mjd.JetID,
				PulseNumber:            mjd.PulseNumber,
				Order:                  i,
				Timestamp:              mjd.Timestamp,
			})
			mrs = append(mrs, models.Record{
				Reference:       models.ReferenceFromTypes(r.Reference()),
				Type:            models.RecordTypeFromTypes(r.TypeOf()),
				ObjectReference: models.ReferenceFromTypes(r.(types.Request).ObjectReference),
				Hash:            r.(types.State).Hash,
				RawData:         r.(types.State).RawData,
				JetID:           mjd.JetID,
				PulseNumber:     mjd.PulseNumber,
				Order:           i,
				Timestamp:       mjd.Timestamp,
			})
		case types.STATE:
			_, ok := r.(types.State)
			if !ok {
				break
			}
			mrs = append(mrs, models.State{
				RecordReference:    models.ReferenceFromTypes(r.Reference()),
				Type:               models.StateTypeFromTypes(r.(types.State).Type),
				RequestReference:   models.ReferenceFromTypes(r.(types.State).Request),
				ParentReference:    models.ReferenceFromTypes(r.(types.State).Parent),
				ObjectReference:    models.ReferenceFromTypes(r.(types.State).ObjectReference),
				PrevStateReference: models.ReferenceFromTypes(r.(types.State).PrevState),
				IsPrototype:        r.(types.State).IsPrototype,
				Payload:            r.(types.State).Payload,
				ImageReference:     models.ReferenceFromTypes(r.(types.State).Image),
				Hash:               r.(types.State).Hash,
				Order:              i,
				JetID:              mjd.JetID,
				PulseNumber:        mjd.PulseNumber,
				Timestamp:          mjd.Timestamp,
			})
			mrs = append(mrs, models.Record{
				Reference:           models.ReferenceFromTypes(r.Reference()),
				Type:                models.RecordTypeFromTypes(r.TypeOf()),
				ObjectReference:     models.ReferenceFromTypes(r.(types.State).ObjectReference),
				PrototypeReference:  models.ReferenceFromTypes(r.(types.State).Image),
				Payload:             r.(types.State).Payload,
				PrevRecordReference: models.ReferenceFromTypes(r.(types.State).PrevState),
				Hash:                r.(types.State).Hash,
				RawData:             r.(types.State).RawData,
				JetID:               mjd.JetID,
				PulseNumber:         mjd.PulseNumber,
				Order:               i,
				Timestamp:           mjd.Timestamp,
			})
		default:
			mrs = append(mrs, models.Record{
				Reference:           models.ReferenceFromTypes(r.Reference()),
				Type:                models.RecordTypeFromTypes(r.TypeOf()),
				ObjectReference:     models.ReferenceFromTypes(r.(types.Record).ObjectReference),
				PrototypeReference:  models.ReferenceFromTypes(r.(types.Record).PrototypeReference),
				Payload:             r.(types.Record).RecordPayload,
				PrevRecordReference: models.ReferenceFromTypes(r.(types.Record).PrevRecordReference),
				Hash:                r.(types.Record).Hash,
				RawData:             r.(types.Record).RawData,
				JetID:               mjd.JetID,
				PulseNumber:         mjd.PulseNumber,
				Order:               i,
				Timestamp:           mjd.Timestamp,
			})
		}
	}
	err = p.storage.SaveJetDropData(mjd, mrs, mp.PulseNumber)
	if err != nil {
		return fmt.Errorf("cannot save jetDrop data: %s. jetDrop:{jetID: %s, pulseNumber: %d}, record amount = %d",
			err.Error(), mjd.JetID, mjd.PulseNumber, len(mrs))
	}
	p.controller.SetJetDropData(pd, mjd.JetID)
	logger.Infof("Processed: pulseNumber = %d, jetID = %v", pd.PulseNo, mjd.JetID)
	return nil
}
