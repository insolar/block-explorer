// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"
	"fmt"
	"testing"

	"github.com/insolar/block-explorer/etl"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var localbatchSize = 2

func TestExporterIsWorking(t *testing.T) {
	address, grpcServer := createGRPCServer(t)
	exporter.RegisterRecordExporterServer(grpcServer, &gserver{})

	// prepare config with listening address
	cfg := etl.GRPCConfig{
		Addr:            address,
		MaxTransportMsg: 100500,
	}

	// initialization MainNet connection
	client, err := connection.NewMainNetClient(cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()

	g := &gclient{}
	extractor := NewMainNetExtractor(uint32(localbatchSize), g)
	jetDrops, errors := extractor.GetJetDrops(context.Background())

	for i := 0; i < localbatchSize; i++ {
		select {
		case err := <-errors:
			println(err)
			require.NoError(t, err)
			panic("sss")
		case jd := <-jetDrops:
			require.NotEmpty(t, jd.Records)
			println(fmt.Sprintf("RecordNumber=%d, Pn=%d\n\n", jd.Records[0].RecordNumber, jd.Records[0].GetRecord().ID))
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
	fmt.Println("client function")

	stream := recordStream{
		recv: generateRecords(localbatchSize),
	}
	return stream, nil
}
