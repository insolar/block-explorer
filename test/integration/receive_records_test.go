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
	"github.com/insolar/block-explorer/testutils/connection_manager"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type receiveRecordsSuite struct {
	suite.Suite
	c connection_manager.ConnectionManager
}

func (a *receiveRecordsSuite) SetupSuite() {
	a.c.Start(a.T())
}

func (a *receiveRecordsSuite) TearDownSuite() {
	a.c.Stop()
}

func (a *receiveRecordsSuite) TestGetRecords_simpleRecord() {
	a.T().Skip("https://insolar.atlassian.net/browse/PENV-295")
	request := &exporter.GetRecords{
		Count: uint32(5),
	}

	stream, err := a.c.ExporterClient.Export(context.Background(), request)
	require.NoError(a.T(), err, "Error when sending client request")

	var res []exporter.Record
	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(a.T(), err, "Err listening stream")
		require.True(a.T(), heavymock.SimpleRecord.Equal(record), "Incorrect response message")
		a.T().Logf("received record: %v", record)
		res = append(res, *record)
	}
	require.Len(a.T(), res, int(request.Count))
}

func (a *receiveRecordsSuite) TestGetRecords_pulseRecords() {
	a.T().Skip("https://insolar.atlassian.net/browse/PENV-295")
	expPulse := gen.PulseNumber()
	request := &exporter.GetRecords{
		Count:       uint32(5),
		PulseNumber: expPulse,
	}

	stream, err := a.c.ExporterClient.Export(context.Background(), request)
	require.NoError(a.T(), err, "Error when sending client request")

	var res []exporter.Record
	for {
		record, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(a.T(), err, "Err listening stream")
		a.T().Logf("received record: %v", record)
		require.Equal(a.T(), &expPulse, record.ShouldIterateFrom, "Incorrect record pulse number")
		res = append(res, *record)
	}
	require.Len(a.T(), res, int(request.Count))
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
		Polymorph: heavymock.MagicPolymorphExport,
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
		require.NoError(a.T(), err, "Err listening stream")
		a.T().Logf("received record: %v", record)
		require.True(a.T(), heavymock.SimpleRecord.Equal(record), "Incorrect record pulse number")
	}
	require.Equal(a.T(), recordsCount, c)
}

func TestAllTests(t *testing.T) {
	suite.Run(t, new(receiveRecordsSuite))
}
