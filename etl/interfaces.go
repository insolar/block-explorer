// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package etl

import (
	"context"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"google.golang.org/grpc"
)

type Starter interface {
	// Start starts the main thread
	Start(ctx context.Context) error
}

type Stopper interface {
	// Stops stops the main thread
	Stop(ctx context.Context) error
}

// JetDropsExtractor represents the main functions of working with Platform
type JetDropsExtractor interface {
	// GetRecords stores Record data in the main Record channel
	GetRecords() (<-chan exporter.Record, error)
}

// ConnectionManager represents management of connection to Platform
type ConnectionManager interface {
	Starter
	Stopper
}

// Transformer represents a transformation raw data from the Platform to conan type
type Transformer interface {
	Starter
	Stopper
	// transform transforms the row data to canonical data
	transform(drop PlatformJetDrops) JetDrop
	// GetJetDropsChannel returns the channel where canonical data will be stored
	GetJetDropsChannel() <-chan JetDrop
}

// Client represents a connection to the Platform
type Client interface {
	// GetGRPCConn returns a configured GRPC connection
	GetGRPCConn() (*grpc.ClientConn, error)
}

// Processor saves canonical data to database
type Processor interface {
	Starter
	Stopper
	process(drop JetDrop)
}

// Controller tracks drops integrity and makes calls to reload data
type Controller interface {
	Starter
	Stopper
	// Save information about saved jetdrops
	SetJetDropData(pulse Pulse, jetID []byte)
}
