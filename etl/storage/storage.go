// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

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
	return s.db.Transaction(func(tx *gorm.DB) error {
		// create zero pulse and zero jetDrop for FK at Record table
		// TODO: save pulse correctly at PENV-266
		pulse := models.Pulse{PulseNumber:1}
		if err := tx.Save(&pulse).Error; err != nil {
			return errors.Wrap(err, "error while saving pulse")
		}
		// TODO: save jetDrop correctly at PENV-267
		jetDrop.PulseNumber = 1
		if err := tx.Save(&jetDrop).Error; err != nil {
			return errors.Wrap(err, "error while saving jetDrop")
		}

		for _, record := range records {
			// TODO: dont rewrite pulseNumber, fix it at PENV-266 or PENV-267
			record.PulseNumber = 1
			if err := tx.Save(&record).Error; err != nil {
				return errors.Wrap(err, "error while saving record")
			}
		}

		return nil
	})
}

func (s *storage) GetRecord(ref models.Reference) (models.Record, error) {
	record := models.Record{}
	err := s.db.Where("reference = ?", []byte(ref)).First(&record).Error
	return record, err
}
