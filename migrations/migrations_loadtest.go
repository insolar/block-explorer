// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package migrations

import (
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"

	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/instrumentation/converter"
)

// generateRandBytesLen generates random bytes array with len
func generateRandBytesLen(l int) []byte {
	b := make([]byte, l)
	rand.Read(b)
	return b
}

func createTables(tx *gorm.DB) error {
	if err := tx.CreateTable(&models.Pulse{}).Error; err != nil {
		return err
	}
	if err := tx.Model(models.Pulse{}).AddIndex("idx_pulse_prevpulsenumber", "prev_pulse_number").Error; err != nil {
		return err
	}

	if err := tx.CreateTable(&models.JetDrop{}).Error; err != nil {
		return err
	}
	if err := tx.Model(models.JetDrop{}).AddIndex("idx_jetdrop_pulsenumber_jetid", "pulse_number", "jet_id").Error; err != nil {
		return err
	}
	if err := tx.Model(&models.JetDrop{}).AddForeignKey("pulse_number", "pulses(pulse_number)", "CASCADE", "CASCADE").Error; err != nil {
		return err
	}

	if err := tx.CreateTable(&models.Record{}).Error; err != nil {
		return err
	}
	if err := tx.Model(models.Record{}).AddIndex(
		"idx_record_objectreference_type_pulsenumber_order", "object_reference", "type", "pulse_number", "order").Error; err != nil {
		return err
	}
	if err := tx.Model(models.Record{}).AddIndex(
		"idx_record_jetid_pulsenumber_order", "jet_id", "pulse_number", "order").Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Record{}).AddForeignKey("jet_id, pulse_number", "jet_drops(jet_id, pulse_number)", "CASCADE", "CASCADE").Error; err != nil {
		return err
	}
	return nil
}

func generatePulses(amount int) []models.Pulse {
	tNow := time.Now().Unix()
	var pulses []models.Pulse
	for i := 1; i < amount; i++ {
		pulses = append(pulses,
			models.Pulse{
				PulseNumber:     i,
				PrevPulseNumber: i - 1,
				NextPulseNumber: i + 1,
				IsComplete:      true,
				IsSequential:    true,
				Timestamp:       tNow + int64(i*10),
			})
	}
	return pulses
}

func notNullJetID() string {
	for {
		jetID := gen.JetID()
		id := binary.BigEndian.Uint64(jetID.Prefix())
		if id == 0 {
			continue
		}
		return converter.JetIDToString(jetID)
	}
}

func generateJetDrops(pulses []models.Pulse, amount int) []models.JetDrop {
	tNow := time.Now().Unix()
	var jDrops []models.JetDrop
	// uniqJetIDs := gen.UniqueJetIDs(amount)
	for i := 1; i < amount; i++ {
		rHash := generateRandBytesLen(16)
		rawData := generateRandBytesLen(32)
		randPulseNum := rand.Intn(len(pulses))
		rPnum := pulses[randPulseNum].PulseNumber
		// randJetID := rand.Intn(len(uniqJetIDs))
		// jID := converter.JetIDToString(uniqJetIDs[randJetID])
		jID := notNullJetID()
		jDrops = append(jDrops, models.JetDrop{
			JetID:          jID,
			PulseNumber:    rPnum,
			FirstPrevHash:  rHash,
			SecondPrevHash: rHash,
			Hash:           rHash,
			RawData:        rawData,
			Timestamp:      tNow + int64(i*2),
			RecordAmount:   100,
		})
	}
	return jDrops
}

func generateRecords(jDrops []models.JetDrop, amount int) []models.Record {
	tNow := time.Now().Unix()
	var records []models.Record
	for i := 1; i < amount; i++ {
		ref := generateRandBytesLen(16)
		rawData := generateRandBytesLen(32)
		randJetID := rand.Intn(len(jDrops))
		randJet := jDrops[randJetID].JetID
		jetPulseNum := jDrops[randJetID].PulseNumber
		records = append(records, models.Record{
			Reference:           ref,
			Type:                "",
			ObjectReference:     ref,
			PrototypeReference:  ref,
			Payload:             ref,
			PrevRecordReference: ref,
			Hash:                ref,
			RawData:             rawData,
			JetID:               randJet,
			PulseNumber:         jetPulseNum,
			Order:               0,
			Timestamp:           tNow + int64(i*2),
		})
	}
	return records
}

func generateData(tx *gorm.DB) error {
	pulses := generatePulses(101)
	for _, p := range pulses {
		if err := tx.Model(&p).Save(&p).Error; err != nil {
			return err
		}
	}
	jdrops := generateJetDrops(pulses, 1001)
	for _, jd := range jdrops {
		if err := tx.Model(&jd).Save(&jd).Error; err != nil {
			return err
		}
	}
	for _, rec := range generateRecords(jdrops, 10001) {
		if err := tx.Model(&rec).Save(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

func LoadTestMigrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "202005180425",
			Migrate: func(tx *gorm.DB) error {
				if err := createTables(tx); err != nil {
					return err
				}
				if err := generateData(tx); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTableIfExists("records", "jet_drops", "pulses").Error
			},
		},
	}
}
