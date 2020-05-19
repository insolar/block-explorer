package processor

import (
	"context"
	"testing"

	"github.com/insolar/block-explorer/etl/models"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/stretchr/testify/require"
)

func TestNewProcessor(t *testing.T) {

	ctx := context.Background()
	JDC := make(chan types.JetDrop)
	trm := mock.NewTransformerMock(t)
	trm.GetJetDropsChannelMock.Set(func() (ch1 <-chan types.JetDrop) {
		return JDC
	})

	sm := mock.NewStorageMock(t)
	sm.SaveJetDropDataMock.Set(func(jetDrop models.JetDrop, records []models.Record) (err error) {
		return nil
	})

	p := NewProcessor(trm, sm, 0)
	require.NotNil(t, p)

	require.NoError(t, p.Start(ctx))

	for i := 0; i < 5; i++ {
		JDC <- types.JetDrop{
			MainSection: &types.MainSection{
				Start: types.DropStart{
					PulseData:           types.Pulse{},
					JetDropPrefix:       nil,
					JetDropPrefixLength: 0,
				},
				DropContinue: types.DropContinue{},
				Sections:     nil,
				Records:      nil,
			},
			Sections: nil,
			RawData:  nil,
		}
	}

	require.NoError(t, p.Stop(ctx))
}
