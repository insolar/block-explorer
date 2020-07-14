// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package migrations

import (
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/insolar/insolar/insolar"
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

func generatePulses(amount int) []models.Pulse {
	tNow := time.Now().Unix()
	startPulse := 4000000
	var pulses []models.Pulse
	for i := startPulse; i < startPulse+amount; i++ {
		pulses = append(pulses,
			models.Pulse{
				PulseNumber:     int64(i),
				PrevPulseNumber: int64(i) - 1,
				NextPulseNumber: int64(i) + 1,
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
	for i := 1; i < amount; i++ {
		rawData := generateRandBytesLen(32)
		randPulseNum := rand.Intn(len(pulses))
		rPnum := pulses[randPulseNum].PulseNumber
		pn := insolar.PulseNumber(rPnum)
		jID := notNullJetID()
		jDrops = append(jDrops, models.JetDrop{
			JetID:          jID,
			PulseNumber:    rPnum,
			FirstPrevHash:  gen.IDWithPulse(pn).Bytes(),
			SecondPrevHash: gen.IDWithPulse(pn).Bytes(),
			Hash:           rawData,
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
		rawData := generateRandBytesLen(32)
		randJetID := rand.Intn(len(jDrops))
		randJet := jDrops[randJetID].JetID
		jetPulseNum := jDrops[randJetID].PulseNumber
		pn := insolar.PulseNumber(jetPulseNum)
		records = append(records, models.Record{
			Reference:           gen.IDWithPulse(pn).Bytes(),
			Type:                "state",
			ObjectReference:     gen.IDWithPulse(pn).Bytes(),
			PrototypeReference:  gen.IDWithPulse(pn).Bytes(),
			Payload:             rawData,
			PrevRecordReference: gen.IDWithPulse(pn).Bytes(),
			Hash:                rawData,
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
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
	}
	jdrops := generateJetDrops(pulses, 1001)
	for _, jd := range jdrops {
		if err := tx.Save(&jd).Error; err != nil {
			return err
		}
	}
	for _, rec := range generateRecords(jdrops, 1001) {
		if err := tx.Save(&rec).Error; err != nil {
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
