// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

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
