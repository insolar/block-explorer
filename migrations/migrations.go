// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package migrations

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"

	"github.com/insolar/block-explorer/etl/models"
)

func Migrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "202005180423",
			Migrate: func(tx *gorm.DB) error {
				type Pulse struct {
					PulseNumber     int `gorm:"primary_key;auto_increment:false"`
					PrevPulseNumber int
					NextPulseNumber int
					IsComplete      bool
					Timestamp       time.Time
				}
				if err := tx.CreateTable(&Pulse{}).Error; err != nil {
					return err
				}
				if err := tx.Model(Pulse{}).AddIndex("idx_prevpulsenumber", "prev_pulse_number").Error; err != nil {
					return err
				}

				type JetDrop struct {
					JetID          []byte `gorm:"primary_key;auto_increment:false"`
					PulseNumber    int    `gorm:"primary_key;auto_increment:false"`
					FirstPrevHash  []byte
					SecondPrevHash []byte
					Hash           []byte
					RawData        []byte
					Timestamp      time.Time
				}
				if err := tx.CreateTable(&JetDrop{}).Error; err != nil {
					return err
				}
				if err := tx.Model(JetDrop{}).AddIndex("idx_pulsenumber_jetid", "pulse_number", "jet_id").Error; err != nil {
					return err
				}
				if err := tx.Model(&JetDrop{}).AddForeignKey("pulse_number", "pulses(pulse_number)", "CASCADE", "CASCADE").Error; err != nil {
					return err
				}

				type Record struct {
					Reference           models.Reference `gorm:"primary_key;auto_increment:false"`
					Type                models.RecordType
					ObjectReference     models.Reference
					PrototypeReference  models.Reference
					Payload             []byte
					PrevRecordReference models.Reference
					Hash                []byte
					RawData             []byte
					JetID               []byte
					PulseNumber         int
					Order               int
					Timestamp           time.Time
				}
				fmt.Println("now c")
				if err := tx.CreateTable(&Record{}).Error; err != nil {
					return err
				}
				fmt.Println("all was well")
				if err := tx.Model(Record{}).AddIndex(
					"idx_objectreference_pulsenumber_order", "object_reference", "pulse_number", "order").Error; err != nil {
					return err
				}
				if err := tx.Model(Record{}).AddIndex(
					"idx_jetid_pulsenumber_order", "jet_id", "pulse_number", "order").Error; err != nil {
					return err
				}
				if err := tx.Model(&Record{}).AddForeignKey("jet_id, pulse_number", "jet_drops(jet_id, pulse_number)", "CASCADE", "CASCADE").Error; err != nil {
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
