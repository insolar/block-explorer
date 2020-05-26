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

// NewStorage returns implementation of interfaces.Storage
func NewStorage(db *gorm.DB) interfaces.Storage {
	return &storage{
		db: db,
	}
}

// SaveJetDropData saves provided jetDrop and records to db in one transaction.
func (s *storage) SaveJetDropData(jetDrop models.JetDrop, records []models.Record) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// create zero pulse and zero jetDrop for FK at Record table
		// TODO: remove it at PENV-212
		pulse := models.Pulse{PulseNumber: 1}
		if err := tx.Save(&pulse).Error; err != nil {
			return errors.Wrap(err, "error while saving pulse")
		}
		// TODO: dont rewrite pulse at jetDrop, remove it at PENV-212
		jetDrop.PulseNumber = 1
		if err := tx.Save(&jetDrop).Error; err != nil {
			return errors.Wrap(err, "error while saving jetDrop")
		}

		for _, record := range records {
			// TODO: dont rewrite pulse at record, remove it at PENV-212
			record.PulseNumber = 1
			if err := tx.Save(&record).Error; err != nil {
				return errors.Wrap(err, "error while saving record")
			}
		}

		return nil
	})
}

// GetJetDrops returns records with provided reference from db.
func (s *storage) GetRecord(ref models.Reference) (models.Record, error) {
	record := models.Record{}
	err := s.db.Where("reference = ?", []byte(ref)).First(&record).Error
	return record, err
}

// GetIncompletePulses returns pulses that are not complete from db.
func (s *storage) GetIncompletePulses() ([]models.Pulse, error) {
	var pulses []models.Pulse
	err := s.db.Where("is_complete = ?", false).Find(&pulses).Error
	return pulses, err
}

// GetJetDrops returns jetDrops for provided pulse from db.
func (s *storage) GetJetDrops(pulse models.Pulse) ([]models.JetDrop, error) {
	var jetDrops []models.JetDrop
	err := s.db.Where("pulse_number = ?", pulse.PulseNumber).Find(&jetDrops).Error
	return jetDrops, err
}
