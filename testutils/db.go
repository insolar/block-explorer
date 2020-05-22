// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/ory/dockertest/v3"
	"github.com/pkg/errors"
	"gopkg.in/gormigrate.v1"

	"github.com/insolar/block-explorer/migrations"
)

func RunDBInDocker(dbName, dbPassword string) (*dockertest.Pool, *dockertest.Resource, func()) {
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run(
		"postgres", "12",
		[]string{
			"POSTGRES_DB=" + dbName,
			"POSTGRES_PASSWORD=" + dbPassword,
		},
	)
	if err != nil {
		log.Panicf("Could not start resource: %s", err)
	}

	poolCleaner := func() {
		// When you're done, kill and remove the container
		err := pool.Purge(resource)
		if err != nil {
			log.Printf("failed to purge docker pool: %s", err)
		}
	}
	return pool, resource, poolCleaner
}

func SetupDB() (*gorm.DB, func(), error) {
	dbName := "test_db"
	dbPassword := "secret"
	pool, resource, poolCleaner := RunDBInDocker(dbName, dbPassword)
	dbURL := fmt.Sprintf("postgres://postgres:%s@localhost:%s/%s?sslmode=disable", dbPassword, resource.GetPort("5432/tcp"), dbName)

	var db *gorm.DB
	err := pool.Retry(func() error {
		var err error

		db, err = gorm.Open("postgres", dbURL)
		if err != nil {
			return err
		}
		err = db.Exec("select 1").Error
		return err
	})
	if err != nil {
		poolCleaner()
		return nil, nil, errors.Wrap(err, "Could not start postgres:")
	}

	dbCleaner := func() {
		err := db.Close()
		if err != nil {
			log.Printf("failed to purge docker pool: %s", err)
		}
	}
	cleaner := func() {
		dbCleaner()
		poolCleaner()
	}

	m := gormigrate.New(db, gormigrate.DefaultOptions, migrations.Migrations())

	if err = m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}

	return db, cleaner, nil
}
