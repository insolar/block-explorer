package heavymock

import (
	"sync"
	"time"

	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

const (
	// timeout to wait between sending records to the stream
	recordSendingIntervalTimeout = 10 * time.Microsecond
)

type RecordExporter struct {
	importerServer *ImporterServer
	mux            sync.Mutex
}

func NewRecordExporter(importerServer *ImporterServer) *RecordExporter {
	return &RecordExporter{
		importerServer,
		sync.Mutex{},
	}
}

func (r *RecordExporter) Export(request *exporter.GetRecords, stream exporter.RecordExporter_ExportServer) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	records := []*exporter.Record{}
	for _, r := range r.importerServer.GetUnsentRecords() {
		if r.Record.ID.Pulse() != request.PulseNumber {
			continue
		}
		records = append(records, r)
	}

	r.importerServer.MarkAsSent(records)

	records = append(records, testutils.GenerateRecordInNextPulse(request.PulseNumber)) // marker of pulse's end

	for _, r := range records {
		time.Sleep(recordSendingIntervalTimeout)
		if err := stream.Send(r); err != nil {
			return err
		}
	}
	return nil
}
