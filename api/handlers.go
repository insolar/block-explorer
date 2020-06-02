// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/instrumentation/belogger"
)

type Server struct {
	storage interfaces.StorageFetcher
	logger  log.Logger
	config  configuration.API
}

// NewServer returns instance of API server
func NewServer(ctx context.Context, storage interfaces.StorageFetcher, config configuration.API) *Server {
	logger := belogger.FromContext(ctx)
	return &Server{storage: storage, logger: logger, config: config}
}
