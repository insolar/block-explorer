package load

import (
	"context"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type SearchAttack struct {
	loadgen.WithRunner
	c     *client.APIClient
	limit int32
}

func (a *SearchAttack) Setup(hc loadgen.RunnerConfig) error {
	a.c = NewGeneratedBEClient(a)
	a.limit = DefaultLimit(a)
	return nil
}

func (a *SearchAttack) Do(ctx context.Context) loadgen.DoResult {
	query := loadgen.DefaultReadCSV(a)[0]
	_, _, err := a.c.SearchApi.Search(ctx, query)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: SearchLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: SearchLabel,
	}
}

func (a *SearchAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &SearchAttack{WithRunner: loadgen.WithRunner{R: r}}
}
