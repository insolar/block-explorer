// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package dbconn

import (
	"github.com/jinzhu/gorm"
	// import database's driver
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/configuration"
)

// Connect returns connection to database
func Connect(cfg configuration.DB) (*gorm.DB, error) {
	return ConnectFn(cfg)()
}

// ConnectFn returns the function which can be used to connect
func ConnectFn(cfg configuration.DB) func() (*gorm.DB, error) {
	return func() (*gorm.DB, error) {
		db, err := gorm.Open("postgres", cfg.URL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open database")
		}
		db.DB().SetMaxOpenConns(cfg.MaxOpenConns)
		db.DB().SetMaxIdleConns(cfg.MaxIdleConns)
		db.DB().SetConnMaxLifetime(cfg.ConnMaxLifetime)
		return db, nil
	}
}
