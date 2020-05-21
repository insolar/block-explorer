// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package connection

import (
	"context"

	"github.com/insolar/block-explorer/configuration"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type GrpcClientConnection struct {
	grpc *grpc.ClientConn
}

// NewGrpcClientConnection returns implementation
func NewGrpcClientConnection(ctx context.Context, cfg configuration.Replicator) (*GrpcClientConnection, error) {
	c, e := func() (*grpc.ClientConn, error) {
		options := grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxTransportMsg),
			grpc.MaxCallSendMsgSize(cfg.MaxTransportMsg),
		)
		//todo: change to logger
		println("trying connect to %s...", cfg.Addr)

		// We omit error here because connect happens in background.
		conn, err := grpc.Dial(cfg.Addr, options, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to grpc.Dial")
		}
		return conn, err
	}()

	if e != nil {
		return &GrpcClientConnection{}, e
	}

	return &GrpcClientConnection{c}, nil
}

func (c *GrpcClientConnection) GetGRPCConn() *grpc.ClientConn {
	return c.grpc
}

func GetClientConfiguration(addr string) configuration.Replicator {
	return configuration.Replicator{
		Addr:            addr,
		MaxTransportMsg: 100500,
	}
}
