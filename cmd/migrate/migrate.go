// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/insolar/insconfig"
	"gopkg.in/gormigrate.v1"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

func main() {
	cfg := &configuration.DB{}
	params := insconfig.Params{
		EnvPrefix:        "migrate",
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	fmt.Println("Starts with configuration:\n", insConfigurator.ToYaml(cfg))

	ctx := context.Background()
	log := belogger.FromContext(ctx)

	db, err := gorm.Open("postgres", cfg.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.LogMode(true)
	db.SetLogger(belogger.NewGORMLogAdapter(log))

	m := gormigrate.New(db, gormigrate.DefaultOptions, migrations())

	if err = m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Info("migrated successfully!")
}

func migrations() []*gormigrate.Migration {
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
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("pulses").Error
			},
		},
		{
			ID: "202005180520",
			Migrate: func(tx *gorm.DB) error {
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
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("jet_drops").Error
			},
		},
		{
			ID: "202005180732",
			Migrate: func(tx *gorm.DB) error {
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
				if err := tx.CreateTable(&Record{}).Error; err != nil {
					return err
				}
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
				return tx.DropTable("records").Error
			},
		},
	}
}
