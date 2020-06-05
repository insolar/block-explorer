// +build bench

package extractor

import (
	"context"
	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func BenchmarkPlatformExtractorGetJetDrops(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		ctx := context.Background()
		server := testutils.CreateTestGRPCServer(b)
		exporter.RegisterRecordExporterServer(server.Server, &Gserver{})
		server.Serve(b)

		// prepare config with listening address
		cfg := configuration.Replicator{
			Addr:            server.Network,
			MaxTransportMsg: 100500,
		}

		// initialization Platform connection
		client, err := connection.NewGRPCClientConnection(ctx, cfg)
		require.NoError(b, err)

		g := &Gclient{}
		extractor := NewPlatformExtractor(uint32(defaultLocalBatchSize), g)
		err = extractor.Start(ctx)
		require.NoError(b, err)

		b.StartTimer()
		jetDrops := extractor.GetJetDrops(ctx)
		for i := 0; i < defaultLocalPulseSize*defaultLocalBatchSize; i++ {
			select {
			case jd := <-jetDrops:
				// when i ∈ [0,1) we received records with some pulse
				// when i ≥ 2 we received records with different pulse, now records from i ∈ [0,1) should be returned
				if i < 1 {
					continue
				}
				require.NotEmpty(b, jd.Records)
			case <-time.After(time.Millisecond * 100):
				b.Fatal("chan receive timeout ")
			}
		}
		b.StopTimer()

		server.Server.Stop()
		client.GetGRPCConn().Close()
		extractor.Stop(ctx)
	}
}
