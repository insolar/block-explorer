// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package testutils

import (
	"log"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/ory/dockertest/v3"
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
