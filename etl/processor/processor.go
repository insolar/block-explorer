package processor

import (
	"context"

	"github.com/insolar/block-explorer/etl"
)

type Processor struct {
	JDC     <-chan etl.JetDrop
	TaskC   chan Task
	Workers int
}

func NewProcessor(jb etl.Transformer, workers int) *Processor {
	this := Processor{}
	this.JDC = jb.GetJetDropsChannel()
	this.Workers = workers

	return &this
}

func (p Processor) Start(ctx context.Context) error {
	p.TaskC = make(chan Task)
	for i := 0; i < p.Workers; i++ {
		go func() {
			for {
				t, ok := <-p.TaskC
				if !ok {
					return
				}
				t.Work()
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
				ms := s.(etl.MainSection)
				for _, r := range ms.Records {
					p.TaskC <- Task{&s, &r}
				}
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
	Section *etl.Section
	Record  *etl.Record
}

func (t Task) Work() {
	// do something with record
}
