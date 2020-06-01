// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package transformer

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/testutils"
	"github.com/stretchr/testify/require"
)

func TestTransformer_withDifferentJetId(t *testing.T) {
	ctx := context.Background()
	var differentJetIdCount = 2
	recordGenFunc := testutils.GenerateRecords(differentJetIdCount)

	jetDrops := new(types.PlatformJetDrops)
	for i := 0; i < differentJetIdCount; i++ {
		record, err := recordGenFunc()
		require.NoError(t, err)
		jetDrops.Records = append(jetDrops.Records, record)
	}

	dropsCh := make(chan *types.PlatformJetDrops)
	var transformer interfaces.Transformer = NewMainNetTransformer(dropsCh)
	err := transformer.Start(ctx)
	require.NoError(t, err)
	defer transformer.Stop(ctx)
	dropsCh <- jetDrops
	jetids := make(map[uint64][]types.Record)

	for i := 0; i < differentJetIdCount; i++ {
		jd := <-transformer.GetJetDropsChannel()
		require.NotNil(t, jd)
		require.NotNil(t, jd.Sections)
		require.NotNil(t, jd.MainSection)
		mainSection := jd.MainSection
		require.Len(t, mainSection.Records, 1)
		// it's easy to compare integers for testing
		id := binary.BigEndian.Uint64(mainSection.Start.JetDropPrefix)
		jetids[id] = mainSection.Records
	}

	// check that we have received enough records
	require.Len(t, jetids, differentJetIdCount, "received not enough jetdrops from transformer")

	// iterate the map and check with expected value
	for i := 0; i < differentJetIdCount; i++ {
		record := jetDrops.Records[i]
		expectedID := binary.BigEndian.Uint64(record.Record.JetID.Prefix())
		value, hasKey := jetids[expectedID]
		require.True(t, hasKey, "received data from transformer has not expected value")
		require.Equal(t, record.Record.ID.Bytes(), []byte(value[0].Ref))
	}
}
