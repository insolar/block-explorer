package belogger

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
)

type GORMLogAdapter struct {
	log log.Logger
}

func NewGORMLogAdapter(log log.Logger) *GORMLogAdapter {
	return &GORMLogAdapter{
		log: log.WithField("service", "database"),
	}
}

func (l *GORMLogAdapter) Print(values ...interface{}) {
	l.log.Info(values)
}
