// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"testing"

	"github.com/insolar/block-explorer/instrumentation/converter"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/etl/models"
)

var pulseDelta = uint16(10)

// InitRecordDB returns generated record
func InitRecordDB(jetDrop models.JetDrop) models.Record {
	return models.Record{
		Reference:           gen.ID().Bytes(),
		Type:                models.State,
		ObjectReference:     gen.ID().Bytes(),
		PrototypeReference:  gen.ID().Bytes(),
		Payload:             GenerateRandBytes(),
		PrevRecordReference: gen.ID().Bytes(),
		Hash:                GenerateRandBytes(),
		RawData:             GenerateRandBytes(),
		JetID:               jetDrop.JetID,
		PulseNumber:         jetDrop.PulseNumber,
		Order:               1,
		Timestamp:           jetDrop.Timestamp,
	}
}

// InitJetDropDB returns generated jet drop with provided pulse
func InitJetDropDB(pulse models.Pulse) models.JetDrop {
	return models.JetDrop{
		JetID:          converter.JetIDToString(GenerateUniqueJetID()),
		PulseNumber:    pulse.PulseNumber,
		FirstPrevHash:  GenerateRandBytes(),
		SecondPrevHash: GenerateRandBytes(),
		Hash:           GenerateRandBytes(),
		RawData:        GenerateRandBytes(),
		Timestamp:      pulse.Timestamp,
	}
}

// GenerateJetDropsWithSomeJetID returns a list of JetDrops with some JetID and ascending pulseNumber
func GenerateJetDropsWithSomeJetID(t *testing.T, jCount int) (string, []models.JetDrop, []models.Pulse) {
	pulses := make([]models.Pulse, jCount)
	pulse, err := InitPulseDB()
	require.NoError(t, err)
	pulses[0] = pulse

	drops := make([]models.JetDrop, jCount)
	jDrop := InitJetDropDB(pulse)
	drops[0] = jDrop
	jID := &jDrop.JetID

	pn := pulse.PulseNumber
	for i := 1; i < jCount; i++ {
		pulse, err := InitNextPulseDB(pn)
		require.NoError(t, err)
		pulses[i] = pulse
		jd := InitJetDropDB(pulse)
		jd.JetID = *jID
		drops[i] = jd
		pn = pulse.PulseNumber
	}
	return *jID, drops, pulses
}

// InitPulseDB returns generated pulse
func InitPulseDB() (models.Pulse, error) {
	pulseNumber := gen.PulseNumber()
	timestamp, err := pulseNumber.AsApproximateTime()
	if err != nil {
		return models.Pulse{}, err
	}
	return models.Pulse{
		PulseNumber:     int(pulseNumber.AsUint32()),
		PrevPulseNumber: int(pulseNumber.Prev(pulseDelta)),
		NextPulseNumber: int(pulseNumber.Next(pulseDelta)),
		IsComplete:      false,
		Timestamp:       timestamp.Unix(),
	}, nil
}

// InitNextPulseDB returns generated pulse after pn
func InitNextPulseDB(pn int) (models.Pulse, error) {
	pulseNumber := insolar.PulseNumber(pn + int(pulseDelta))
	timestamp, err := pulseNumber.AsApproximateTime()
	if err != nil {
		return models.Pulse{}, err
	}
	return models.Pulse{
		PulseNumber:     int(pulseNumber.AsUint32()),
		PrevPulseNumber: int(pulseNumber.Prev(pulseDelta)),
		NextPulseNumber: int(pulseNumber.Next(pulseDelta)),
		IsComplete:      false,
		Timestamp:       timestamp.Unix(),
	}, nil
}

// CreateRecord creates provided record at db
func CreateRecord(db *gorm.DB, record models.Record) error {
	if err := db.Create(&record).Error; err != nil {
		return errors.Wrap(err, "error while saving record")
	}
	return nil
}

// CreatePulse creates provided jet drop at db
func CreateJetDrop(db *gorm.DB, jetDrop models.JetDrop) error {
	if err := db.Create(&jetDrop).Error; err != nil {
		return errors.Wrap(err, "error while saving jetDrop")
	}
	return nil
}

// CreateJetDrops creates provided jet drop list to db
func CreateJetDrops(db *gorm.DB, jetDrops []models.JetDrop) error {
	for _, v := range jetDrops {
		err := CreateJetDrop(db, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreatePulse creates provided pulse at db
func CreatePulse(db *gorm.DB, pulse models.Pulse) error {
	if err := db.Create(&pulse).Error; err != nil {
		return errors.Wrap(err, "error while saving pulse")
	}
	return nil
}

// CreatePulses creates provided pulses to db
func CreatePulses(db *gorm.DB, pulses []models.Pulse) error {
	for _, p := range pulses {
		err := CreatePulse(db, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func OrderedRecords(t *testing.T, db *gorm.DB, jetDrop models.JetDrop, objRef insolar.ID, amount int) []models.Record {
	var result []models.Record
	for i := 1; i <= amount; i++ {
		record := InitRecordDB(jetDrop)
		record.ObjectReference = objRef.Bytes()
		record.Order = i
		err := CreateRecord(db, record)
		require.NoError(t, err)
		result = append(result, record)
	}
	return result
}
