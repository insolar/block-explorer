// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package etl

// GRPCConfig represents a configuration of the GRPC
type GRPCConfig struct {
	Addr            string
	MaxTransportMsg int
}
