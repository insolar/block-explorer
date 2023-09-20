package load

import (
	"context"
	"net/http"
	"strconv"

	"github.com/skudasov/loadgen"

	"github.com/insolar/block-explorer/etl/models"
)

type GetRecordsAttack struct {
	loadgen.WithRunner
	rc *http.Client
}

func (a *GetRecordsAttack) Setup(hc loadgen.RunnerConfig) error {
	a.rc = loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10)
	return nil
}

func (a *GetRecordsAttack) Do(ctx context.Context) loadgen.DoResult {
	d := loadgen.DefaultReadCSV(a)
	jetDropID := d[0]
	pulseNumber := d[1]
	pn, _ := strconv.ParseInt(pulseNumber, 10, 64)
	id := models.NewJetDropID(jetDropID, pn).ToString()
	err := GetRecordsByID(a, id)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetRecordsLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetRecordsLabel,
	}
}

func (a *GetRecordsAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetRecordsAttack{WithRunner: loadgen.WithRunner{R: r}}
}
