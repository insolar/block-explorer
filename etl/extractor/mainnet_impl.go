// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"fmt"
	"io"

	"github.com/insolar/block-explorer/etl"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

type MainNetExtractor struct {
	client           exporter.RecordExporterClient
	request          *exporter.GetRecords
	mainJetDropsChan chan *etl.PlatformJetDrops
}

func NewMainNetExtractor(batchSize uint32, exporterClient exporter.RecordExporterClient) *MainNetExtractor {
	request := &exporter.GetRecords{Count: batchSize}
	return &MainNetExtractor{
		client:           exporterClient,
		request:          request,
		mainJetDropsChan: make(chan *etl.PlatformJetDrops),
	}
}

func (m *MainNetExtractor) GetJetDrops(ctx context.Context) <-chan *etl.PlatformJetDrops {
	m.request.PulseNumber = 0
	m.request.RecordNumber = 0
	client := m.client

	//todo: register event in some monitoring service
	errorChan := make(chan error)

	go func() {
		//todo: enable logger

		// logger := belogger.FromContext(ctx)
		for {
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

				m.request.RecordNumber = resp.RecordNumber
				jetDrops := new(etl.PlatformJetDrops)
				jetDrops.Records = append(jetDrops.Records, resp)
				m.mainJetDropsChan <- jetDrops
			}
		}
	}()

	return m.mainJetDropsChan
}
