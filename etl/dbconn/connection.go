package dbconn

import (
	"github.com/jinzhu/gorm"
	// import database's driver
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/configuration"
)

// Connect returns connection to database
func Connect(cfg configuration.DB) (*gorm.DB, error) {
	db, err := gorm.Open("postgres", cfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database")
	}
	db.DB().SetMaxOpenConns(cfg.MaxOpenConns)
	db.DB().SetMaxIdleConns(cfg.MaxIdleConns)
	db.DB().SetConnMaxLifetime(cfg.ConnMaxLifetime)
	return db, nil
}
