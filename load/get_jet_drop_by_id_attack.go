package load

import (
	"context"
	"net/http"
	"strconv"

	"github.com/skudasov/loadgen"

	"github.com/insolar/block-explorer/etl/models"
)

type GetJetDropByIDAttack struct {
	loadgen.WithRunner
	rc *http.Client
}

func (a *GetJetDropByIDAttack) Setup(hc loadgen.RunnerConfig) error {
	a.rc = loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10)
	return nil
}

func (a *GetJetDropByIDAttack) Do(ctx context.Context) loadgen.DoResult {
	d := loadgen.DefaultReadCSV(a)
	jetDropID := d[0]
	pulseNumber := d[1]
	pn, _ := strconv.ParseInt(pulseNumber, 10, 64)
	id := models.NewJetDropID(jetDropID, pn).ToString()
	// swagger "allowReserved" is not working in our go-codegen tools, generated client jet drop id is still urlEncoded
	// by default and we get 400 when using generated client
	err := GetJetDropsByID(a, id)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetJetDropByIDLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetJetDropByIDLabel,
	}
}

func (a *GetJetDropByIDAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetJetDropByIDAttack{WithRunner: loadgen.WithRunner{R: r}}
}
