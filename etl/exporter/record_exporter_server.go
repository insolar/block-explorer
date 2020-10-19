// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type RecordServer struct {
	storage interfaces.StorageExporterFetcher
	logger  log.Logger
	config  configuration.Exporter
}

func NewRecordServer(ctx context.Context, storage interfaces.StorageExporterFetcher, config configuration.Exporter) *RecordServer {
	logger := belogger.FromContext(ctx)
	return &RecordServer{storage: storage, logger: logger, config: config}
}

func (s *RecordServer) GetRecords(request *GetRecordsRequest, stream RecordExporter_GetRecordsServer) error {
	pn := int64(request.PulseNumber)
	recs, err := s.storage.GetRecordsByPrototype(request.Prototypes, pn, request.Count, request.RecordNumber)
	if err != nil {
		return fmt.Errorf("error on requesting records")
	}
	for i, rec := range recs {
		response := GetRecordsResponse{
			RecordNumber:        uint32(i),
			Reference:           rec.Reference,
			Type:                string(rec.Type),
			ObjectReference:     rec.ObjectReference,
			PrototypeReference:  rec.PrototypeReference,
			Payload:             rec.Payload,
			PrevRecordReference: rec.PrevRecordReference,
			PulseNumber:         uint32(rec.PulseNumber),
			Timestamp:           uint32(rec.Timestamp),
		}
		if err := stream.Send(&response); err != nil {
			return err
		}
	}
	return nil
}
