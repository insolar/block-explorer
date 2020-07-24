// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package integration

import (
	"context"
	"io"
	"testing"

	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type receiveRecordsSuite struct {
	suite.Suite
	c connectionmanager.ConnectionManager
}

func (a *receiveRecordsSuite) SetupSuite() {
	a.c.Start(a.T())
}

func (a *receiveRecordsSuite) TearDownSuite() {
	a.c.Stop()
}

func (a *receiveRecordsSuite) TestReceiveRecords_sendAndReceiveWithImporter() {
	var expectedRecords []exporter.Record
	recordsCount := 10
	for i := 0; i < recordsCount; i++ {
		expectedRecords = append(expectedRecords, *heavymock.SimpleRecord)
	}

	stream, err := a.c.ImporterClient.Import(context.Background())
	require.NoError(a.T(), err)
	for _, record := range expectedRecords {
		if err := stream.Send(&record); err != nil {
			if err == io.EOF {
				break
			}
			a.T().Fatal("Error sending to stream", err)
		}
	}
	reply, err := stream.CloseAndRecv()
	require.NoError(a.T(), err)
	require.True(a.T(), reply.Ok)

	request := &exporter.GetRecords{
		PulseNumber: heavymock.SimpleRecord.Record.ID.Pulse(),
	}

	expStream, err := a.c.ExporterClient.Export(context.Background(), request)
	require.NoError(a.T(), err, "Error when sending export request")

	var c int
	for {
		record, err := expStream.Recv()
		if err == io.EOF {
			break
		}
		c++
		if c == recordsCount { // lastrecord
			break
		}
		require.NoError(a.T(), err, "Err listening stream")
		require.Equal(a.T(), heavymock.SimpleRecord.Record.ID.Pulse(), record.Record.ID.Pulse(), "Incorrect record pulse number")
	}
	require.Equal(a.T(), recordsCount, c)
}

func TestAllTests(t *testing.T) {
	suite.Run(t, new(receiveRecordsSuite))
}
