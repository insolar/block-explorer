package main

import (
	"context"
	"encoding/csv"
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
		ctx := context.Background()
		cfg := &client.Configuration{
			BasePath:   config.Generator.Target,
			HTTPClient: loadgen.NewLoggingHTTPClient(false, 10),
		}
		c := client.NewAPIClient(cfg)

		// Get all pulses
		res, _, err := c.PulseApi.Pulses(ctx, &client.PulsesOpts{
			Limit: optional.NewInt32(100),
		})
		if err != nil {
			return err
		}
		if len(res.Result) == 0 {
			return errors.New("empty pulses, no data")
		}
		pulsesFile := loadgen.CreateOrReplaceFile("pulses.csv")
		defer pulsesFile.Close()
		csvPulses := csv.NewWriter(pulsesFile)
		pulseNumbers := make([]int64, 0)
		for _, p := range res.Result {
			pn := strconv.FormatInt(p.PulseNumber, 10)
			pulseNumbers = append(pulseNumbers, p.PulseNumber)
			if err := csvPulses.Write([]string{pn}); err != nil {
				log.Fatal(err)
			}
		}
		csvPulses.Flush()

		// Get all uniq jet/pn ids
		jetIDSFile := loadgen.CreateOrReplaceFile("jet_ids.csv")
		defer jetIDSFile.Close()
		csvJetIDS := csv.NewWriter(jetIDSFile)
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
		csvJetIDS.Flush()

		// Get all uniq object refs
		objectsFile := loadgen.CreateOrReplaceFile("objects.csv")
		defer objectsFile.Close()
		objectsIDS := csv.NewWriter(objectsFile)
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
		objectsIDS.Flush()
		return nil
	}
	loadgen.Run(load.AttackerFromName, load.CheckFromName, beforeAll, nil)
}
