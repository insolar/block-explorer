// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"strconv"
	"strings"

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
		if err := tx.Save(&jetDrop).Error; err != nil {
			return errors.Wrap(err, "error while saving jetDrop")
		}

		for _, record := range records {
			if err := tx.Save(&record).Error; err != nil { // nolint
				return errors.Wrap(err, "error while saving record")
			}
		}

		return nil
	})
}

// SavePulse saves provided pulse to db.
func (s *storage) SavePulse(pulse models.Pulse) error {
	return errors.Wrap(s.db.Save(&pulse).Error, "error while saving pulse")
}

// CompletePulse update pulse with provided number to completeness in db.
func (s *storage) CompletePulse(pulseNumber int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		pulse := models.Pulse{PulseNumber: pulseNumber}
		update := tx.Model(&pulse).Update(models.Pulse{IsComplete: true})
		if update.Error != nil {
			return errors.Wrap(update.Error, "error while updating pulse completeness")
		}
		rowsAffected := update.RowsAffected
		if rowsAffected == 0 {
			return errors.Errorf("try to complete not existing pulse with number %d", pulseNumber)
		}
		if rowsAffected != 1 {
			return errors.Errorf("several rows were affected by update for pulse with number %d, it was not expected", pulseNumber)
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

func checkIndex(i string) (int, int, error) {
	index := strings.Split(i, ":")
	if len(index) != 2 {
		return 0, 0, errors.New("query parameter 'index' should have the '<pulse_number>:<order>' format")
	}
	var err error
	var pulseNumber, order int64
	pulseNumber, err = strconv.ParseInt(index[0], 10, 64)
	if err != nil {
		return 0, 0, errors.New("query parameter 'index' should have the '<pulse_number>:<order>' format")
	}
	order, err = strconv.ParseInt(index[1], 10, 64)
	if err != nil {
		return 0, 0, errors.New("query parameter 'index' should have the '<pulse_number>:<order>' format")
	}
	return int(pulseNumber), int(order), nil
}

func filterByPulse(query *gorm.DB, pulseNumberLt, pulseNumberGt *int) *gorm.DB {
	if pulseNumberGt != nil {
		query = query.Where("pulse_number > ?", *pulseNumberGt)
	}
	if pulseNumberLt != nil {
		query = query.Where("pulse_number < ?", *pulseNumberLt)
	}
	return query
}

func filterRecordsByIndex(query *gorm.DB, fromIndex string, sort string) (*gorm.DB, error) {
	pulseNumber, order, err := checkIndex(fromIndex)
	if err != nil {
		return nil, err
	}
	switch sort {
	case "asc":
		query = query.Where("(pulse_number > ?", pulseNumber)
		query = query.Or("pulse_number = ? AND \"order\" >= ?)", pulseNumber, order)
	case "desc":
		query = query.Where("(pulse_number < ?", pulseNumber)
		query = query.Or("pulse_number = ? AND \"order\" <= ?)", pulseNumber, order)

	default:
		return nil, errors.New("query parameter 'sort' should be 'asc'' or 'desc'")
	}
	return query, nil
}

func sortRecordsByDirection(query *gorm.DB, sort string) (*gorm.DB, error) {
	switch sort {
	case "asc":
		query = query.Order("pulse_number asc").Order("\"order\" asc")
	case "desc":
		query = query.Order("pulse_number desc").Order("\"order\" desc")
	default:
		return nil, errors.New("query parameter 'sort' should be 'asc'' or 'desc'")
	}
	return query, nil
}

func getRecords(query *gorm.DB, limit, offset int) ([]models.Record, int, error) {
	records := []models.Record{}
	var total int
	err := query.Limit(limit).Offset(offset).Find(&records).Error
	if err != nil {
		return nil, 0, err
	}
	err = query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// GetLifeline returns records for provided object reference, ordered by pulse number and order fields.
func (s *storage) GetLifeline(objRef []byte, fromIndex *string, pulseNumberLt, pulseNumberGt *int, limit, offset int, sort string) ([]models.Record, int, error) {
	query := s.db.Model(&models.Record{}).Where("object_reference = ?", objRef).Where("type = ?", models.State)

	query = filterByPulse(query, pulseNumberLt, pulseNumberGt)

	var err error
	if fromIndex != nil {
		query, err = filterRecordsByIndex(query, *fromIndex, sort)
		if err != nil {
			return nil, 0, err
		}
	}

	query, err = sortRecordsByDirection(query, sort)
	if err != nil {
		return nil, 0, err
	}

	records, total, err := getRecords(query, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "error while select records for object %v from db", objRef)
	}
	return records, total, nil
}

// GetPulse returns pulse with provided pulse number from db.
func (s *storage) GetPulse(pulseNumber int) (models.Pulse, int64, int64, error) {
	var pulse models.Pulse
	err := s.db.Where("pulse_number = ?", pulseNumber).First(&pulse).Error
	if err != nil {
		return pulse, 0, 0, err
	}

	pulse = s.updateNextPulse(pulse)

	jetDrops, records, err := s.GetAmounts(pulseNumber)
	if err != nil {
		return pulse, 0, 0, errors.Wrapf(err, "error while select count of records from db for pulse number %d", pulseNumber)
	}

	return pulse, jetDrops, records, err
}

// GetAmounts return amount of jetDrops and records at provided pulse.
func (s *storage) GetAmounts(pulseNumber int) (int64, int64, error) {
	res := struct {
		JetDrops int
		Records  int
	}{}
	err := s.db.Model(models.JetDrop{}).Select("count(*) as jet_drops, sum(record_amount) as records").Where("pulse_number = ?", pulseNumber).Scan(&res).Error
	if err != nil {
		return 0, 0, errors.Wrapf(err, "error while select count of records from db for pulse number %d", pulseNumber)
	}

	return int64(res.JetDrops), int64(res.Records), err
}

func (s *storage) updateNextPulse(pulse models.Pulse) models.Pulse {
	var nextPulse models.Pulse
	err := s.db.Where("prev_pulse_number = ?", pulse.PulseNumber).First(&nextPulse).Error
	if err != nil {
		return pulse
	}
	pulse.NextPulseNumber = nextPulse.PulseNumber
	return pulse
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
