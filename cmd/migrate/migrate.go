// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"fmt"

	"github.com/insolar/insconfig"
	"gopkg.in/gormigrate.v1"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/insolar/block-explorer/configuration"
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

	// TODO: enable logger after PENV-279
	// ctx := context.Background()
	// log := belogger.FromContext(ctx)

	db, err := gorm.Open("postgres", cfg.URL)
	if err != nil {
		// TODO: change to logger after PENV-279
		// logger.Fatalf("Error while connecting to database: %s", err.Error())
		fmt.Printf("Error while connecting to database: %s\n", err.Error())
		return
	}
	defer db.Close()

	db = db.LogMode(true)
	// TODO: enable logger after PENV-279
	// db.SetLogger(belogger.NewGORMLogAdapter(log))

	m := gormigrate.New(db, gormigrate.DefaultOptions, migrations.Migrations())

	if err = m.Migrate(); err != nil {
		// TODO: change to logger after PENV-279
		// log.Fatalf("Could not migrate: %v", err)
		fmt.Printf("Could not migrate: %v\n", err)
		return
	}
	// TODO: change to logger after PENV-279
	// log.Info("migrated successfully!")
	fmt.Println("migrated successfully!")
}
