// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package interfaces

import (
	"context"

	"github.com/insolar/block-explorer/etl/types"
	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/models"
)

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Starter -o ./mock -s _mock.go -g
type Starter interface {
	// Start starts the main thread
	Start(ctx context.Context) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Stopper -o ./mock -s _mock.go -g
type Stopper interface {
	// Stops stops the main thread
	Stop(ctx context.Context) error
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.JetDropsExtractor -o ./mock -s _mock.go -g
// JetDropsExtractor represents the main functions of working with Platform
type JetDropsExtractor interface {
	// GetJetDrops stores JetDrop data in the main JetDrop channel
	GetJetDrops(ctx context.Context) <-chan *types.PlatformJetDrops
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.ConnectionManager -o ./mock -s _mock.go -g
// ConnectionManager represents management of connection to Platform
type ConnectionManager interface {
	Starter
	Stopper
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Transformer -o ./mock -s _mock.go -g
// Transformer represents a transformation raw data from the Platform to conan type
type Transformer interface {
	Starter
	Stopper
	// GetJetDropsChannel returns the channel where canonical data will be stored
	GetJetDropsChannel() <-chan *types.JetDrop
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Client -o ./mock -s _mock.go -g
// Client represents a connection to the Platform
type Client interface {
	// GetGRPCConn returns a configured GRPC connection
	GetGRPCConn() *grpc.ClientConn
}

// Processor saves canonical data to database
type Processor interface {
	Starter
	Stopper
	process(drop types.JetDrop)
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Controller -o ./mock -s _mock.go -g
// Controller tracks drops integrity and makes calls to reload data
type Controller interface {
	Starter
	Stopper
	// Save information about saved jetdrops
	SetJetDropData(pulse types.Pulse, jetID []byte)
}

//go:generate minimock -i github.com/insolar/block-explorer/etl/interfaces.Storage -o ./mock -s _mock.go -g
// storage saves data to database
type Storage interface {
	SaveJetDropData(jetDrop models.JetDrop, records []models.Record) error
}
