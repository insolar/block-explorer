package main

import (
	"context"
	"os"

	"github.com/insolar/insconfig"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

func main() {
	configs := map[string]interface{}{
		".artifacts/block-explorer.yaml":  configuration.BlockExplorer{},
		".artifacts/migrate.yaml":         configuration.DB{},
		".artifacts/api.yaml":             configuration.API{},
		".artifacts/exporter-api.yaml":    configuration.Exporter{},
		"./load/migrate_cfg/migrate.yaml": configuration.TestDB{},
	}

	log := belogger.FromContext(context.Background())

	for filePath, config := range configs {
		func() {
			f, err := os.Create(filePath)
			if err != nil {
				log.Fatal(errors.Wrapf(err, "failed to create config file %s", filePath))
			}
			err = insconfig.NewYamlTemplater(config).TemplateTo(f)

			defer func() {
				err := f.Close()
				if err != nil {
					log.Fatal(errors.Wrapf(err, "failed to close config file %s", filePath))
				}
			}()

			if err != nil {
				log.Fatal(errors.Wrapf(err, "failed to write config file %s", filePath))
			}
		}()
	}
}
