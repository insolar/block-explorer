// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package heavymock

import (
	"time"

	"github.com/insolar/insolar/ledger/heavy/exporter"
)

const (
	Timeout = 10 * time.Microsecond
)

type RecordExporter struct {
}

func NewRecordExporter() *RecordExporter {
	return &RecordExporter{}
}

func (r *RecordExporter) Export(records *exporter.GetRecords, stream exporter.RecordExporter_ExportServer) error {
	count := int(records.Count)
	pulse := records.PulseNumber

	if records.PulseNumber == 0 {
		for i := 0; i < count; i++ {
			time.Sleep(Timeout)
			if err := stream.Send(SimpleRecord); err != nil {
				return err
			}
		}
	} else {
		records := GetRecordsByPulse(pulse, count)
		for _, r := range records {
			time.Sleep(Timeout)
			if err := stream.Send(&r); err != nil {
				return err
			}
		}
	}
	return nil
}
