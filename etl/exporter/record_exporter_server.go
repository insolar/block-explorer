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
	states, err := s.storage.GetRecordsByPrototype(request.Prototypes, request.PulseNumber, request.Count, request.RecordNumber)
	if err != nil {
		s.logger.Error(err)
		return fmt.Errorf("error on requesting records %v", err)
	}
	for i, state := range states {
		response := GetRecordsResponse{
			RecordNumber:        uint32(i),
			Reference:           state.RecordReference,
			Type:                string(state.Type),
			ObjectReference:     state.ObjectReference,
			PrototypeReference:  state.ImageReference,
			Payload:             state.Payload,
			PrevRecordReference: state.PrevStateReference,
			PulseNumber:         state.PulseNumber,
			Timestamp:           state.Timestamp,
		}
		if err := stream.Send(&response); err != nil {
			return err
		}
	}
	return nil
}
