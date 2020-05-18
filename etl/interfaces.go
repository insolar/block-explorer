// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package etl

import (
	"context"

	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/models"
)

//go:generate minimock -i github.com/insolar/block-explorer/etl.Starter -o ./mock -s _mock.go -g
type Starter interface {
	// Start starts the main thread
	Start(ctx context.Context) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl.Stopper -o ./mock -s _mock.go -g
type Stopper interface {
	// Stops stops the main thread
	Stop(ctx context.Context) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl.JetDropsExtractor -o ./mock -s _mock.go -g
// JetDropsExtractor represents the main functions of working with Platform
type JetDropsExtractor interface {
	// GetJetDrops stores JetDrop data in the main JetDrop channel
	GetJetDrops(ctx context.Context) <-chan *PlatformJetDrops
}

//go:generate minimock -i github.com/insolar/block-explorer/etl.ConnectionManager -o ./mock -s _mock.go -g
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

//go:generate minimock -i github.com/insolar/block-explorer/etl.Client -o ./mock -s _mock.go -g
// Client represents a connection to the Platform
type Client interface {
	// GetGRPCConn returns a configured GRPC connection
	GetGRPCConn() *grpc.ClientConn
}

// Processor saves canonical data to database
type Processor interface {
	Starter
	Stopper
	process(drop JetDrop)
}

//go:generate minimock -i github.com/insolar/block-explorer/etl.Controller -o ./mock -s _mock.go -g
// Controller tracks drops integrity and makes calls to reload data
type Controller interface {
	Starter
	Stopper
	// Save information about saved jetdrops
	SetJetDropData(pulse Pulse, jetID []byte)
}

// Storage saves data to database
type Storage interface {
	SaveJetDropData(jetDrop models.JetDrop, records []models.Record) error
}
