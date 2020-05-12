// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package etl

import (
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/tsovak/awesomeProject/etl"
	"google.golang.org/grpc"
)

// JetDropsExtractor represents the main functions of working with Platform
type JetDropsExtractor interface {
	// GetRecords stores Record data in the main Record channel
	GetRecords() (<-chan exporter.Record, error)
}

// ConnectionManager represents management of connection to Platform
type ConnectionManager interface {
	// Start starts the main thread
	Start()
	// Stops stops the main thread
	Stop()
}

// Transformer represents a transformation raw data from the Platform to conan type
type Transformer interface {
	// // transforms the row data to canonical data
	// transform(insolar.JetDrop) JetDrop

	// Start starts the main thread
	Start() error
	// Stop stops the main thread
	Stop() error

	GetJetDropsChannel() <-chan etl.JetDrop
}

// Client represents a connection to the Platform
type Client interface {
	// GetGRPCConn returns a configured GRPC connection
	GetGRPCConn() (*grpc.ClientConn, error)
}
