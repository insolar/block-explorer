// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package dbconn

import (
	"fmt"
	"testing"
	"time"

	"github.com/insolar/block-explorer/etl/dbconn/reconnect"
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
		URL:          fmt.Sprintf("postgres://postgres:%s@localhost:%s/%s?sslmode=disable", dbPassword, resource.GetPort("5432/tcp"), dbName),
		MaxOpenConns: 100,
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

func TestReconnect(t *testing.T) {
	dbName := "test_db"
	dbPassword := "secret"
	hostPort, pool, resource, poolCleaner := testutils.RunDBInDockerWithPortBindings(dbName, dbPassword)
	containerID := resource.Container.ID
	defer poolCleaner()

	cfg := configuration.DB{
		URL:          fmt.Sprintf("postgres://postgres:%s@localhost:%d/%s?sslmode=disable", dbPassword, hostPort, dbName),
		MaxOpenConns: 100,
		Reconnect: configuration.Reconnect{
			Attempts: 100,
			Interval: 3 * time.Second,
		},
	}
	var db *gorm.DB
	connectFn := ConnectFn(cfg)
	err := pool.Retry(func() error {
		var err error
		db, err = connectFn()
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, db)

	r := reconnect.New(cfg.Reconnect, connectFn)
	r.Apply(db)

	// try to do select and it working
	err = db.Raw("select 1").Error
	require.NoError(t, db.Raw("select 1").Error)

	err = pool.Client.StopContainer(containerID, 0)
	require.NoError(t, err)
	_, err = pool.Client.WaitContainer(containerID)
	require.NoError(t, err)

	// try to do select and for getting error
	err = db.Raw("select 1").Error
	require.Nil(t, db.Raw("select 1").Error)

	err = pool.Client.StartContainer(containerID, nil)
	require.NoError(t, err)

	// try to do select
	err = db.Raw("select 1").Error
	require.NoError(t, db.Raw("select 1").Error)
}
