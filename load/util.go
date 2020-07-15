// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package load

import (
	"encoding/csv"
	"strconv"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"
)

func NewCSVWriter(name string) (*csv.Writer, func() error) {
	f := loadgen.CreateOrReplaceFile(name)
	return csv.NewWriter(f), f.Close
}

func NewGeneratedBEClient(a loadgen.Attack) *client.APIClient {
	cfg := &client.Configuration{
		BasePath:   a.GetManager().GeneratorConfig.Generator.Target,
		HTTPClient: loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 10),
	}
	return client.NewAPIClient(cfg)
}

func DefaultLimit(a loadgen.Attack) int32 {
	var limit int32
	if _, ok := a.GetRunner().Config.Metadata["limit"]; !ok {
		limit = 100
	} else {
		pulsesLimit := a.GetRunner().Config.Metadata["limit"]
		l, err := strconv.ParseInt(pulsesLimit, 10, 0)
		if err != nil {
			a.GetRunner().L.Fatal(err)
		}
		limit = int32(l)
	}
	return limit
}
