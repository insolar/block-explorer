package load

import (
	"context"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type GetJetDropsByJetIDAttack struct {
	loadgen.WithRunner
	c *client.APIClient
}

func (a *GetJetDropsByJetIDAttack) Setup(hc loadgen.RunnerConfig) error {
	a.c = NewGeneratedBEClient(a)
	return nil
}

func (a *GetJetDropsByJetIDAttack) Do(ctx context.Context) loadgen.DoResult {
	jetID := loadgen.DefaultReadCSV(a)[0]
	_, _, err := a.c.JetDropApi.JetDropsByJetID(ctx, jetID, nil)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetJetDropsByJetIDLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetJetDropsByJetIDLabel,
	}
}

func (a *GetJetDropsByJetIDAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetJetDropsByJetIDAttack{WithRunner: loadgen.WithRunner{R: r}}
}
