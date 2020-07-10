package load

import (
	"context"
	"strconv"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type SearchAttack struct {
	loadgen.WithRunner
	c     *client.APIClient
	limit int32
}

func (a *SearchAttack) Setup(hc loadgen.RunnerConfig) error {
	cfg := &client.Configuration{
		BasePath:   a.GetManager().GeneratorConfig.Generator.Target,
		HTTPClient: loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10),
	}
	a.c = client.NewAPIClient(cfg)
	if _, ok := a.GetRunner().Config.Metadata["limit"]; !ok {
		a.limit = 100
	}
	pulses_limit := a.GetRunner().Config.Metadata["limit"]
	l, _ := strconv.Atoi(pulses_limit)
	a.limit = int32(l)
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
