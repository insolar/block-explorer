// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/block-explorer/etl/models"
)

type Storage struct {
	db *gorm.DB
}

// NewStorage returns implementation of interfaces.Storage
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{
		db: db,
	}
}

// SaveJetDropData saves provided jetDrop and records to db in one transaction.
// increase jet_drop_amount and record_amount
func (s *Storage) SaveJetDropData(jetDrop models.JetDrop, records []models.Record, pulseNumber int64) error {
	timer := prometheus.NewTimer(SaveJetDropDataDuration)
	defer timer.ObserveDuration()

	err := s.initJD(jetDrop, records, pulseNumber)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "error while saving jetDrop") {
		return s.updateJD(jetDrop, records)
	}
	return err
}

func (s *Storage) initJD(jetDrop models.JetDrop, records []models.Record, pulseNumber int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		jd := &jetDrop
		transaction := tx.New()
		if err := transaction.LogMode(false).Create(jd).Error; err != nil {
			return errors.Wrap(err, "error while saving jetDrop")
		}

		for _, record := range records {
			if err := tx.Save(&record).Error; err != nil { // nolint
				return errors.Wrap(err, "error while saving record")
			}
		}

		err := tx.Model(&models.Pulse{PulseNumber: pulseNumber}).
			UpdateColumns(map[string]interface{}{
				"jet_drop_amount": gorm.Expr("jet_drop_amount + ?", 1),
				"record_amount":   gorm.Expr("record_amount + ?", len(records)),
			}).Error
		if err != nil {
			return errors.Wrap(err, "error to update pulse data")
		}
		return nil
	})
}

func (s *Storage) updateJD(jetDrop models.JetDrop, records []models.Record) error {
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
	timer := prometheus.NewTimer(SavePulseDuration)
	defer timer.ObserveDuration()

	err := s.db.Set("gorm:insert_option", ""+
		"ON CONFLICT (pulse_number) DO UPDATE SET prev_pulse_number=EXCLUDED.prev_pulse_number, "+
		"next_pulse_number=EXCLUDED.next_pulse_number, timestamp=EXCLUDED.timestamp",
	).Create(&pulse).Error
	return errors.Wrap(err, "error while saving pulse")
}

// CompletePulse update pulse with provided number to completeness in db.
func (s *Storage) CompletePulse(pulseNumber int64) error {
	timer := prometheus.NewTimer(CompletePulseDuration)
	defer timer.ObserveDuration()
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
	timer := prometheus.NewTimer(SequencePulseDuration)
	defer timer.ObserveDuration()
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
	timer := prometheus.NewTimer(GetRecordDuration)
	defer timer.ObserveDuration()
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
	timer := prometheus.NewTimer(GetLifelineDuration)
	defer timer.ObserveDuration()

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
func (s *Storage) GetPulse(pulseNumber int64) (models.Pulse, error) {
	timer := prometheus.NewTimer(GetPulseDuration)
	defer timer.ObserveDuration()

	var pulse models.Pulse
	err := s.db.Where("pulse_number = ?", pulseNumber).First(&pulse).Error
	if err != nil {
		return pulse, err
	}

	pulse = s.updateNextPulse(pulse)
	pulse = s.updatePrevPulse(pulse)

	return pulse, err
}

// GetPulses returns pulses from db.
func (s *Storage) GetPulses(fromPulse *int64, timestampLte, timestampGte, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64, sortByAsc bool, limit, offset int) ([]models.Pulse, int, error) {
	timer := prometheus.NewTimer(GetPulsesDuration)
	defer timer.ObserveDuration()

	query := s.db.Model(&models.Pulse{})
	query = filterByTimestamp(query, timestampLte, timestampGte)
	query = filterByPulseNumber(query, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt)
	if sortByAsc {
		query = query.Order("pulse_number asc")
	} else {
		query = query.Order("pulse_number desc")
	}

	var err error
	if fromPulse != nil {
		query = query.Where("pulse_number <= ?", &fromPulse)
	}

	pulses, total, err := getPulses(query, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, "error while select pulses from db")
	}

	// set real NextPulseNumber and PrevPulseNumber to every pulse (if we don't know it, set -1)
	pulsesLen := len(pulses)
	if sortByAsc {
		for i := pulsesLen - 1; i > 0; i-- {
			if pulses[i].PrevPulseNumber == pulses[i-1].PulseNumber {
				pulses[i-1].NextPulseNumber = pulses[i].PulseNumber
			} else {
				pulses[i].PrevPulseNumber = -1
				pulses[i-1].NextPulseNumber = -1
			}
		}

		if pulsesLen > 0 {
			pulses[0] = s.updatePrevPulse(pulses[0])
			pulses[pulsesLen-1] = s.updateNextPulse(pulses[pulsesLen-1])
		}
	} else {
		for i := 0; i < pulsesLen-1; i++ {
			if pulses[i].PrevPulseNumber == pulses[i+1].PulseNumber {
				pulses[i+1].NextPulseNumber = pulses[i].PulseNumber
			} else {
				pulses[i+1].NextPulseNumber = -1
				pulses[i].PrevPulseNumber = -1
			}
		}

		if pulsesLen > 0 {
			pulses[0] = s.updateNextPulse(pulses[0])
			pulses[pulsesLen-1] = s.updatePrevPulse(pulses[pulsesLen-1])
		}
	}
	return pulses, total, err
}

func (s *Storage) updateNextPulse(pulse models.Pulse) models.Pulse {
	var nextPulse models.Pulse
	err := s.db.Where("prev_pulse_number = ?", pulse.PulseNumber).First(&nextPulse).Error
	if err != nil {
		pulse.NextPulseNumber = -1
	} else {
		pulse.NextPulseNumber = nextPulse.PulseNumber
	}

	return pulse
}

func (s *Storage) updatePrevPulse(pulse models.Pulse) models.Pulse {
	var prevPulse models.Pulse
	err := s.db.Where("pulse_number = ?", pulse.PrevPulseNumber).First(&prevPulse).Error
	if err != nil {
		pulse.PrevPulseNumber = -1
	}
	return pulse
}

// GetRecordsByJetDrop returns records for provided jet drop, ordered by order field.
func (s *Storage) GetRecordsByJetDrop(jetDropID models.JetDropID, fromIndex, recordType *string, limit, offset int) ([]models.Record, int, error) {
	timer := prometheus.NewTimer(GetRecordsByJetDropDuration)
	defer timer.ObserveDuration()

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
	timer := prometheus.NewTimer(GetIncompletePulsesDuration)
	defer timer.ObserveDuration()

	var pulses []models.Pulse
	err := s.db.Where("is_complete = ?", false).Find(&pulses).Error
	return pulses, err
}

// GetPulseByPrev returns pulse with provided prev pulse number from db.
func (s *Storage) GetPulseByPrev(prevPulse models.Pulse) (models.Pulse, error) {
	timer := prometheus.NewTimer(GetPulseByPrevDuration)
	defer timer.ObserveDuration()

	var pulse models.Pulse
	err := s.db.Where("prev_pulse_number = ?", prevPulse.PulseNumber).First(&pulse).Error
	return pulse, err
}

// GetSequentialPulse returns max pulse that have is_sequential as true from db.
func (s *Storage) GetSequentialPulse() (models.Pulse, error) {
	timer := prometheus.NewTimer(GetSequentialPulseDuration)
	defer timer.ObserveDuration()

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
	timer := prometheus.NewTimer(GetNextSavedPulseDuration)
	defer timer.ObserveDuration()

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
	timer := prometheus.NewTimer(GetJetDropsDuration)
	defer timer.ObserveDuration()

	var jetDrops []models.JetDrop
	err := s.db.Where("pulse_number = ?", pulse.PulseNumber).Find(&jetDrops).Error
	return jetDrops, err
}

func (s *Storage) GetJetDropsWithParams(pulse models.Pulse, fromJetDropID *models.JetDropID, limit int, offset int) ([]models.JetDrop, int, error) {
	timer := prometheus.NewTimer(GetJetDropsWithParamsDuration)
	defer timer.ObserveDuration()

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

func (s *Storage) GetJetDropByID(id models.JetDropID) (models.JetDrop, []models.JetDrop, []models.JetDrop, error) {
	timer := prometheus.NewTimer(GetJetDropByIDDuration)
	defer timer.ObserveDuration()

	var jetDrop models.JetDrop
	err := s.db.Model(&jetDrop).Where("pulse_number = ? AND jet_id = ?", id.PulseNumber, id.JetID).Find(&jetDrop).Error
	if err != nil {
		return jetDrop, nil, nil, err
	}
	// get prev and next pulse
	var pulse models.Pulse
	err = s.db.Where("pulse_number = ?", id.PulseNumber).First(&pulse).Error
	if err != nil {
		return jetDrop, nil, nil, err
	}
	pulse = s.updateNextPulse(pulse)

	siblings := jetDrop.Siblings()

	var nextJetDrops []models.JetDrop
	// If NextPulseNumber == -1 after call to updateNextPulse, it doesn't exist in db
	if pulse.NextPulseNumber != -1 {
		err = s.db.Model(&nextJetDrops).Where("pulse_number = ? AND jet_id in (?)", pulse.NextPulseNumber, siblings).Find(&nextJetDrops).Error
		if err != nil {
			return jetDrop, nil, nil, err
		}
	}

	var prevJetDrops []models.JetDrop
	err = s.db.Model(&prevJetDrops).Where("pulse_number = ? AND jet_id in (?)", pulse.PrevPulseNumber, siblings).Find(&prevJetDrops).Error
	if err != nil {
		return jetDrop, nil, nil, err
	}

	return jetDrop, prevJetDrops, nextJetDrops, err
}

func (s *Storage) GetJetDropsByJetID(jetID string, pulseNumberLte, pulseNumberLt, pulseNumberGte, pulseNumberGt *int64, limit int, sortByPnAsc bool) ([]models.JetDrop, int, error) {
	timer := prometheus.NewTimer(GetJetDropsByJetIDDuration)
	defer timer.ObserveDuration()

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
