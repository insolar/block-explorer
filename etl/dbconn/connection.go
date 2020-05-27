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

func Connect(cfg configuration.DB) (*gorm.DB, error) {
	db, err := gorm.Open("postgres", cfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database")
	}
	db.DB().SetMaxOpenConns(cfg.PoolSize)
	return db, nil
}
