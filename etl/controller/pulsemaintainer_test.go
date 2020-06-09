// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package controller

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/insolar/block-explorer/configuration"

	"github.com/insolar/block-explorer/etl/interfaces/mock"
	"github.com/insolar/block-explorer/etl/types"
	"github.com/stretchr/testify/require"
)

func TestController_pulseMaintainer(t *testing.T) {
	var cfg = configuration.Controller{PulsePeriod: 0}

	extractor := mock.NewJetDropsExtractorMock(t)

	sm := mock.NewStorageMock(t)
	sm.GetIncompletePulsesMock.Return(nil, nil)

	defer leaktest.Check(t)()
	c, err := NewController(cfg, extractor, sm)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, c.Start(ctx))
	require.NoError(t, c.Stop(ctx))
	time.Sleep(time.Millisecond)
}

func Test_pulseIsComplete(t *testing.T) {
	require.True(t, pulseIsComplete(types.Pulse{}, nil))
}
