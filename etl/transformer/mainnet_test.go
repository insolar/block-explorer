// +build unit

package transformer

import (
	"context"
	"testing"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
)

func TestTransformer_withDifferentJetId(t *testing.T) {
	ctx := context.Background()
	var differentJetIdCount = 2
	recordGenFunc := testutils.GenerateRecords(differentJetIdCount)

	jetDrops := new(types.PlatformPulseData)
	jets := []exporter.JetDropContinue{}
	for i := 0; i < differentJetIdCount; i++ {
		record, err := recordGenFunc()
		require.NoError(t, err)
		jetDrops.Records = append(jetDrops.Records, record)
		jets = append(jets, exporter.JetDropContinue{JetID: record.Record.JetID})
	}
	pulseNumber := gen.PulseNumber()
	jetDrops.Pulse = &exporter.FullPulse{
		PulseNumber:      pulseNumber,
		PrevPulseNumber:  pulseNumber,
		NextPulseNumber:  pulseNumber,
		Entropy:          insolar.Entropy{},
		PulseTimestamp:   0,
		EpochPulseNumber: 0,
		Jets:             jets,
	}

	dropsCh := make(chan *types.PlatformPulseData)
	var transformer interfaces.Transformer = NewMainNetTransformer(dropsCh, 100)
	err := transformer.Start(ctx)
	require.NoError(t, err)
	defer transformer.Stop(ctx)
	dropsCh <- jetDrops
	jetids := make(map[string][]types.Record)

	for i := 0; i < differentJetIdCount; i++ {
		jd := <-transformer.GetJetDropsChannel()
		require.NotNil(t, jd)
		require.NotNil(t, jd.Sections)
		require.NotNil(t, jd.MainSection)
		mainSection := jd.MainSection
		require.Len(t, mainSection.Records, 1)
		// it's easy to compare integers for testing
		id := mainSection.Start.JetDropPrefix
		jetids[id] = mainSection.Records
	}

	// check that we have received enough records
	require.Len(t, jetids, differentJetIdCount, "received not enough jetdrops from transformer")

	// iterate the map and check with expected value
	for i := 0; i < differentJetIdCount; i++ {
		record := jetDrops.Records[i]
		expectedID := converter.JetIDToString(record.Record.JetID)
		value, hasKey := jetids[expectedID]
		require.True(t, hasKey, "received data from transformer has not expected value")
		require.Equal(t, record.Record.ID.Bytes(), []byte(value[0].Ref))
	}
}
