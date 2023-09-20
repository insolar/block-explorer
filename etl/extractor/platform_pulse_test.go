// +build unit

package extractor

import (
	"context"
	"errors"
	"testing"

	"github.com/insolar/block-explorer/testutils/clients"
	"github.com/stretchr/testify/require"
)

func TestPulse_getCurrentPulse_Success(t *testing.T) {
	ctx := context.Background()
	expectedPulse := uint32(0)
	client := clients.GetTestPulseClient(expectedPulse, nil)

	pe := NewPlatformPulseExtractor(client)
	currentPulse, err := pe.GetCurrentPulse(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedPulse, currentPulse)
}

func TestPulse_getCurrentPulse_Fail(t *testing.T) {
	ctx := context.Background()
	client := clients.GetTestPulseClient(1, errors.New("test error"))

	pe := NewPlatformPulseExtractor(client)
	_, err := pe.GetCurrentPulse(ctx)
	require.Error(t, err)
}
