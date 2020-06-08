// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package connection

import (
	"context"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/instrumentation/belogger"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type GRPCClientConnection struct {
	grpc *grpc.ClientConn
}

// NewGRPCClientConnection returns implementation
func NewGRPCClientConnection(ctx context.Context, cfg configuration.Replicator) (*GRPCClientConnection, error) {
	log := belogger.FromContext(ctx)
	c, e := func() (*grpc.ClientConn, error) {
		options := grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxTransportMsg),
			grpc.MaxCallSendMsgSize(cfg.MaxTransportMsg),
		)
		log.Infof("trying connect to %s...", cfg.Addr)

		// We omit error here because connect happens in background.
		conn, err := grpc.Dial(cfg.Addr, options, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to grpc.Dial")
		}
		return conn, err
	}()

	if e != nil {
		return &GRPCClientConnection{}, e
	}

	return &GRPCClientConnection{c}, nil
}

func (c *GRPCClientConnection) GetGRPCConn() *grpc.ClientConn {
	return c.grpc
}

func GetClientConfiguration(addr string) configuration.Replicator {
	return configuration.Replicator{
		Addr:            addr,
		MaxTransportMsg: 100500,
	}
}
