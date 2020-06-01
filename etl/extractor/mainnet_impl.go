// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"io"
	"sync"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

type MainNetExtractor struct {
	stopSignal     chan bool
	hasStarted     bool
	startStopMutes *sync.Mutex

	client           exporter.RecordExporterClient
	request          *exporter.GetRecords
	mainJetDropsChan chan *types.PlatformJetDrops
}

func NewMainNetExtractor(batchSize uint32, exporterClient exporter.RecordExporterClient) *MainNetExtractor {
	request := &exporter.GetRecords{Count: batchSize}
	return &MainNetExtractor{
		stopSignal:       make(chan bool, 1),
		startStopMutes:   &sync.Mutex{},
		client:           exporterClient,
		request:          request,
		mainJetDropsChan: make(chan *types.PlatformJetDrops),
	}
}

func (m *MainNetExtractor) GetJetDrops(ctx context.Context) <-chan *types.PlatformJetDrops {
	return m.mainJetDropsChan
}

func (m *MainNetExtractor) LoadJetDrops(ctx context.Context, fromPulseNumber int, toPulseNumber int) error {
	if fromPulseNumber < 0 {
		return errors.New("fromPulseNumber cannot be negative")
	}
	if toPulseNumber < 1 {
		return errors.New("toPulseNumber cannot be less than 1")
	}
	if fromPulseNumber > toPulseNumber {
		return errors.New("fromPulseNumber cannot be greater than toPulseNumber")
	}
	unsignedToPulseNumber := uint32(toPulseNumber)

	lastPulseNumber := uint32(fromPulseNumber)
	receivedPulseNumber := uint32(0)

	client := m.client
	request := &exporter.GetRecords{
		Count:        100,
		PulseNumber:  insolar.PulseNumber(fromPulseNumber),
		RecordNumber: 0,
	}

	go func() {
		logger := belogger.FromContext(ctx)
		for {
			if m.needStop() {
				return
			}

			log := logger.WithField("request_pulse_number", request.PulseNumber)
			log.Debug("Data request: ", request)
			stream, err := client.Export(ctx, request)

			if err != nil {
				logGRPCError(ctx, err)
				continue
			}

			// need to collect all records from pulse
			jetDrops := new(types.PlatformJetDrops)
			// Get records from the stream
			for {
				if m.needStop() {
					closeStream(ctx, stream)
					return
				}

				resp, err := stream.Recv()
				if yes := isEOF(ctx, err, stream); yes {
					//todo: batchSize
					m.mainJetDropsChan <- jetDrops
					break
				}
				logIfErrorReceived(ctx, err, request)

				if resp.ShouldIterateFrom != nil {
					request.PulseNumber = *resp.ShouldIterateFrom
					receivedPulseNumber = request.PulseNumber.AsUint32()
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

				if receivedPulseNumber > unsignedToPulseNumber {
					// now we have received all needed data
					return
				}
			}
		}
	}()

	return nil
}

func (m *MainNetExtractor) getJetDropsContinuously(ctx context.Context) {
	// from pulse, 0 means start to get from pulse number 0
	//todo: add pulse fetcher
	m.request.PulseNumber = 0
	m.request.RecordNumber = 0
	client := m.client

	lastPulseNumber := uint32(0)
	receivedPulseNumber := uint32(0)

	go func() {
		logger := belogger.FromContext(ctx)
		for {
			if m.needStop() {
				return
			}
			log := logger.WithField("request_pulse_number", m.request.PulseNumber)
			log.Debug("Data request: ", m.request)
			stream, err := client.Export(ctx, m.request)

			if err != nil {
				logGRPCError(ctx, err)
				continue
			}
			// need to collect all records from pulse
			jetDrops := new(types.PlatformJetDrops)

			// Get records from the stream
			for {
				if m.needStop() {
					return
				}
				resp, err := stream.Recv()

				if yes := isEOF(ctx, err, stream); yes {
					//todo: batchSize
					m.mainJetDropsChan <- jetDrops
					break
				}
				logIfErrorReceived(ctx, err, m.request)

				if resp.ShouldIterateFrom != nil {
					m.request.PulseNumber = *resp.ShouldIterateFrom
					receivedPulseNumber = m.request.PulseNumber.AsUint32()
					break
				}

				// save the last pulse for future requests
				m.request.RecordNumber = resp.RecordNumber
				m.request.PulseNumber = resp.Record.ID.Pulse()
				receivedPulseNumber = m.request.PulseNumber.AsUint32()

				// collect all records by PulseNumber
				if receivedPulseNumber == lastPulseNumber {
					jetDrops.Records = append(jetDrops.Records, resp)
					continue
				}

				lastPulseNumber = receivedPulseNumber

				m.mainJetDropsChan <- jetDrops
				// zeroing variable which collecting jetDrops
				jetDrops = new(types.PlatformJetDrops)
				// don't forget to save the last data
				jetDrops.Records = append(jetDrops.Records, resp)
			}
		}
	}()
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

func isEOF(ctx context.Context, err error, stream exporter.RecordExporter_ExportClient) bool {
	if err == io.EOF {
		belogger.FromContext(ctx).Debug("EOF received, quit")
		closeStream(ctx, stream)
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

func (m *MainNetExtractor) Stop(ctx context.Context) error {
	m.startStopMutes.Lock()
	defer m.startStopMutes.Unlock()
	if m.hasStarted {
		belogger.FromContext(ctx).Info("Stopping MainNet extractor...")
		m.stopSignal <- true
		m.hasStarted = false
	}
	return nil
}

func (m *MainNetExtractor) needStop() bool {
	select {
	case <-m.stopSignal:
		return true
	default:
		// continue
	}
	return false
}

func (m *MainNetExtractor) Start(ctx context.Context) error {
	m.startStopMutes.Lock()
	defer m.startStopMutes.Unlock()
	if !m.hasStarted {
		belogger.FromContext(ctx).Info("Starting MainNet extractor...")
		m.getJetDropsContinuously(ctx)
		m.hasStarted = true
	}
	return nil
}
