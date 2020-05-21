// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package heavymock

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type ImporterClient struct {
	grpc *grpc.ClientConn
}

func NewImporterClient(addr string) (*ImporterClient, error) {
	c, e := func() (*grpc.ClientConn, error) {
		options := grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100500),
			grpc.MaxCallSendMsgSize(100500),
		)

		// We omit error here because connect happens in background.
		conn, err := grpc.Dial(addr, options, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to grpc.Dial")
		}
		return conn, err
	}()

	if e != nil {
		return &ImporterClient{}, e
	}

	return &ImporterClient{c}, nil
}

func (c *ImporterClient) GetGRPCConn() *grpc.ClientConn {
	return c.grpc
}
