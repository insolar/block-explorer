// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package exporter

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/testutils"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testExporterSuite struct {
	suite.Suite
	gRPCClient *connection.GRPCClientConnection
	server     *testutils.TestGRPCServer

	testDB     *gorm.DB
	dbCleaner  func()
	repository *storage.Storage

	pulseExporter *PulseServer
}

func (s *testExporterSuite) SetupSuite() {
	ctx := context.Background()

	testDB, dbCleaner, err := testutils.SetupDB()
	if err != nil {
		belogger.FromContext(ctx).Fatal(err)
	}
	s.testDB, s.dbCleaner = testDB.LogMode(true), dbCleaner
	s.repository = storage.NewStorage(s.testDB)
	s.pulseExporter = NewPulseServer(s.repository, time.Second, nil)

	s.server = testutils.CreateTestGRPCServer(s.T(), nil)
	RegisterPulseExporterServer(s.server.Server, s.pulseExporter)
	s.server.Serve(s.T())

	// prepare config with listening address
	cfg := configuration.Replicator{
		Addr:            s.server.Address,
		MaxTransportMsg: 100500,
	}

	gRPCClient, err := connection.NewGRPCClientConnection(ctx, cfg)
	require.NoError(s.T(), err)
	s.gRPCClient = gRPCClient
}

func (s *testExporterSuite) TearDownSuite() {
	s.server.Server.Stop()
	s.gRPCClient.GetGRPCConn().Close()
	s.dbCleaner()
}

func (s *testExporterSuite) TestPulseExporter() {
	t := s.T()

	pulseExporterClient := NewPulseExporterClient(s.gRPCClient.GetGRPCConn())
	ctx := context.Background()

	pulse1 := models.Pulse{PulseNumber: 10, PrevPulseNumber: 1, RecordAmount: 0, IsComplete: true, IsSequential: true}
	pulse2 := models.Pulse{PulseNumber: 20, PrevPulseNumber: 10, RecordAmount: 0, IsComplete: true, IsSequential: true}
	pulse3 := models.Pulse{PulseNumber: 30, PrevPulseNumber: 20, RecordAmount: 1, IsComplete: true, IsSequential: true}
	pulse4 := models.Pulse{PulseNumber: 40, PrevPulseNumber: 30, RecordAmount: 2, IsComplete: true, IsSequential: true}
	pulses := []models.Pulse{pulse1, pulse2, pulse3, pulse4}
	jetDrop1 := testutils.InitJetDropDB(pulse1)
	jetDrop2 := testutils.InitJetDropDB(pulse2)
	jetDrop3 := testutils.InitJetDropDB(pulse3)
	jetDrop4 := testutils.InitJetDropDB(pulse4)
	record31 := testutils.InitRecordDB(jetDrop3)
	record41 := testutils.InitRecordDB(jetDrop4)
	record42 := testutils.InitRecordDB(jetDrop4)

	createAll := func(p models.Pulse, jd models.JetDrop, r []models.Record) {
		err := testutils.CreatePulse(s.testDB, p)
		require.NoError(t, err)
		err = testutils.CreateJetDrop(s.testDB, jd)
		require.NoError(t, err)

		for _, v := range r {
			err = testutils.CreateRecord(s.testDB, v)
			require.NoError(t, err)
		}
	}
	createAll(pulse1, jetDrop1, nil)
	createAll(pulse2, jetDrop2, nil)
	createAll(pulse3, jetDrop3, []models.Record{record31})
	createAll(pulse4, jetDrop4, []models.Record{record41, record42})

	_, i, _ := s.repository.GetPulses(nil, nil, nil, nil, nil, nil, nil, true, 10, 0)
	require.Equal(t, 4, i)

	t.Run("all pulses can be fetched", func(t *testing.T) {
		stream, err := pulseExporterClient.GetNextPulse(ctx,
			&GetNextPulseRequest{pulse1.PrevPulseNumber, [][]byte{}})
		require.NoError(t, err, "Error when sending client request")

		for i := 0; i < 4; i++ {
			response, err := stream.Recv()
			if err != nil {
				t.Fatalf("%v.Export(_) = _, %v", s.gRPCClient, err)
			}
			expected := pulses[i]
			require.NotNil(t, response)
			require.Equal(t, expected.PulseNumber, response.PulseNumber)
			require.Equal(t, expected.PrevPulseNumber, response.PrevPulseNumber)
			require.NoError(t, err, "Err listening stream")
		}
	})

	t.Run("part of pulses can be fetched", func(t *testing.T) {
		stream, err := pulseExporterClient.GetNextPulse(ctx,
			&GetNextPulseRequest{pulse2.PrevPulseNumber, [][]byte{}})
		require.NoError(t, err, "Error when sending client request")

		for i := 1; i < 4; i++ {
			response, err := stream.Recv()
			if err != nil {
				t.Fatalf("%v.Export(_) = _, %v", s.gRPCClient, err)
			}
			expected := pulses[i]
			require.NotNil(t, response)
			require.Equal(t, expected.PulseNumber, response.PulseNumber)
			require.Equal(t, expected.PrevPulseNumber, response.PrevPulseNumber)
			require.NoError(t, err, "Err listening stream")
		}
	})

	t.Run("get pulses with specified prototype reference", func(t *testing.T) {
		stream, err := pulseExporterClient.GetNextPulse(ctx,
			&GetNextPulseRequest{pulse3.PrevPulseNumber, [][]byte{record31.PrototypeReference}})
		require.NoError(t, err, "Error when sending client request")

		response, err := stream.Recv()
		if err != nil {
			t.Fatalf("%v.Export(_) = _, %v", s.gRPCClient, err)
		}
		expected := pulses[2]
		require.NotNil(t, response)
		require.Equal(t, expected.PulseNumber, response.PulseNumber)
		require.Equal(t, int64(1), response.RecordAmount)

	})

	t.Run("get pulses with two prototype reference", func(t *testing.T) {
		stream, err := pulseExporterClient.GetNextPulse(ctx,
			&GetNextPulseRequest{pulse3.PrevPulseNumber, [][]byte{record41.PrototypeReference, record42.PrototypeReference}})
		require.NoError(t, err, "Error when sending client request")

		for i := 2; i < 4; i++ {
			response, err := stream.Recv()
			if err != nil {
				t.Fatalf("%v.Export(_) = _, %v", s.gRPCClient, err)
			}
			expectedRecordAmount := int64(0)
			if i == 3 {
				expectedRecordAmount = 2
			}

			expected := pulses[i]
			require.NotNil(t, response)
			require.Equal(t, expected.PulseNumber, response.PulseNumber)
			require.Equal(t, expectedRecordAmount, response.RecordAmount)
		}
	})
}

func TestPulseExporterTests(t *testing.T) {
	suite.Run(t, new(testExporterSuite))
}
