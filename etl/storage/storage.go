// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"database/sql"
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

func CheckIndex(i string) (int, int, error) {
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

func filterByJetDropId(query *gorm.DB, jetDropIDGte, jetDropIDLte *models.JetDropID) *gorm.DB {
	if jetDropIDGte != nil {
		query = query.Where("(pulse_number >= ? and jet_id >= ?)", jetDropIDGte.PulseNumber, jetDropIDGte.JetID)
	}

	if jetDropIDLte != nil {
		query = query.Where("(pulse_number <= ? and jet_id <= ?)", jetDropIDLte.PulseNumber, jetDropIDLte.JetID)
	}

	return query
}

func filterRecordsByIndex(query *gorm.DB, fromIndex string, sort string) (*gorm.DB, error) {
	pulseNumber, order, err := CheckIndex(fromIndex)
	if err != nil {
		return nil, err
	}
	switch sort {
	case "+index":
		query = query.Where("(pulse_number > ?", pulseNumber)
		query = query.Or("pulse_number = ? AND \"order\" >= ?)", pulseNumber, order)
	case "-index":
		query = query.Where("(pulse_number < ?", pulseNumber)
		query = query.Or("pulse_number = ? AND \"order\" <= ?)", pulseNumber, order)

	default:
		return nil, errors.New("query parameter 'sort' should be 'asc'' or 'desc'")
	}
	return query, nil
}

func filterByTimestamp(query *gorm.DB, timestampLte, timestampGte *int) *gorm.DB {
	if timestampGte != nil {
		query = query.Where("timestamp >= ?", *timestampGte)
	}
	if timestampLte != nil {
		query = query.Where("timestamp <= ?", *timestampLte)
	}
	return query
}

func sortRecordsByDirection(query *gorm.DB, sort string) (*gorm.DB, error) {
	switch sort {
	case "+index":
		query = query.Order("pulse_number asc").Order("\"order\" asc")
	case "-index":
		query = query.Order("pulse_number desc").Order("\"order\" desc")
	default:
		return nil, errors.New("query parameter 'sort' should be '-index'' or '+index'")
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

func getPulses(query *gorm.DB, limit, offset int) ([]models.Pulse, int, error) {
	pulses := []models.Pulse{}
	var total int
	err := query.Limit(limit).Offset(offset).Find(&pulses).Error
	if err != nil {
		return nil, 0, err
	}
	err = query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return pulses, total, nil
}

// GetLifeline returns records for provided object reference, ordered by pulse number and order fields.
func (s *storage) GetLifeline(objRef []byte, fromIndex *string, pulseNumberLt, pulseNumberGt, timestampLte, timestampGte *int, limit, offset int, sort string) ([]models.Record, int, error) {
	query := s.db.Model(&models.Record{}).Where("object_reference = ?", objRef).Where("type = ?", models.State)

	query = filterByPulse(query, pulseNumberLt, pulseNumberGt)

	query = filterByTimestamp(query, timestampLte, timestampGte)

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

// GetPulses returns pulses from db.
func (s *storage) GetPulses(fromPulse *int64, timestampLte, timestampGte *int, limit, offset int) ([]models.Pulse, int, error) {
	query := s.db.Model(&models.Pulse{})

	query = filterByTimestamp(query, timestampLte, timestampGte)

	var err error
	if fromPulse != nil {
		query = query.Where("pulse_number <= ?", &fromPulse)
	}

	query = query.Order("pulse_number desc")

	pulses, total, err := getPulses(query, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, "error while select pulses from db")
	}

	// set real NextPulseNumber to every pulse (if we know it)
	for i := 0; i < len(pulses)-1; i++ {
		if pulses[i].PrevPulseNumber == pulses[i+1].PulseNumber {
			pulses[i+1].NextPulseNumber = pulses[i].PulseNumber
		}
		if i == 0 {
			pulses[i] = s.updateNextPulse(pulses[i])
		}
	}

	return pulses, total, err
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

// GetRecordsByJetDrop returns records for provided jet drop, ordered by order field.
func (s *storage) GetRecordsByJetDrop(jetDropID models.JetDropID, fromIndex, recordType *string, limit, offset int) ([]models.Record, int, error) {
	query := s.db.Model(&models.Record{}).Where("pulse_number = ?", jetDropID.PulseNumber).Where("jet_id = ?", jetDropID.JetID)

	if recordType != nil {
		query = query.Where("type = ?", *recordType)
	}

	var err error
	if fromIndex != nil {
		query, err = filterRecordsByIndex(query, *fromIndex, "+index")
		if err != nil {
			return nil, 0, err
		}
	}

	query, err = sortRecordsByDirection(query, "+index")
	if err != nil {
		return nil, 0, err
	}

	records, total, err := getRecords(query, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "error while select records for pulse %v, jet %v from db", jetDropID.PulseNumber, jetDropID.JetID)
	}
	return records, total, nil
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

func (s *storage) GetJetDropsWithParams(pulse models.Pulse, fromJetDropID *models.JetDropID, limit int, offset int) ([]models.JetDrop, int, error) {
	var jetDrops []models.JetDrop
	q := s.db.Model(&jetDrops).Where("pulse_number = ?", pulse.PulseNumber).Order("jet_id asc")
	if fromJetDropID != nil {
		q = q.Where("jet_id >= ?", fromJetDropID.JetID)
	}
	err := q.Limit(limit).Offset(offset).Find(&jetDrops).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return jetDrops, int(total), err
}

func (s *storage) GetJetDropByID(id models.JetDropID) (models.JetDrop, error) {
	var jetDrop models.JetDrop
	err := s.db.Model(&jetDrop).Where("pulse_number = ? AND jet_id = ?", id.PulseNumber, id.JetID).Find(&jetDrop).Error
	return jetDrop, err
}

func (s *storage) GetJetDropsByJetId(jetID []byte, fromJetDropID *models.JetDropID, jetDropIDGte, jetDropIDLte *models.JetDropID, limit int, offset int, sortByPnAsc bool) ([]models.JetDrop, int, error) {
	var jetDrops []models.JetDrop
	var total int64

	q := s.db.Model(&jetDrops).Where(&models.JetDrop{JetID: jetID})

	if fromJetDropID != nil {
		q = q.Where("jet_id >= ?", fromJetDropID.JetID)
	}

	// s := "+pulse_number,-jet_id"
	// s := "-pulse_number,+jet_id"
	q = filterByJetDropId(q, jetDropIDGte, jetDropIDLte)

	if sortByPnAsc {
		q = q.Order("pulse_number asc").Order(" jet_id desc")
	} else {
		q = q.Order("pulse_number desc").Order("jet_id asc")
	}

	err := q.Limit(limit).Offset(offset).Find(&jetDrops).Error
	if err == sql.ErrNoRows {
		return jetDrops, 0, nil
	}

	if err != nil {
		return nil, 0, err
	}
	err = q.Count(&total).Error

	if err == sql.ErrNoRows {
		return jetDrops, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}

	return jetDrops, int(total), nil
}
