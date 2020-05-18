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

type MainnetClient struct {
	grpc *grpc.ClientConn
}

// NewMainNetClient returns implementation
func NewMainNetClient(ctx context.Context, cfg configuration.Replicator) (*MainnetClient, error) {
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
		return &MainnetClient{}, e
	}

	return &MainnetClient{c}, nil
}

func (c *MainnetClient) GetGRPCConn() *grpc.ClientConn {
	return c.grpc
}
