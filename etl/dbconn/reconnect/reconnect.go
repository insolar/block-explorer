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

	"github.com/insolar/block-explorer/configuration"
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
func New(cfg configuration.Reconnect, connectionFn func() (*gorm.DB, error)) *Reconnect {
	config := &Config{
		Attempts:       cfg.Attempts,
		Interval:       cfg.Interval,
		BadConnChecker: defaultConnectionChecker,
	}

	return &Reconnect{
		mutex:        sync.Mutex{},
		Config:       config,
		connectionFn: connectionFn,
	}
}

// defaultConnectionChecker checks is network error received or not
func defaultConnectionChecker(errors []error) bool {
	for _, err := range errors {
		if err == driver.ErrBadConn || badConnectRegexp.MatchString(err.Error()) {
			return true
		}
	}
	return false
}

// Apply apply reconnect to GORM DB instance
func (reconnect *Reconnect) Apply(db *gorm.DB) {
	db.Callback().Create().Before("gbe:gorm:plugins:reconnect").
		Register("gbe:gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().Update().Before("gorm:plugins:reconnect").
		Register("gbe:gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().Delete().Before("gorm:plugins:reconnect").
		Register("gbe:gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().Query().Before("gorm:plugins:reconnect").
		Register("gbe:gorm:plugins:reconnect", reconnect.generateCallback)
	db.Callback().RowQuery().Before("gorm:plugins:reconnect").
		Register("gbe:gorm:plugins:reconnect", reconnect.generateCallback)
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
