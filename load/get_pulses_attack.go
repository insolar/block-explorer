package load

import (
	"context"
	"fmt"
	"strconv"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

type GetPulsesAttack struct {
	loadgen.WithRunner
	c *client.APIClient
}

func (a *GetPulsesAttack) Setup(hc loadgen.RunnerConfig) error {
	cfg := &client.Configuration{
		BasePath:   fmt.Sprintf(a.GetManager().GeneratorConfig.Generator.Target),
		HTTPClient: loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10),
	}
	a.c = client.NewAPIClient(cfg)
	return nil
}

func (a *GetPulsesAttack) Do(ctx context.Context) loadgen.DoResult {
	pulse := loadgen.DefaultReadCSV(a)[0]
	pulseNum, _ := strconv.ParseInt(pulse, 10, 64)
	pr, _, err := a.c.PulseApi.Pulse(ctx, pulseNum)
	a.R.L.Infof("resp: %v", pr)
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
