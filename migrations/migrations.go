package migrations

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"
)

func Migrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "202005180421",
			Migrate: func(tx *gorm.DB) error {
				// the initial database tables. Do not delete it's
				type Pulse struct {
					PulseNumber     int64 `gorm:"primary_key;auto_increment:false"`
					PrevPulseNumber int64
					NextPulseNumber int64
					IsComplete      bool
					IsSequential    bool
					Timestamp       int64
					JetDropAmount   int64
					RecordAmount    int64
				}
				type JetDrop struct {
					PulseNumber    int64  `gorm:"primary_key;auto_increment:false"`
					JetID          string `gorm:"type:varchar(255);primary_key;auto_increment:false;default:''"`
					FirstPrevHash  []byte
					SecondPrevHash []byte
					Hash           []byte
					RawData        []byte
					Timestamp      int64
					RecordAmount   int
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
					JetID               string
					PulseNumber         int64
					Order               int
					Timestamp           int64
				}
				if err := tx.CreateTable(&Pulse{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&Pulse{}).AddIndex("idx_pulse_prevpulsenumber", "prev_pulse_number").Error; err != nil {
					return err
				}

				if err := tx.CreateTable(&JetDrop{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&JetDrop{}).AddIndex("idx_jetdrop_pulsenumber_jetid", "pulse_number", "jet_id").Error; err != nil {
					return err
				}
				if err := tx.Model(&JetDrop{}).AddForeignKey("pulse_number", "pulses(pulse_number)", "CASCADE", "CASCADE").Error; err != nil {
					return err
				}

				if err := tx.CreateTable(&Record{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&Record{}).AddIndex(
					"idx_record_objectreference_type_pulsenumber_order", "object_reference", "type", "pulse_number", "order").Error; err != nil {
					return err
				}
				if err := tx.Model(&Record{}).AddIndex(
					"idx_record_jetid_pulsenumber_order", "jet_id", "pulse_number", "order").Error; err != nil {
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

func LoadTestMigrations(cfg *configuration.TestDB) *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202005180425",
		Migrate: func(tx *gorm.DB) error {
			if err := generateData(tx, cfg); err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTableIfExists("records", "jet_drops", "pulses").Error
		},
	}
}

func MigrationOptions() *gormigrate.Options {
	options := gormigrate.DefaultOptions
	options.UseTransaction = true
	options.ValidateUnknownMigrations = true
	return options
}
