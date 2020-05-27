// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package testutils

import (
	"github.com/insolar/insolar/insolar/gen"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/models"
)

var pulseDelta = uint16(10)

// InitRecordDB returns generated record
func InitRecordDB(jetDrop models.JetDrop) models.Record {
	return models.Record{
		Reference:           gen.Reference().Bytes(),
		Type:                "",
		ObjectReference:     gen.Reference().Bytes(),
		PrototypeReference:  gen.Reference().Bytes(),
		Payload:             GenerateRandBytes(),
		PrevRecordReference: gen.Reference().Bytes(),
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
		JetID:          GenerateUniqueJetID().Prefix(),
		PulseNumber:    pulse.PulseNumber,
		FirstPrevHash:  GenerateRandBytes(),
		SecondPrevHash: GenerateRandBytes(),
		Hash:           GenerateRandBytes(),
		RawData:        GenerateRandBytes(),
		Timestamp:      pulse.Timestamp,
	}
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

// CreatePulse creates provided jet drop at db
func CreateJetDrop(db *gorm.DB, jetDrop models.JetDrop) error {
	if err := db.Create(&jetDrop).Error; err != nil {
		return errors.Wrap(err, "error while saving jetDrop")
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
