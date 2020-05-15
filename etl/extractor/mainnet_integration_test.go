// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"fmt"
	"testing"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var localBatchSize = 2

func TestExporterIsWorking(t *testing.T) {
	ctx := context.Background()
	address, grpcServer := createGRPCServer(t)
	exporter.RegisterRecordExporterServer(grpcServer, &gserver{})

	// prepare config with listening address
	cfg := configuration.Replicator{
		Addr:            address,
		MaxTransportMsg: 100500,
	}

	// initialization MainNet connection
	client, err := connection.NewMainNetClient(ctx, cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()

	g := &gclient{}
	extractor := NewMainNetExtractor(uint32(localBatchSize), g)
	jetDrops, errors := extractor.GetJetDrops(ctx)

	for i := 0; i < localBatchSize; i++ {
		select {
		case err := <-errors:
			require.NoError(t, err)
		case jd := <-jetDrops:
			require.NotEmpty(t, jd.Records)
			t.Log(fmt.Sprintf("RecordNumber=%d, Pn=%d\n\n", jd.Records[0].RecordNumber, jd.Records[0].GetRecord().ID))
			//todo: replace to logger
			// logger.Debug("RecordNumber=%d, Pn=%d\n\n", jd.Records[0].RecordNumber, jd.Records[0].GetRecord().ID)
		}
	}
}

type gserver struct {
	exporter.RecordExporterServer
}

type gclient struct {
	exporter.RecordExporterClient
	grpc.ClientStream
}

func (c *gclient) Export(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (exporter.RecordExporter_ExportClient, error) {
	stream := recordStream{
		recv: generateRecords(localBatchSize),
	}
	return stream, nil
}
