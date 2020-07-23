// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package dbconn

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/dbconn/plugins"
	"github.com/insolar/block-explorer/testutils"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
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

func TestShutDownPlugin(t *testing.T) {
	dbName := "test_db"
	dbPassword := "secret"
	hostPort, pool, resource, poolCleaner := testutils.RunDBInDockerWithPortBindings(dbName, dbPassword)
	containerID := resource.Container.ID
	defer poolCleaner()

	cfg := configuration.DB{
		URL:             fmt.Sprintf("postgres://postgres:%s@localhost:%d/%s?sslmode=disable", dbPassword, hostPort, dbName),
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Millisecond * 100,
	}
	var db *gorm.DB
	err := pool.Retry(func() error {
		var err error
		db, err = Connect(cfg)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, db)

	stopChannel := make(chan struct{})
	r := plugins.NewDefaultShutdownPlugin(stopChannel)
	r.Apply(db)

	type User struct {
		ID   uint
		Name string
	}

	db.DropTableIfExists(new(User))
	if err := db.AutoMigrate(new(User)).Error; err != nil {
		t.Error(err)
	}

	user := User{ID: 1, Name: "test"}

	// try to save and it's working
	err = db.Save(&User{ID: 100, Name: "test user"}).Error
	require.NoError(t, err)

	var called int32 = 0
	db.Callback().Update().Register("TestShutDownPlugin", func(scope *gorm.Scope) {
		called = called + 1
		atomic.CompareAndSwapInt32(&called, 0, 1)
	})

	err = pool.Client.StopContainer(containerID, 0)
	require.NoError(t, err)
	_, err = pool.Client.WaitContainer(containerID)
	require.NoError(t, err)

	// no need to wait until the connection return error
	go func() {
		// try to do save and for getting error
		err = db.Save(&user).Error
		require.Error(t, err)
	}()

	select {
	case <-stopChannel:
		// error happened
	case <-time.After(time.Millisecond * 100):
		t.Fatal("chan receive timeout. Stop signal was not received")
	}

	require.Equal(t, atomic.LoadInt32(&called), int32(1), "plugin should be called once")

	err = pool.Client.StartContainer(containerID, nil)
	require.NoError(t, err)

	// wait for container
	err = pool.Retry(func() error {
		return db.DB().Ping()
	})
	require.NoError(t, err)

	// try to exec the query successfully
	err = db.Save(&user).Error
	require.NoError(t, err)
}
