// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package plugins

import (
	"database/sql/driver"
	"regexp"

	"github.com/jinzhu/gorm"
)

var badConnectRegexp = regexp.MustCompile("(connection refused|invalid connection)$")

// Shutdown GORM plugin
type Shutdown struct {
	stopChannel    chan struct{}
	badConnChecker func(errors []error) bool
}

// NewDefaultShutdownPlugin initialize GORM plugin
func NewDefaultShutdownPlugin(stopChannel chan struct{}) *Shutdown {
	return &Shutdown{
		stopChannel:    stopChannel,
		badConnChecker: defaultConnectionChecker,
	}
}

// NewShutdownPlugin initialize GORM plugin
func NewShutdownPlugin(stopChannel chan struct{}, badConnChecker func(errors []error) bool) *Shutdown {
	return &Shutdown{
		stopChannel:    stopChannel,
		badConnChecker: badConnChecker,
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

// Apply apply shutdown callbacks to GORM DB instance
func (shutdown *Shutdown) Apply(db *gorm.DB) {
	db.Callback().Create().Register("gbe:gorm:plugins:shutdown", shutdown.shutdownCallback)
	db.Callback().Update().Register("gbe:gorm:plugins:shutdown", shutdown.shutdownCallback)
	db.Callback().Delete().Register("gbe:gorm:plugins:shutdown", shutdown.shutdownCallback)
	db.Callback().Query().Register("gbe:gorm:plugins:shutdown", shutdown.shutdownCallback)
	db.Callback().RowQuery().Register("gbe:gorm:plugins:shutdown", shutdown.shutdownCallback)
}

// if callback was called and no connection to database,
// need to stop the application gracefully
func (shutdown *Shutdown) shutdownCallback(scope *gorm.Scope) {
	if scope.HasError() {
		// check the error message
		if db := scope.DB(); shutdown.badConnChecker(db.GetErrors()) {
			connected := db.DB().Ping() == nil
			if !connected {
				// stop the application gracefully
				shutdown.stopChannel <- struct{}{}
			}
		}
	}
}
