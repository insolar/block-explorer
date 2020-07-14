package load

import (
	"context"

	"github.com/antihax/optional"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type GetPulsesAttack struct {
	loadgen.WithRunner
	c     *client.APIClient
	limit int32
}

func (a *GetPulsesAttack) Setup(hc loadgen.RunnerConfig) error {
	a.c = NewGeneratedBEClient(a)
	a.limit = DefaultLimit(a)
	return nil
}

func (a *GetPulsesAttack) Do(ctx context.Context) loadgen.DoResult {
	_, _, err := a.c.PulseApi.Pulses(ctx, &client.PulsesOpts{
		Limit: optional.NewInt32(a.limit),
	})
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetPulsesLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetPulsesLabel,
	}
}

func (a *GetPulsesAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetPulsesAttack{WithRunner: loadgen.WithRunner{R: r}}
}
