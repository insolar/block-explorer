// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/models"
)

type Storage struct {
	db            *gorm.DB
	savePulseLock sync.Mutex
}

// NewStorage returns implementation of interfaces.Storage
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{
		db: db,
	}
}

// SaveJetDropData saves provided jetDrop and records to db in one transaction.
func (s *Storage) SaveJetDropData(jetDrop models.JetDrop, records []models.Record) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		jd := &jetDrop
		if err := tx.Save(jd).Error; err != nil {
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
func (s *Storage) SavePulse(pulse models.Pulse) error {
	s.savePulseLock.Lock()
	defer s.savePulseLock.Unlock()
	return errors.Wrap(s.db.Save(&pulse).Error, "error while saving pulse")
}

// CompletePulse update pulse with provided number to completeness in db.
func (s *Storage) CompletePulse(pulseNumber int64) error {
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
			return errors.Errorf("several rows were affected by update for pulse with number %d to complete, it was not expected", pulseNumber)
		}
		return nil
	})
}

// SequencePulse update pulse with provided number to sequential in db.
func (s *Storage) SequencePulse(pulseNumber int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		pulse := models.Pulse{PulseNumber: pulseNumber}
		update := tx.Model(&pulse).Update(models.Pulse{IsSequential: true})
		if update.Error != nil {
			return errors.Wrap(update.Error, "error while updating pulse to sequential")
		}
		rowsAffected := update.RowsAffected
		if rowsAffected == 0 {
			return errors.Errorf("try to sequence not existing pulse with number %d", pulseNumber)
		}
		if rowsAffected != 1 {
			return errors.Errorf("several rows were affected by update for pulse with number %d to sequential, it was not expected", pulseNumber)
		}
		return nil
	})
}

// GetJetDrops returns records with provided reference from db.
func (s *Storage) GetRecord(ref models.Reference) (models.Record, error) {
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

func filterByPulse(query *gorm.DB, pulseNumberLt, pulseNumberGt *int64) *gorm.DB {
	if pulseNumberGt != nil {
		query = query.Where("pulse_number > ?", *pulseNumberGt)
	}
	if pulseNumberLt != nil {
		query = query.Where("pulse_number < ?", *pulseNumberLt)
	}
	return query
}

func filterByPulseNumber(query *gorm.DB, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64) *gorm.DB {
	if pulseNumberLte != nil {
		query = query.Where("pulse_number <= ?", *pulseNumberLte)
	}

	if pulseNumberLt != nil {
		query = query.Where("pulse_number < ?", *pulseNumberLt)
	}

	if pulseNumberGte != nil {
		query = query.Where("pulse_number >= ?", *pulseNumberGte)
	}

	if pulseNumberGt != nil {
		query = query.Where("pulse_number > ?", *pulseNumberGt)
	}

	return query
}

func filterRecordsByIndex(query *gorm.DB, fromIndex string, sortByIndexAsc bool) (*gorm.DB, error) {
	pulseNumber, order, err := CheckIndex(fromIndex)
	if err != nil {
		return nil, err
	}
	if sortByIndexAsc {
		query = query.Where("(pulse_number > ?", pulseNumber)
		query = query.Or("pulse_number = ? AND \"order\" >= ?)", pulseNumber, order)
	} else {
		query = query.Where("(pulse_number < ?", pulseNumber)
		query = query.Or("pulse_number = ? AND \"order\" <= ?)", pulseNumber, order)
	}
	return query, nil
}

func filterByTimestamp(query *gorm.DB, timestampLte, timestampGte *int64) *gorm.DB {
	if timestampGte != nil {
		query = query.Where("timestamp >= ?", *timestampGte)
	}
	if timestampLte != nil {
		query = query.Where("timestamp <= ?", *timestampLte)
	}
	return query
}

func sortRecordsByDirection(query *gorm.DB, sortByIndexAsc bool) *gorm.DB {
	if sortByIndexAsc {
		query = query.Order("pulse_number asc").Order("\"order\" asc")
	} else {
		query = query.Order("pulse_number desc").Order("\"order\" desc")
	}
	return query
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
func (s *Storage) GetLifeline(objRef []byte, fromIndex *string, pulseNumberLt, pulseNumberGt, timestampLte, timestampGte *int64, limit, offset int, sortByIndexAsc bool) ([]models.Record, int, error) {
	query := s.db.Model(&models.Record{}).Where("object_reference = ?", objRef).Where("type = ?", models.State)

	query = filterByPulse(query, pulseNumberLt, pulseNumberGt)

	query = filterByTimestamp(query, timestampLte, timestampGte)

	var err error
	if fromIndex != nil {
		query, err = filterRecordsByIndex(query, *fromIndex, sortByIndexAsc)
		if err != nil {
			return nil, 0, err
		}
	}

	query = sortRecordsByDirection(query, sortByIndexAsc)

	records, total, err := getRecords(query, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "error while select records for object %v from db", objRef)
	}
	return records, total, nil
}

// GetPulse returns pulse with provided pulse number from db.
func (s *Storage) GetPulse(pulseNumber int64) (models.Pulse, int64, int64, error) {
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
func (s *Storage) GetAmounts(pulseNumber int64) (int64, int64, error) {
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
func (s *Storage) GetPulses(fromPulse *int64, timestampLte, timestampGte *int64, limit, offset int) ([]models.Pulse, int, error) {
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

func (s *Storage) updateNextPulse(pulse models.Pulse) models.Pulse {
	var nextPulse models.Pulse
	err := s.db.Where("prev_pulse_number = ?", pulse.PulseNumber).First(&nextPulse).Error
	if err != nil {
		return pulse
	}
	pulse.NextPulseNumber = nextPulse.PulseNumber
	return pulse
}

// GetRecordsByJetDrop returns records for provided jet drop, ordered by order field.
func (s *Storage) GetRecordsByJetDrop(jetDropID models.JetDropID, fromIndex, recordType *string, limit, offset int) ([]models.Record, int, error) {
	query := s.db.Model(&models.Record{}).Where("pulse_number = ?", jetDropID.PulseNumber).Where("jet_id = ?", jetDropID.JetID)

	if recordType != nil {
		query = query.Where("type = ?", *recordType)
	}

	var err error
	if fromIndex != nil {
		query, err = filterRecordsByIndex(query, *fromIndex, true)
		if err != nil {
			return nil, 0, err
		}
	}

	query = sortRecordsByDirection(query, true)

	records, total, err := getRecords(query, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "error while select records for pulse %v, jet %v from db", jetDropID.PulseNumber, jetDropID.JetID)
	}
	return records, total, nil
}

// GetIncompletePulses returns pulses that are not complete from db.
func (s *Storage) GetIncompletePulses() ([]models.Pulse, error) {
	var pulses []models.Pulse
	err := s.db.Where("is_complete = ?", false).Find(&pulses).Error
	return pulses, err
}

// GetPulseByPrev returns pulse with provided prev pulse number from db.
func (s *Storage) GetPulseByPrev(prevPulse models.Pulse) (models.Pulse, error) {
	var pulse models.Pulse
	err := s.db.Where("prev_pulse_number = ?", prevPulse.PulseNumber).First(&pulse).Error
	return pulse, err
}

// GetSequentialPulse returns max pulse that have is_sequential as true from db.
func (s *Storage) GetSequentialPulse() (models.Pulse, error) {
	var pulses []models.Pulse
	err := s.db.Where("is_sequential = ?", true).Order("pulse_number desc").Limit(1).Find(&pulses).Error
	if err != nil {
		return models.Pulse{}, err
	}
	if len(pulses) == 0 {
		return models.Pulse{}, nil
	}
	return pulses[0], err
}

// GetNextSavedPulse returns first pulse with pulse number bigger then fromPulseNumber from db.
func (s *Storage) GetNextSavedPulse(fromPulseNumber models.Pulse) (models.Pulse, error) {
	var pulses []models.Pulse
	err := s.db.Where("pulse_number > ?", fromPulseNumber.PulseNumber).Order("pulse_number asc").Limit(1).Find(&pulses).Error
	if err != nil {
		return models.Pulse{}, err
	}
	if len(pulses) == 0 {
		return models.Pulse{}, nil
	}
	return pulses[0], err
}

// GetJetDrops returns jetDrops for provided pulse from db.
func (s *Storage) GetJetDrops(pulse models.Pulse) ([]models.JetDrop, error) {
	var jetDrops []models.JetDrop
	err := s.db.Where("pulse_number = ?", pulse.PulseNumber).Find(&jetDrops).Error
	return jetDrops, err
}

func (s *Storage) GetJetDropsWithParams(pulse models.Pulse, fromJetDropID *models.JetDropID, limit int, offset int) ([]models.JetDrop, int, error) {
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

func (s *Storage) GetJetDropByID(id models.JetDropID) (models.JetDrop, error) {
	var jetDrop models.JetDrop
	err := s.db.Model(&jetDrop).Where("pulse_number = ? AND jet_id = ?", id.PulseNumber, id.JetID).Find(&jetDrop).Error
	return jetDrop, err
}

func (s *Storage) GetJetDropsByJetID(jetID string, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64, limit int, sortByPnAsc bool) ([]models.JetDrop, int, error) {
	var jetDrops []models.JetDrop
	var total int64

	q := s.db.Model(&jetDrops).Where("jet_id in (?) or jet_id like ?", GetJetIDParents(jetID), fmt.Sprintf("%s%%", jetID))

	q = filterByPulseNumber(q, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt)

	if sortByPnAsc {
		q = q.Order("pulse_number asc").Order(" jet_id desc")
	} else {
		q = q.Order("pulse_number desc").Order("jet_id asc")
	}

	err := q.Limit(limit).Find(&jetDrops).Error
	if err != nil {
		return nil, 0, err
	}

	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	return jetDrops, int(total), nil
}
