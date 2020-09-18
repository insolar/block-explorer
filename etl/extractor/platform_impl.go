// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/pulse"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"google.golang.org/grpc/metadata"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

const PlatformAPIVersion = "2"

type PlatformExtractor struct {
	hasStarted     bool
	startStopMutex *sync.Mutex
	workers        int32
	maxWorkers     int32

	pulseExtractor interfaces.PulseExtractor

	client            exporter.RecordExporterClient
	request           *exporter.GetRecords
	mainPulseDataChan chan *types.PlatformPulseData
	cancel            context.CancelFunc

	batchSize                                 uint32
	continuousPulseRetrievingHalfPulseSeconds uint32

	shutdownBE func()
}

func NewPlatformExtractor(
	batchSize uint32,
	continuousPulseRetrievingHalfPulseSeconds uint32,
	maxWorkers int32,
	pulseExtractor interfaces.PulseExtractor,
	exporterClient exporter.RecordExporterClient,
	shutdownBE func(),
) *PlatformExtractor {
	request := &exporter.GetRecords{Count: batchSize}
	return &PlatformExtractor{
		startStopMutex:    &sync.Mutex{},
		client:            exporterClient,
		request:           request,
		mainPulseDataChan: make(chan *types.PlatformPulseData, 1000),

		pulseExtractor: pulseExtractor,
		batchSize:      batchSize,
		continuousPulseRetrievingHalfPulseSeconds: continuousPulseRetrievingHalfPulseSeconds,
		maxWorkers: maxWorkers,
		shutdownBE: shutdownBE,
	}
}

func (e *PlatformExtractor) GetJetDrops(ctx context.Context) <-chan *types.PlatformPulseData {
	return e.mainPulseDataChan
}

func (e *PlatformExtractor) LoadJetDrops(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) error {
	go e.retrievePulses(ctx, fromPulseNumber, toPulseNumber)
	return nil
}

func (e *PlatformExtractor) Stop(ctx context.Context) error {
	e.startStopMutex.Lock()
	defer e.startStopMutex.Unlock()
	if e.hasStarted {
		e.cancel()
		belogger.FromContext(ctx).Info("Stopping platform extractor...")
		e.hasStarted = false
	}
	return nil
}

func (e *PlatformExtractor) Start(ctx context.Context) error {
	e.startStopMutex.Lock()
	defer e.startStopMutex.Unlock()
	if !e.hasStarted {
		belogger.FromContext(ctx).Info("Starting platform extractor mainthread...")
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
	RetrievePulsesCount.Inc()
	defer RetrievePulsesCount.Dec()

	mainThread := until <= 0
	pu := &exporter.FullPulse{PulseNumber: insolar.PulseNumber(from)}
	var err error
	logger := belogger.FromContext(ctx)
	if mainThread {
		logger = logger.WithField("main", mainThread)
	} else {
		logger = logger.WithField("from", from).WithField("until", until)
	}

	ctx = appendPlatformVersionToCtx(ctx)
	halfPulse := time.Duration(e.continuousPulseRetrievingHalfPulseSeconds) * time.Second
	var nextNotEmptyPulseNumber *insolar.PulseNumber

	for {
		log := logger.WithField("pulse_number", pu.PulseNumber)
		log.Debug("retrievePulses(): Start")

		select {
		case <-ctx.Done(): // we need context with cancel
			log.Debug("retrievePulses(): terminating")
			return
		default:
		}

		// check free workers if not main thread
		if !mainThread {
			for !e.takeWorker() {
				sleepMs := rand.Intn(1500) + 500
				time.Sleep(time.Millisecond * time.Duration(sleepMs))
			}
		}
		ExtractProcessCount.Set(float64(atomic.LoadInt32(&e.workers)))

		before := *pu
		pu, err = e.pulseExtractor.GetNextFinalizedPulse(ctx, int64(before.PulseNumber))
		if err != nil { // network error ?
			pu = &before
			if isVersionError(err) {
				log.Errorf("retrievePulses(): version error occurred, debug: %s", debugVersionError(ctx))
				e.shutdownBE()
				break
			}
			if isRateLimitError(err) {
				log.Error("retrievePulses(): on rpc call: ", err.Error())
				Errors.With(ErrorTypeRateLimitExceeded).Inc()
				if !mainThread {
					e.freeWorker()
				}
				time.Sleep(halfPulse)
				continue
			}
			if strings.Contains(err.Error(), pulse.ErrNotFound.Error()) { // seems this pulse already last
				log.Debugf("retrievePulses(): sleep on not found pulse, before=%d err=%s", before.PulseNumber, err)
				Errors.With(ErrorTypeNotFound).Inc()
				if !mainThread {
					e.freeWorker()
				}
				time.Sleep(halfPulse * 3)
				continue
			}
			log.Errorf("retrievePulses(): before=%d err=%s", before.PulseNumber, err)
			if !mainThread {
				e.freeWorker()
			}
			time.Sleep(time.Second)
			continue
		}
		if pu.PulseNumber == before.PulseNumber { // no new pulse happens
			time.Sleep(halfPulse)
			if !mainThread {
				e.freeWorker()
			}
			continue
		}

		log.Debugf("retrievePulses(): Done, jets %d", len(pu.Jets))

		ReceivedPulses.Inc()
		LastPulseFetched.Set(float64(pu.PulseNumber))
		if !mainThread && e.maxWorkers <= 3 {
			// go to single worker scheme
			if nextNotEmptyPulseNumber == nil || pu.PulseNumber >= *nextNotEmptyPulseNumber {
				sif := "nil"
				if nextNotEmptyPulseNumber != nil {
					sif = nextNotEmptyPulseNumber.String()
				}
				log.Debugf("SIF is %s, get records", sif)
				nextNotEmptyPulseNumber = e.retrieveRecords(ctx, *pu, mainThread, false)
			} else {
				log.Debugf("SIF is %d, skipping records", *nextNotEmptyPulseNumber)
				e.retrieveRecords(ctx, *pu, mainThread, true)
			}
		} else {
			go e.retrieveRecords(ctx, *pu, mainThread, false)
		}

		if mainThread { // we are going on the edge of history
			time.Sleep(halfPulse * 2)
		} else if pu.PulseNumber >= insolar.PulseNumber(until) { // we are at the end
			return
		}

	}
}

// retrieveRecords - retrieves all records for specified pulse and puts this to channel
func (e *PlatformExtractor) retrieveRecords(ctx context.Context, pu exporter.FullPulse, mainThread, skipRequest bool) *insolar.PulseNumber {
	startedAt := time.Now()
	RetrieveRecordsCount.Inc()
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer func() {
		if !mainThread {
			e.freeWorker()
		}
		RetrieveRecordsCount.Dec()
		cancelFunc()
	}()

	// fast return when we don't need to get records
	if skipRequest {
		e.mainPulseDataChan <- &types.PlatformPulseData{Pulse: &pu}
		return nil
	}

	logger := belogger.FromContext(cancelCtx)
	log := logger.WithField("pulse_number", pu.PulseNumber).WithField("main", mainThread)
	log.Debug("retrieveRecords(): Start")
	pulseData := &types.PlatformPulseData{Pulse: &pu} // save pulse info

	halfPulse := time.Duration(e.continuousPulseRetrievingHalfPulseSeconds) * time.Second
	for { // each portion
		select {
		case <-cancelCtx.Done():
			return nil
		default:
		}
		stream, err := e.client.Export(cancelCtx, &exporter.GetRecords{PulseNumber: pu.PulseNumber,
			RecordNumber: uint32(len(pulseData.Records)),
			Count:        e.batchSize},
		)
		if err != nil {
			log.Error("retrieveRecords() on rpc call: ", err.Error())
			if isVersionError(err) {
				e.shutdownBE()
				return nil
			}
			if isRateLimitError(err) {
				Errors.With(ErrorTypeRateLimitExceeded).Inc()
				time.Sleep(halfPulse)
				continue
			}
			Errors.With(ErrorTypeOnRecordExport).Inc()
			time.Sleep(time.Second)
			continue
		}

		for { // per record in request
			select {
			case <-cancelCtx.Done():
				closeStream(cancelCtx, stream)
				return nil
			default:
			}

			resp, err := stream.Recv()
			if err == io.EOF { // stream ended, we have our portion
				break
			}
			if err != nil && isRateLimitError(err) {
				log.Error("retrieveRecords() on rpc call: ", err.Error())
				Errors.With(ErrorTypeRateLimitExceeded).Inc()
				closeStream(cancelCtx, stream)
				time.Sleep(halfPulse)
				// we should break inner for loop and reopen a stream because the clientStream finished and can't retry
				break
			}
			if resp == nil { // error, assume the data is broken
				if strings.Contains(err.Error(), exporter.ErrNotFinalPulseData.Error()) ||
					strings.Contains(err.Error(), pulse.ErrNotFound.Error()) {
					Errors.With(ErrorTypeNotFound).Inc()
					log.Debugf("retrieveRecords(): GBR Rerequest cur pulse=%d err=%s", pu.PulseNumber, err)
					time.Sleep(halfPulse * 2)
					closeStream(cancelCtx, stream)
					break
				}
				log.Errorf("retrieveRecords(): empty response: err=%s", err)
				closeStream(cancelCtx, stream)
				return nil
			}
			if resp.ShouldIterateFrom != nil || resp.Record.ID.Pulse() != pu.PulseNumber { // next pulse packet
				closeStream(cancelCtx, stream)
				e.mainPulseDataChan <- pulseData
				FromExtractorDataQueue.Set(float64(len(e.mainPulseDataChan)))
				log.Debug("retrieveRecords(): Done in %s", time.Since(startedAt))
				iterateFrom := resp.ShouldIterateFrom
				if iterateFrom == nil {
					itf := resp.Record.ID.Pulse()
					iterateFrom = &itf
				}
				return iterateFrom // we have whole pulse
			}

			pulseData.Records = append(pulseData.Records, resp)
			ReceivedRecords.Inc()
		}
	}

}

func (e *PlatformExtractor) takeWorker() bool {
	if atomic.AddInt32(&e.workers, 1) > e.maxWorkers {
		atomic.AddInt32(&e.workers, -1)
		return false
	}
	return true
}

func (e *PlatformExtractor) freeWorker() {
	atomic.AddInt32(&e.workers, -1)
}

func debugVersionError(ctx context.Context) string {
	mtd, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return "metadata not found"
	}
	return fmt.Sprintf("Client Type: %s, Client version: %s", mtd.Get(exporter.KeyClientType), mtd.Get(exporter.KeyClientVersionHeavy))
}

func appendPlatformVersionToCtx(ctx context.Context) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, exporter.KeyClientType, exporter.ValidateHeavyVersion.String())
	return metadata.AppendToOutgoingContext(ctx, exporter.KeyClientVersionHeavy, PlatformAPIVersion)
}

func isVersionError(err error) bool {
	return strings.Contains(err.Error(), exporter.ErrDeprecatedClientVersion.Error()) ||
		strings.Contains(err.Error(), "unknown heavy-version") ||
		strings.Contains(err.Error(), "unknown type client") ||
		strings.Contains(err.Error(), "incorrect format of the heavy-version")

}

func isRateLimitError(err error) bool {
	return strings.Contains(err.Error(), exporter.RateLimitExceededMsg)
}
