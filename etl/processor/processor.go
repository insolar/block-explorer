package processor

import (
	"context"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/types"
)

type Processor struct {
	JDC     <-chan types.JetDrop
	TaskC   chan Task
	Workers int
	Storage interfaces.Storage
}

func NewProcessor(jb interfaces.Transformer, storage interfaces.Storage, workers int) *Processor {
	this := Processor{
		JDC:     jb.GetJetDropsChannel(),
		Workers: workers,
		Storage: storage,
	}

	return &this
}

func (p *Processor) Start(ctx context.Context) error {
	p.TaskC = make(chan Task)
	for i := 0; i < p.Workers; i++ {
		go func() {
			for {
				t, ok := <-p.TaskC
				if !ok {
					return
				}
				MakeSection(t.Section).Process(p, t.JD)
			}
		}()
	}

	go func() {
		for {
			jd, ok := <-p.JDC
			if !ok {
				close(p.TaskC)
				return
			}

			for _, s := range jd.Sections {
				p.TaskC <- Task{s, &jd}
			}

		}

	}()

	return nil
}

func (p Processor) Stop(ctx context.Context) error {
	close(p.TaskC)
	return nil
}

type Task struct {
	Section types.Section
	JD      *types.JetDrop
}

type Section interface {
	Process(processor *Processor, jd *types.JetDrop)
}

func MakeSection(s types.Section) Section {
	switch d := s.(type) {
	case types.MainSection:
		return &MainSection{d}
	default:
		panic("Unknown internal section type")
	}
}

type MainSection struct {
	types.MainSection
}

func (ms *MainSection) Process(p *Processor, jd *types.JetDrop) {
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
