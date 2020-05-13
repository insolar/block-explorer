// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package etl

import (
	"github.com/insolar/insolar/ledger/heavy/exporter"
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
	// transform transforms the row data to canonical data
	transform(drop PlatformJetDrops) JetDrop
	// Start starts the main thread
	Start() error
	// Stop stops the main thread
	Stop() error
	// GetJetDropsChannel returns the channel where canonical data will be stored
	GetJetDropsChannel() <-chan JetDrop
}

// Client represents a connection to the Platform
type Client interface {
	// GetGRPCConn returns a configured GRPC connection
	GetGRPCConn() (*grpc.ClientConn, error)
}
