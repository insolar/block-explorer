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
	// timeout to wait between sending records to the stream
	recordSendingIntervalTimeout = 10 * time.Microsecond
	// this constant is used to be set in the field exporter.GetRecords.Polymorph when
	// sending request to heavymock. Response stream must contain saved records in heavymock, which
	// were previously imported using heavymock.ImporterClient.
	MagicPolymorphExport = 101010
)

type RecordExporter struct {
	importerServer *ImporterServer
}

func NewRecordExporter(importerServer *ImporterServer) *RecordExporter {
	return &RecordExporter{
		importerServer,
	}
}

func (r *RecordExporter) Export(records *exporter.GetRecords, stream exporter.RecordExporter_ExportServer) error {
	// count := int(records.Count)
	// pulse := records.PulseNumber
	//
	// if records.PulseNumber == 0 {
	// 	for i := 0; i < count; i++ {
	// 		time.Sleep(recordSendingIntervalTimeout)
	// 		if err := stream.Send(SimpleRecord); err != nil {
	// 			return err
	// 		}
	// 	}
	// } else {
	// 	records := GetRecordsByPulse(pulse, count)
	// 	for _, r := range records {
	// 		time.Sleep(recordSendingIntervalTimeout)
	// 		if err := stream.Send(&r); err != nil {
	// 			return err
	// 		}
	// 	}
	// }
	//
	// if records.Polymorph == MagicPolymorphExport {
	savedRecords := r.importerServer.GetSavedRecords()
	for _, r := range savedRecords {
		time.Sleep(recordSendingIntervalTimeout)
		if err := stream.Send(&r); err != nil {
			return err
		}
	}
	// }
	return nil
}
