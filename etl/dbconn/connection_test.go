// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package dbconn

import (
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/testutils"
)

func TestConnect(t *testing.T) {
	dbName := "test_db"
	dbPassword := "secret"
	pool, resource, poolCleaner := testutils.RunDBInDocker(dbName, dbPassword)
	defer poolCleaner()

	cfg := configuration.DB{
		URL: fmt.Sprintf("postgres://postgres:%s@localhost:%s/%s?sslmode=disable", dbPassword, resource.GetPort("5432/tcp"), dbName),
		PoolSize: 100,
	}
	var db *gorm.DB
	err := pool.Retry(func() error {
		var err error
		db, err = Connect(cfg)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, db)
}

func TestConnect_WrongURL(t *testing.T) {
	cfg := configuration.DB{
		URL: "wrong_url",
	}
	db, err := Connect(cfg)
	require.Error(t, err)
	require.Nil(t, db)
}
