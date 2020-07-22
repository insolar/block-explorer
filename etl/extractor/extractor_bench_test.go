// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build bench

package extractor

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/block-explorer/testutils/clients"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func BenchmarkPlatformExtractorGetJetDrops(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		ctx := context.Background()
		server := testutils.CreateTestGRPCServer(b)
		exporter.RegisterRecordExporterServer(server.Server, &RecordExporterServer{})
		server.Serve(b)

		// prepare config with listening address
		cfg := configuration.Replicator{
			Addr:            server.Network,
			MaxTransportMsg: 100500,
		}

		// initialization Platform connection
		client, err := connection.NewGRPCClientConnection(ctx, cfg)
		require.NoError(b, err)

		pulseClient := clients.GetTestPulseClient(1, nil)
		extractor := NewPlatformExtractor(uint32(defaultLocalBatchSize), 0, 100, NewPlatformPulseExtractor(pulseClient), &RecordExporterClient{})
		fullPulse, err := clients.GetFullPulse(uint32(StartPulseNumber))
		require.NoError(b, err)
		go extractor.retrieveRecords(ctx, fullPulse)

		b.StartTimer()
		jetDrops := extractor.GetJetDrops(ctx)
		select {
		case jd := <-jetDrops:
			require.NotEmpty(b, jd.Records)
		case <-time.After(time.Millisecond * 100):
			b.Fatal("chan receive timeout ")
		}

		b.StopTimer()

		server.Server.Stop()
		client.GetGRPCConn().Close()
		extractor.Stop(ctx)
	}
}
