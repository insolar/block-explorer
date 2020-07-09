// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/pulse"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
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
	}
}

func (e *PlatformExtractor) GetJetDrops(ctx context.Context) <-chan *types.PlatformJetDrops {
	return e.mainJetDropsChan
}

func (e *PlatformExtractor) LoadJetDrops(ctx context.Context, fromPulseNumber int64, toPulseNumber int64) error {
	e.retrievePulses(ctx, fromPulseNumber, toPulseNumber)
	return nil
}

func (e *PlatformExtractor) getJetDrops(ctx context.Context, request *exporter.GetRecords, fromPulseNumber int64, toPulseNumber int64, shouldReload bool) {
	unsignedToPulseNumber := uint32(toPulseNumber)

	client := e.client
	lastPulseNumber := uint32(fromPulseNumber)
	receivedPulseNumber := uint32(0)

	go func() {
		logger := belogger.FromContext(ctx)
		// need to collect all records from pulse
		jetDrops := new(types.PlatformJetDrops)
		for {
			if e.needStop() {
				return
			}

			var log = logger.WithField("request_pulse_number", request.PulseNumber)
			log.Debugf("export data for pulseNumber:%d, recordNumber:%d", request.PulseNumber, request.RecordNumber)
			stream, err := client.Export(ctx, request)

			if err != nil {
				logGRPCError(ctx, err)
				continue
			}

			// Get records from the stream
			for {
				if e.needStop() {
					closeStream(ctx, stream)
					return
				}

				resp, err := stream.Recv()
				if yes := isEOF(ctx, err); yes {
					// that means we have received all records in the batchSize
					break
				}
				logIfErrorReceived(ctx, err, request)

				if resp.ShouldIterateFrom != nil {
					if receivedPulseNumber == resp.ShouldIterateFrom.AsUint32() {
						log.Warnf("no data in the pulse. waiting for pulse will be changed. sleep %v", pulseDelta)
						time.Sleep(time.Second * pulseDelta)
					}
					// if we have received all data for topSyncPulse and the pulse didn't change yet, we need to continue
					// when it happened it means that we need to request again without resetting the pulse number
					if lastPulseNumber == receivedPulseNumber {
						// wait a bit to prevent multiple callings
						time.Sleep(time.Second)
						continue
					}
					request.PulseNumber = *resp.ShouldIterateFrom
					request.RecordNumber = 0
					receivedPulseNumber = request.PulseNumber.AsUint32()
					log.Debugf("jump to pulse number: %v", receivedPulseNumber)
					break
				}

				// save the last pulse for future requests
				request.RecordNumber = resp.RecordNumber
				request.PulseNumber = resp.Record.ID.Pulse()

				receivedPulseNumber = request.PulseNumber.AsUint32()

				// collect all records by PulseNumber
				if receivedPulseNumber == lastPulseNumber {
					jetDrops.Records = append(jetDrops.Records, resp)
					continue
				}

				lastPulseNumber = receivedPulseNumber

				// sending data to channel
				e.mainJetDropsChan <- jetDrops
				// zeroing variable which collecting jetDrops
				jetDrops = new(types.PlatformJetDrops)
				// don't forget to save the last data
				jetDrops.Records = append(jetDrops.Records, resp)

				if shouldReload && receivedPulseNumber > unsignedToPulseNumber {
					// now we have received all needed data
					return
				}
			}
		}
	}()
}

func (e *PlatformExtractor) getJetDropsContinuously(ctx context.Context) {
	logger := belogger.FromContext(ctx)
	var pulse uint32 = 0
	var err error

	// try to get current pulse with attempts
	for i := 0; i < e.pulseExtractAttempts; i++ {
		pulse, err = e.pulseExtractor.GetCurrentPulse(ctx)
		if err != nil {
			logger.Warnf("trying to get current pulse, attempt: %d", i)
			time.Sleep(time.Duration(pulseDelta) * time.Second)
		} else {
			break
		}
	}

	// fatal and exit if could not get current pulse
	if pulse == 0 || err != nil {
		logger.Fatalf("could not get current pulse number after %d attempts", e.pulseExtractAttempts)
	}

	logger.Infof("current pulse number: %d", pulse)
	e.request.PulseNumber = insolar.PulseNumber(pulse)
	e.request.RecordNumber = 0
	e.getJetDrops(ctx, e.request, 0, math.MaxUint32, false)
}

func logGRPCError(ctx context.Context, err error) {
	log := belogger.FromContext(ctx)
	if err != nil {
		log.Debug("Data request failed: ", err)
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method").Error())
	}
}

func logIfErrorReceived(ctx context.Context, err error, request interface{}) {
	if err != nil {
		belogger.FromContext(ctx).Debug("received error value from records gRPC stream %v", request)
	}
}

func isEOF(ctx context.Context, err error) bool {
	if err == io.EOF {
		belogger.FromContext(ctx).Debug("EOF received, quit")
		return true
	}
	return false
}

func closeStream(ctx context.Context, stream exporter.RecordExporter_ExportClient) {
	if stream != nil {
		streamError := stream.CloseSend()
		if streamError != nil {
			belogger.FromContext(ctx).Warn("Error closing stream: ", streamError)
		}
	}
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
		// e.getJetDropsContinuously(ctx)
		e.hasStarted = true
		ctx, e.cancel = context.WithCancel(ctx)
		go e.retrievePulses(ctx, 0, 0)
	}
	return nil
}

// retrievePulses - initiates full pulse retrieving between not including from and until
// zero from is latest pulse, zero until - never stop
func (e *PlatformExtractor) retrievePulses(ctx context.Context, from, until int64) {
	pu := &exporter.FullPulse{PulseNumber: insolar.PulseNumber(from)}
	var err error
	logger := belogger.FromContext(ctx)

	halfPulse := 5 * time.Second // guess a real half of pulse, but we do not known it from the platform
	for {
		if until > 0 && pu.PulseNumber >= insolar.PulseNumber(until) {
			return
		}
		select {
		case <-ctx.Done(): // we need context with cancel
			return
		default:
		}
		time.Sleep(halfPulse)
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
		log.Info("retrievePulses(): successfully retrieved")
		go e.retrieveRecords(ctx, pu)
	}
}

// retrieveRecords - retrieves all records for specified pulse
func (e *PlatformExtractor) retrieveRecords(ctx context.Context, pu *exporter.FullPulse) {
	logger := belogger.FromContext(ctx)
	for { // each pulse
		jetDrops := &types.PlatformJetDrops{Pulse: pu} // save pulse info
		log := logger.WithField("request_pulse_number", pu.PulseNumber)
		stream, err := e.client.Export(ctx, &exporter.GetRecords{PulseNumber: pu.PulseNumber, Count: 1 << 31})
		if err != nil {
			log.Error("retrieveRecords(): ", err.Error())
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
			if isEOF(ctx, err) { // stream ended, assume we have whole pulse (we have no other information of it content)
				log.Info("retrievePulses(): stream finished")
				break
			}
			if resp.ShouldIterateFrom != nil || resp.Record.ID.Pulse() != pu.PulseNumber { // next pulse packet
				closeStream(ctx, stream)
				break // FIXME
			}
			if err != nil {
				if strings.Contains(err.Error(), exporter.ErrNotFinalPulseData.Error()) {
					// let's check again, we jumped into next pulse
					if err == exporter.ErrNotFinalPulseData {
						log.Warn("ErrNotFinalPulseData: jump into next pulse", err)
					}
					log.Warn("not EOF ErrNotFinalPulseData: ", err)
					break
				}
				log.Error("not EOF: ", err)
				return // something bad happens, let's think we have broken pulse
			}

			jetDrops.Records = append(jetDrops.Records, resp)
		}
		if len(jetDrops.Records) > 0 { // we actually grabbed pulse, assume it full.
			e.mainJetDropsChan <- jetDrops
		}
	}

}
