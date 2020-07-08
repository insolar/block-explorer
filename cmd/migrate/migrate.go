// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"

	"github.com/insolar/insconfig"
	"gopkg.in/gormigrate.v1"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/migrations"
)

func main() {
	cfg := &configuration.DB{}
	params := insconfig.Params{
		EnvPrefix:        "migrate",
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	fmt.Println("Starts with configuration:\n", insConfigurator.ToYaml(cfg))

	ctx := context.Background()
	log := belogger.FromContext(ctx)

	db, err := gorm.Open("postgres", cfg.URL)
	if err != nil {
		log.Fatalf("Error while connecting to database: %s", err.Error())
		return
	}
	defer db.Close()

	db = db.LogMode(true)
	db.SetLogger(belogger.NewGORMLogAdapter(log))

	options := gormigrate.DefaultOptions
	options.UseTransaction = true
	options.ValidateUnknownMigrations = true
	m := gormigrate.New(db, options, migrations.Migrations())

	if err = m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
		return
	}
	log.Info("migrated successfully!")
}
