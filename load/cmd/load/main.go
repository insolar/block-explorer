package main

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/antihax/optional"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/skudasov/loadgen"

	"github.com/insolar/block-explorer/load"
)

func main() {
	beforeAll := func(config *loadgen.DefaultGeneratorConfig) error {
		var (
			pulsesToGet     int32 = 100
			pulsesFileName        = "pulses.csv"
			jetIDSFileName        = "jet_ids.csv"
			objectsFileName       = "objects.csv"
		)

		csvPulses, _ := load.NewCSVWriter(pulsesFileName)
		defer csvPulses.Flush()
		csvJetIDS, _ := load.NewCSVWriter(jetIDSFileName)
		defer csvJetIDS.Flush()
		objectsIDS, _ := load.NewCSVWriter(objectsFileName)
		defer objectsIDS.Flush()

		ctx := context.Background()
		cfg := &client.Configuration{
			BasePath:   config.Generator.Target,
			HTTPClient: loadgen.NewLoggingHTTPClient(false, 10),
		}
		c := client.NewAPIClient(cfg)

		// Get all pulses
		log.Printf("getting pulses: %d", pulsesToGet)
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
		log.Printf("getting all uniq jet/pn ids")
		uniqJetDropIds := hashset.New()
		for _, pn := range pulseNumbers {
			res, _, err := c.JetDropApi.JetDropsByPulseNumber(ctx, pn, nil)
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
		log.Printf("getting all uniq objects refs")
		uniqObjectRefs := hashset.New()
		for _, jdID := range uniqJetDropIds.Values() {
			res, _, err := c.RecordApi.JetDropRecords(ctx, jdID.(string), nil)
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
