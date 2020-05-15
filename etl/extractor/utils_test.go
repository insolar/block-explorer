// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	insrecord "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// create test grpc server
func createGRPCServer(t *testing.T) (address string, server *grpc.Server) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "failed to listen")
	grpcServer := grpc.NewServer()

	// need to run grpcServer.Serve in different goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			require.Error(t, err, "server exited with error")
			return
		}
	}()
	return listener.Addr().Network(), grpcServer
}

// return a function for generating record
func generateRecords(batchSize int) func() (record *exporter.Record, e error) {
	pn := insolar.PulseNumber(10000000)
	totalRecs := 1
	cnt := 0
	eof := true

	generateRecords := func() (record *exporter.Record, e error) {
		fmt.Println("Start generating records number " + string(cnt))
		if !eof && cnt%batchSize == 0 {
			eof = true
			return &exporter.Record{}, io.EOF
		}
		cnt++
		eof = false
		if cnt > totalRecs {
			return &exporter.Record{}, io.EOF
		}
		return &exporter.Record{
			RecordNumber: uint32(cnt),
			Record: insrecord.Material{
				ID: gen.IDWithPulse(pn),
			},
			ShouldIterateFrom: nil,
		}, nil
	}

	return generateRecords
}
