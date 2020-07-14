package load

import (
	"context"
	"strconv"

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
	cfg := &client.Configuration{
		BasePath:   a.GetManager().GeneratorConfig.Generator.Target,
		HTTPClient: loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10),
	}
	a.c = client.NewAPIClient(cfg)
	if _, ok := a.GetRunner().Config.Metadata["limit"]; !ok {
		a.limit = 100
	}
	pulsesLimit := a.GetRunner().Config.Metadata["limit"]
	l, err := strconv.ParseInt(pulsesLimit, 10, 0)
	if err != nil {
		a.R.L.Fatal(err)
	}
	a.limit = int32(l)
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
