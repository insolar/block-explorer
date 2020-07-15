// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/pulse"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

const pulseDelta = 10

type PlatformExtractor struct {
	stopSignal     chan bool
	hasStarted     bool
	startStopMutex *sync.Mutex

	pulseExtractor       interfaces.PulseExtractor
	pulseExtractAttempts int

	client           exporter.RecordExporterClient
	request          *exporter.GetRecords
	mainJetDropsChan chan *types.PlatformJetDrops
	cancel           context.CancelFunc

	batchSize uint32
}

func NewPlatformExtractor(batchSize uint32, pulseExtractor interfaces.PulseExtractor, exporterClient exporter.RecordExporterClient) *PlatformExtractor {
	request := &exporter.GetRecords{Count: batchSize}
	return &PlatformExtractor{
		stopSignal:       make(chan bool, 1),
		startStopMutex:   &sync.Mutex{},
		client:           exporterClient,
		request:          request,
		mainJetDropsChan: make(chan *types.PlatformJetDrops),

		pulseExtractor:       pulseExtractor,
		pulseExtractAttempts: 50,
		batchSize:            batchSize,
	}
}

func (e *PlatformExtractor) GetJetDrops(ctx context.Context) <-chan *types.PlatformJetDrops {
	return e.mainJetDropsChan
}

func (e *PlatformExtractor) LoadJetDrops(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) error {
	e.retrievePulses(ctx, fromPulseNumber, toPulseNumber)
	return nil
}

func (e *PlatformExtractor) Stop(ctx context.Context) error {
	e.cancel()
	e.startStopMutex.Lock()
	defer e.startStopMutex.Unlock()
	if e.hasStarted {
		belogger.FromContext(ctx).Info("Stopping platform extractor...")
		e.stopSignal <- true
		e.hasStarted = false
	}
	return nil
}

func (e *PlatformExtractor) needStop() bool {
	select {
	case <-e.stopSignal:
		return true
	default:
		// continue
	}
	return false
}

func (e *PlatformExtractor) Start(ctx context.Context) error {
	e.startStopMutex.Lock()
	defer e.startStopMutex.Unlock()
	if !e.hasStarted {
		belogger.FromContext(ctx).Info("Starting platform extractor...")
		e.hasStarted = true
		ctx, e.cancel = context.WithCancel(ctx)
		go e.retrievePulses(ctx, 0, 0)
	}
	return nil
}

func closeStream(ctx context.Context, stream exporter.RecordExporter_ExportClient) {
	if stream != nil {
		streamError := stream.CloseSend()
		if streamError != nil {
			belogger.FromContext(ctx).Warn("Error closing stream: ", streamError)
		}
	}
}

// retrievePulses - initiates full pulse retrieving between not including from and until
// zero from is latest pulse, zero until - never stop
func (e *PlatformExtractor) retrievePulses(ctx context.Context, from, until int64) {
	pu := &exporter.FullPulse{PulseNumber: insolar.PulseNumber(from)}
	var err error
	logger := belogger.FromContext(ctx)

	halfPulse := 5 * time.Second // guess a real half of pulse, but we do not known it from the platform
	for {
		select {
		case <-ctx.Done(): // we need context with cancel
			return
		default:
		}

		var before insolar.PulseNumber // already processed pulse

		if pu != nil { // not a first iteration
			before = pu.PulseNumber
		}
		pu, err = e.pulseExtractor.GetNextFinalizedPulse(ctx, int64(before))
		if err != nil { // network error ?
			if strings.Contains(err.Error(), pulse.ErrNotFound.Error()) { // seems this pulse already last
				time.Sleep(halfPulse)
				continue
			}
			logger.Error("retrievePulses(): before=%d", before, err)
			time.Sleep(time.Second)
			continue
		}
		if pu.PulseNumber == before { // no new pulse happens
			time.Sleep(halfPulse)
			continue
		}

		log := logger.WithField("pulse_number", pu.PulseNumber)
		log.Info("retrievePulses(): Done")

		go e.retrieveRecords(ctx, pu)

		if until <= 0 { // we are going on the edge of history
			time.Sleep(halfPulse)
		} else if pu.PulseNumber >= insolar.PulseNumber(until) { // we are at the end
			return
		}

	}
}

// retrieveRecords - retrieves all records for specified pulse and puts this to channel
func (e *PlatformExtractor) retrieveRecords(ctx context.Context, pu *exporter.FullPulse) {
	logger := belogger.FromContext(ctx)
	jetDrops := &types.PlatformJetDrops{Pulse: pu} // save pulse info

	for { // each portion
		log := logger.WithField("request_pulse_number", pu.PulseNumber)
		stream, err := e.client.Export(ctx, &exporter.GetRecords{PulseNumber: pu.PulseNumber,
			RecordNumber: uint32(len(jetDrops.Records)),
			Count:        e.batchSize})
		if err != nil {
			log.Error("retrieveRecords() on rpc call: ", err.Error())
			time.Sleep(time.Second)
			continue
		}

		for { // per record in request
			select {
			case <-ctx.Done():
				closeStream(ctx, stream)
				return
			default:
			}

			resp, err := stream.Recv()
			if err == io.EOF { // stream ended, we have our portion
				break
			}
			if resp == nil { // error, assume the data is broken
				log.Errorf("retrieveRecords(): empty response: err=%s", err)
				return
			}
			if resp.ShouldIterateFrom != nil || resp.Record.ID.Pulse() != pu.PulseNumber { // next pulse packet
				closeStream(ctx, stream)
				e.mainJetDropsChan <- jetDrops
				return // we have whole pulse
			}

			jetDrops.Records = append(jetDrops.Records, resp)
		}
	}

}
