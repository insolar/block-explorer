package processor

import (
	"context"
	"testing"

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
	p := NewProcessor(trm, sm, 10)
	require.NotNil(t, p)

	require.NoError(t, p.Start(ctx))
	require.NoError(t, p.Stop(ctx))
}
