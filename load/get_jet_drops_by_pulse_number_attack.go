// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package load

import (
	"context"
	"strconv"

	"github.com/antihax/optional"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type GetJetDropsByPulseNumberAttack struct {
	loadgen.WithRunner
	c     *client.APIClient
	limit int32
}

func (a *GetJetDropsByPulseNumberAttack) Setup(hc loadgen.RunnerConfig) error {
	a.c = NewGeneratedBEClient(a)
	a.limit = DefaultLimit(a)
	return nil
}

func (a *GetJetDropsByPulseNumberAttack) Do(ctx context.Context) loadgen.DoResult {
	pulse := loadgen.DefaultReadCSV(a)[0]
	pulseNum, _ := strconv.ParseInt(pulse, 10, 64)
	_, _, err := a.c.JetDropApi.JetDropsByPulseNumber(ctx, pulseNum, &client.JetDropsByPulseNumberOpts{
		Limit: optional.NewInt32(a.limit),
	})
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetJetDropsByPulseNumLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetJetDropsByPulseNumLabel,
	}
}

func (a *GetJetDropsByPulseNumberAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetJetDropsByPulseNumberAttack{WithRunner: loadgen.WithRunner{R: r}}
}
