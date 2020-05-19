// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"fmt"
	"io"

	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

type MainNetExtractor struct {
	stopSignal chan bool

	client           exporter.RecordExporterClient
	request          *exporter.GetRecords
	mainJetDropsChan chan *types.PlatformJetDrops
}

func NewMainNetExtractor(batchSize uint32, exporterClient exporter.RecordExporterClient) *MainNetExtractor {
	request := &exporter.GetRecords{Count: batchSize}
	return &MainNetExtractor{
		stopSignal:       make(chan bool, 1),
		client:           exporterClient,
		request:          request,
		mainJetDropsChan: make(chan *types.PlatformJetDrops),
	}
}

func (m *MainNetExtractor) GetJetDrops(ctx context.Context) <-chan *types.PlatformJetDrops {
	// from pulse, 0 means start to get from pulse number 0
	//todo: add pulse fetcher
	m.request.PulseNumber = 0
	m.request.RecordNumber = 0
	client := m.client

	//todo: register event in some monitoring service
	errorChan := make(chan error)

	go func() {
		//todo: enable logger

		// logger := belogger.FromContext(ctx)
		for {
			if m.needStop() {
				return
			}
			// log := logger.WithField("request_pulse_number", m.request.PulseNumber)
			// m.log.Debug("Data request: ", m.request)
			fmt.Println("Data request")
			stream, err := client.Export(ctx, m.request)

			if err != nil {
				// log.Debug("Data request failed: ", err)
				println("Data request failed")
				errorChan <- errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method")
				continue
			}

			// Get records from the stream
			for {
				if m.needStop() {
					return
				}
				resp, err := stream.Recv()
				if err == io.EOF {
					// log.Debug("EOF received, quit")
					println("EOF received, quit")
					break
				}
				if err != nil {
					// log.Debug("received error value from records gRPC stream %v", m.request)
					println("received error value from records gRPC stream %v", m.request)
					errorChan <- errors.Wrapf(err, "received error value from records gRPC stream %v", m.request)
				}

				// save the last pulse for future requests
				m.request.RecordNumber = resp.RecordNumber
				m.request.PulseNumber = resp.Record.ID.Pulse()

				jetDrops := new(types.PlatformJetDrops)
				jetDrops.Records = append(jetDrops.Records, resp)
				m.mainJetDropsChan <- jetDrops
			}
		}
	}()

	return m.mainJetDropsChan
}

func (m *MainNetExtractor) Stop() {
	m.stopSignal <- true
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
