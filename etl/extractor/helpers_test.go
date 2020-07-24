// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"

	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/pulse"
	"google.golang.org/grpc"
)

const defaultLocalBatchSize = 2
const defaultLocalPulseSize = 2
const StartPulseNumber = int64(pulse.MinTimePulse)

type recordStream struct {
	grpc.ClientStream
	recvFunc func() (*exporter.Record, error)
}

func (s recordStream) Recv() (*exporter.Record, error) {
	return s.recvFunc()
}

func (s recordStream) CloseSend() error {
	return nil
}

type RecordExporterServer struct {
	exporter.RecordExporterServer
}

type RecordExporterClient struct {
	exporter.RecordExporterClient
	grpc.ClientStream
}

func (c *RecordExporterClient) Export(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (exporter.RecordExporter_ExportClient, error) {
	withDifferencePulses := testutils.GenerateRecordsWithDifferencePulses(defaultLocalPulseSize, defaultLocalBatchSize, StartPulseNumber)
	stream := recordStream{
		recvFunc: withDifferencePulses,
	}
	return stream, nil
}
