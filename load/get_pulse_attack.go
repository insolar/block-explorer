package load

import (
	"context"
	"strconv"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type GetPulseAttack struct {
	loadgen.WithRunner
	c *client.APIClient
}

func (a *GetPulseAttack) Setup(hc loadgen.RunnerConfig) error {
	cfg := &client.Configuration{
		BasePath:   a.GetManager().GeneratorConfig.Generator.Target,
		HTTPClient: loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10),
	}
	a.c = client.NewAPIClient(cfg)
	return nil
}

func (a *GetPulseAttack) Do(ctx context.Context) loadgen.DoResult {
	pulse := loadgen.DefaultReadCSV(a)[0]
	pulseNum, _ := strconv.ParseInt(pulse, 10, 64)
	_, _, err := a.c.PulseApi.Pulse(ctx, pulseNum)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetPulseLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetPulseLabel,
	}
}

func (a *GetPulseAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetPulseAttack{WithRunner: loadgen.WithRunner{R: r}}
}
