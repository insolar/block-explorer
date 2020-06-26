// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"io"
	"math"
	"sync"
	"time"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/insolar/insolar"
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

func (m *PlatformExtractor) GetJetDrops(ctx context.Context) <-chan *types.PlatformJetDrops {
	return m.mainJetDropsChan
}

func (m *PlatformExtractor) LoadJetDrops(ctx context.Context, fromPulseNumber int, toPulseNumber int) error {
	if fromPulseNumber < 0 {
		return errors.New("fromPulseNumber cannot be negative")
	}
	if toPulseNumber < 1 {
		return errors.New("toPulseNumber cannot be less than 1")
	}
	if fromPulseNumber > toPulseNumber {
		return errors.New("fromPulseNumber cannot be greater than toPulseNumber")
	}

	request := &exporter.GetRecords{
		Count:        m.request.Count,
		PulseNumber:  insolar.PulseNumber(fromPulseNumber),
		RecordNumber: 0,
	}
	m.getJetDrops(ctx, request, fromPulseNumber, toPulseNumber, true)
	return nil
}

func (m *PlatformExtractor) getJetDrops(ctx context.Context, request *exporter.GetRecords, fromPulseNumber int, toPulseNumber int, shouldReload bool) {
	unsignedToPulseNumber := uint32(toPulseNumber)

	client := m.client
	lastPulseNumber := uint32(fromPulseNumber)
	receivedPulseNumber := uint32(0)

	go func() {
		logger := belogger.FromContext(ctx)
		// need to collect all records from pulse
		jetDrops := new(types.PlatformJetDrops)
		for {
			if m.needStop() {
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
				if m.needStop() {
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
				m.mainJetDropsChan <- jetDrops
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

func (m *PlatformExtractor) getJetDropsContinuously(ctx context.Context) {
	logger := belogger.FromContext(ctx)
	var pulse uint32 = 0
	var err error

	// try to get current pulse with attempts
	for i := 0; i < m.pulseExtractAttempts; i++ {
		pulse, err = m.pulseExtractor.GetCurrentPulse(ctx)
		if err != nil {
			logger.Warnf("trying to get current pulse, attempt: %d", i)
			time.Sleep(time.Duration(pulseDelta) * time.Second)
		} else {
			break
		}
	}

	// fatal and exit if could not get current pulse
	if pulse == 0 || err != nil {
		logger.Fatalf("could not get current pulse number after %d attempts", m.pulseExtractAttempts)
	}

	logger.Infof("current pulse number: %d", pulse)
	m.request.PulseNumber = insolar.PulseNumber(pulse)
	m.request.RecordNumber = 0
	m.getJetDrops(ctx, m.request, 0, math.MaxUint32, false)
}

func logGRPCError(ctx context.Context, err error) {
	log := belogger.FromContext(ctx)
	if err != nil {
		log.Debug("Data request failed: ", err)
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
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

func (m *PlatformExtractor) Stop(ctx context.Context) error {
	m.startStopMutex.Lock()
	defer m.startStopMutex.Unlock()
	if m.hasStarted {
		belogger.FromContext(ctx).Info("Stopping platform extractor...")
		m.stopSignal <- true
		m.hasStarted = false
	}
	return nil
}

func (m *PlatformExtractor) needStop() bool {
	select {
	case <-m.stopSignal:
		return true
	default:
		// continue
	}
	return false
}

func (m *PlatformExtractor) Start(ctx context.Context) error {
	m.startStopMutex.Lock()
	defer m.startStopMutex.Unlock()
	if !m.hasStarted {
		belogger.FromContext(ctx).Info("Starting platform extractor...")
		m.getJetDropsContinuously(ctx)
		m.hasStarted = true
	}
	return nil
}
