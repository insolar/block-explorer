// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package extractor

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestExporterIsWorking(t *testing.T) {
	ctx := context.Background()
	server := testutils.CreateTestGRPCServer(t)
	exporter.RegisterRecordExporterServer(server.Server, &RecordExporterServer{})
	server.Serve(t)
	defer server.Server.Stop()

	// prepare config with listening address
	cfg := configuration.Replicator{
		Addr:            server.Network,
		MaxTransportMsg: 100500,
	}

	// initialization Platform connection
	client, err := connection.NewGRPCClientConnection(ctx, cfg)
	require.NoError(t, err)
	defer client.GetGRPCConn().Close()

	g := &RecordExporterClient{}
	extractor := NewPlatformExtractor(uint32(defaultLocalBatchSize), g)
	err = extractor.Start(ctx)
	require.NoError(t, err)
	defer extractor.Stop(ctx)
	jetDrops := extractor.GetJetDrops(ctx)

	for i := 0; i < defaultLocalPulseSize*defaultLocalBatchSize; i++ {
		select {
		case jd := <-jetDrops:
			// when i ∈ [0,1) we received records with some pulse
			// when i ≥ 2 we received records with different pulse, now records from i ∈ [0,1) should be returned
			if i < 1 {
				continue
			}

			t.Logf("i=%d, r=%v", i, jd)
			require.NotEmpty(t, jd.Records)
		case <-time.After(time.Millisecond * 100):
			t.Fatal("chan receive timeout ")
		}
	}
}
