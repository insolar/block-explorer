// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package load

import (
	"context"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type GetLifelineAttack struct {
	loadgen.WithRunner
	c *client.APIClient
}

func (a *GetLifelineAttack) Setup(hc loadgen.RunnerConfig) error {
	a.c = NewGeneratedBEClient(a)
	return nil
}

func (a *GetLifelineAttack) Do(ctx context.Context) loadgen.DoResult {
	objectRef := loadgen.DefaultReadCSV(a)[0]
	_, _, err := a.c.RecordApi.ObjectLifeline(ctx, objectRef, nil)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetLifelineLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetLifelineLabel,
	}
}

func (a *GetLifelineAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetLifelineAttack{WithRunner: loadgen.WithRunner{R: r}}
}
