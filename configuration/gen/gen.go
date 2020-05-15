// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

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
	filePath := ".artifacts/block-explorer.yaml"

	log := belogger.FromContext(context.Background())

	f, err := os.Create(filePath)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to create config file %s", filePath))
		return
	}
	err = insconfig.NewYamlTemplater(new(configuration.BlockExplorer)).TemplateTo(f)

	defer func() {
		err := f.Close()
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to close config file %s", filePath))
		}
	}()

	if err != nil {
		log.Error(errors.Wrapf(err, "failed to write config file %s", filePath))
		return
	}
}