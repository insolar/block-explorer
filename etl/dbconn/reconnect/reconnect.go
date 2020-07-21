// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package reconnect

import (
	"database/sql/driver"
	"regexp"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
)

var badConnectRegexp = regexp.MustCompile("(connection refused|invalid connection)$")

// Reconnect GORM reconnect plugin
type Reconnect struct {
	Config *Config
	mutex  sync.Mutex

	connectionFn func() (*gorm.DB, error)
}

// Config reconnect config
type Config struct {
	Attempts       int
	Interval       time.Duration
	BadConnChecker func(errors []error) bool
}

// New initialize GORM reconnect DB
func New(config *Config, connectionFn func() (*gorm.DB, error)) *Reconnect {
	if config == nil {
		config = &Config{}
	}

	if config.BadConnChecker == nil {
		config.BadConnChecker = func(errors []error) bool {
			for _, err := range errors {
				if err == driver.ErrBadConn || badConnectRegexp.MatchString(err.Error()) {
					return true
				}
			}
			return false
		}
	}

	if config.Attempts == 0 {
		config.Attempts = 5
	}

	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}

	return &Reconnect{
		mutex:        sync.Mutex{},
		Config:       config,
		connectionFn: connectionFn,
	}
}

// Apply apply reconnect to GORM DB instance
func (reconnect *Reconnect) Apply(db *gorm.DB) {
	db.Callback().Create().Before("gorm:plugins:reconnect").
		Register("gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().Update().Before("gorm:plugins:reconnect").
		Register("gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().Delete().Before("gorm:plugins:reconnect").
		Register("gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().Query().Before("gorm:plugins:reconnect").
		Register("gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().RowQuery().Before("gorm:plugins:reconnect").
		Register("gorm:plugins:reconnect", reconnect.generateCallback)
}

// if callback was called and no connection to database,
// need to reconnect to the database and continue
func (reconnect *Reconnect) generateCallback(scope *gorm.Scope) {
	if scope.HasError() {
		// check the error message
		if db := scope.DB(); reconnect.Config.BadConnChecker(db.GetErrors()) {
			reconnect.mutex.Lock()

			connected := db.DB().Ping() == nil

			if !connected {
				for i := 0; i < reconnect.Config.Attempts; i++ {
					if err := reconnect.reconnectDB(scope); err == nil {
						break
					}
					time.Sleep(reconnect.Config.Interval)
				}
			}
			reconnect.mutex.Unlock()
		}
	}
}

func (reconnect *Reconnect) reconnectDB(scope *gorm.Scope) error {
	var (
		db         = scope.DB()
		sqlDB      = db.DB()
		newDB, err = reconnect.connectionFn()
	)

	if newDB == nil {
		return err
	}
	err = newDB.DB().Ping()

	if err == nil {
		db.Error = nil
		*sqlDB = *newDB.DB()
	}

	return err
}
