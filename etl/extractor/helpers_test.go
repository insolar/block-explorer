// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"google.golang.org/grpc"
)

type recordStream struct {
	grpc.ClientStream
	recvFunc func() (*exporter.Record, error)
}

func (s recordStream) Recv() (*exporter.Record, error) {
	return s.recvFunc()
}

func (c recordStream) CloseSend() error {
	return nil
}
