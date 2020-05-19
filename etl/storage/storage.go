// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"github.com/jinzhu/gorm"

	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/models"
)

type storage struct {
	db *gorm.DB
}

func NewStorage(db *gorm.DB) interfaces.Storage {
	return &storage{
		db: db,
	}
}

func (s *storage) SaveJetDropData(jetDrop models.JetDrop, records []models.Record) error {
	panic("not implemented")
}
