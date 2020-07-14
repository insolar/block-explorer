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
