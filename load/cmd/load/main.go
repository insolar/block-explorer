// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"context"
	"errors"
	"flag"
	"strconv"

	"github.com/antihax/optional"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/insolar/insconfig"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/load"
)

type PathGetter struct {
	GoFlags *flag.FlagSet
}

func (g *PathGetter) GetConfigPath() string {
	return "load/migrate_cfg/migrate.yaml"
}

func main() {
	ctx := context.Background()
	log := belogger.FromContext(ctx)

	dbCfg := &configuration.DB{}
	params := insconfig.Params{
		EnvPrefix:        "migrate",
		ConfigPathGetter: &PathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(dbCfg); err != nil {
		panic(err)
	}
	log.Infof("Starts with configuration:\n", insConfigurator.ToYaml(dbCfg))
	beforeAll := func(config *loadgen.GeneratorConfig) error {
		var (
			pulsesToGet     = int32(dbCfg.TestPulses)
			jetDropsToGet   = int32(dbCfg.TestJetDrops)
			recordsToGet    = int32(dbCfg.TestRecords)
			pulsesFileName  = "pulses.csv"
			jetIDSFileName  = "jet_ids.csv"
			objectsFileName = "objects.csv"
		)

		csvPulses, _ := load.NewCSVWriter(pulsesFileName)
		defer csvPulses.Flush()
		csvJetIDS, _ := load.NewCSVWriter(jetIDSFileName)
		defer csvJetIDS.Flush()
		objectsIDS, _ := load.NewCSVWriter(objectsFileName)
		defer objectsIDS.Flush()

		cfg := &client.Configuration{
			BasePath:   config.Generator.Target,
			HTTPClient: loadgen.NewLoggingHTTPClient(false, 10),
		}
		c := client.NewAPIClient(cfg)

		// Get all pulses
		log.Infof("getting pulses: %d", pulsesToGet)
		res, _, err := c.PulseApi.Pulses(ctx, &client.PulsesOpts{
			Limit: optional.NewInt32(pulsesToGet),
		})
		if err != nil {
			return err
		}
		if len(res.Result) == 0 {
			return errors.New("empty pulses, no data")
		}
		pulseNumbers := make([]int64, 0)
		for _, p := range res.Result {
			pn := strconv.FormatInt(p.PulseNumber, 10)
			pulseNumbers = append(pulseNumbers, p.PulseNumber)
			if err := csvPulses.Write([]string{pn}); err != nil {
				log.Fatal(err)
			}
		}

		// Get all uniq jet/pn ids
		log.Infof("getting all uniq jet/pn ids")
		uniqJetDropIds := hashset.New()
		for _, pn := range pulseNumbers {
			res, _, err := c.JetDropApi.JetDropsByPulseNumber(ctx, pn, &client.JetDropsByPulseNumberOpts{
				Limit: optional.NewInt32(jetDropsToGet),
			})
			if err != nil {
				log.Fatal(err)
			}
			for _, uj := range res.Result {
				uniqJetDropIds.Add(uj.JetDropId)
				if err := csvJetIDS.Write([]string{uj.JetId, strconv.FormatInt(uj.PulseNumber, 10)}); err != nil {
					log.Fatal(err)
				}
			}
		}

		// Get all uniq object refs
		log.Infof("getting all uniq objects refs")
		uniqObjectRefs := hashset.New()
		for _, jdID := range uniqJetDropIds.Values() {
			res, _, err := c.RecordApi.JetDropRecords(ctx, jdID.(string), &client.JetDropRecordsOpts{
				Limit: optional.NewInt32(recordsToGet),
			})
			if err != nil {
				log.Fatal(err)
			}
			for _, r := range res.Result {
				uniqObjectRefs.Add(r.ObjectReference)
				if err := objectsIDS.Write([]string{r.ObjectReference}); err != nil {
					log.Fatal(err)
				}
			}
		}
		return nil
	}
	loadgen.Run(load.AttackerFromName, load.CheckFromName, beforeAll, nil)
}
