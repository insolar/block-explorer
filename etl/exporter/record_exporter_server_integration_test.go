// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build integration

package exporter

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/testutils"
)

var getRecordGRPCConnection *grpc.ClientConn
var testDB *gorm.DB

func TestMain(m *testing.M) {
	// create storage
	ctx := context.Background()
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}
	var dbCleaner func()
	testDB, dbCleaner, err = testutils.SetupDB()
	if err != nil {
		belogger.FromContext(context.Background()).Fatal(err)
	}

	cfg := configuration.Exporter{
		Listen: fmt.Sprintf(":%d", port),
	}
	s := storage.NewStorage(testDB)

	// create grpc server
	recordExporter := NewRecordServer(ctx, s, cfg)
	grpcServer := grpc.NewServer()
	RegisterRecordExporterServer(grpcServer, recordExporter)
	exporterServer := NewServer(cfg.Listen, grpcServer)
	err = exporterServer.Start(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		exporterServer.Stop(ctx)
		grpcServer.GracefulStop()
		dbCleaner()
	}()

	// create client
	serverAddr := fmt.Sprintf("localhost:%d", port)
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure(), grpc.WithBlock())
	getRecordGRPCConnection, err = grpc.Dial(serverAddr, opts...)
	if err != nil {
		panic(err)
	}
	defer getRecordGRPCConnection.Close()
	retCode := m.Run()
	os.Exit(retCode)
}

func TestRecordServer_GetRecords(t *testing.T) {
	var accountPrototypeReference, _ = insolar.NewObjectReferenceFromString("insolar:0AAABAiGN1L8F9gCH_keBaxOP4atp9fzLiIci7xOg-hs")
	ctx := context.Background()

	pulse, err := testutils.InitPulseDB()
	require.NoError(t, err)
	err = testutils.CreatePulse(testDB, pulse)
	require.NoError(t, err)
	jetDrop1 := testutils.InitJetDropDB(pulse)
	jetDrop1.JetID = "10101"
	err = testutils.CreateJetDrop(testDB, jetDrop1)
	require.NoError(t, err)
	recordResult := testutils.InitRecordDB(jetDrop1)
	recordResult.Type = models.Result
	recordResult.Order = 1
	recordResult.PrototypeReference = accountPrototypeReference.Bytes()
	err = testutils.CreateRecord(testDB, recordResult)
	require.NoError(t, err)
	recordState1 := testutils.InitRecordDB(jetDrop1)
	recordState1.Order = 2
	recordState1.PrototypeReference = accountPrototypeReference.Bytes()
	err = testutils.CreateRecord(testDB, recordState1)
	require.NoError(t, err)
	recordState2 := testutils.InitRecordDB(jetDrop1)
	recordState2.Order = 3
	recordState2.PrototypeReference = accountPrototypeReference.Bytes()
	err = testutils.CreateRecord(testDB, recordState2)
	require.NoError(t, err)

	client := NewRecordExporterClient(getRecordGRPCConnection)

	t.Run("happy", func(t *testing.T) {
		var recordsResponses []*GetRecordsResponse
		request := GetRecordsRequest{
			PulseNumber:  pulse.PulseNumber,
			Prototypes:   [][]byte{accountPrototypeReference.Bytes()},
			RecordNumber: 0,
			Count:        3,
		}
		stream, err := client.GetRecords(ctx, &request)
		require.Nil(t, err)
		for {
			pulseResp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			recordsResponses = append(recordsResponses, pulseResp)
		}
		require.Len(t, recordsResponses, 3)
		require.Equal(t, []byte(recordResult.Reference), recordsResponses[0].Reference)
		require.Equal(t, []byte(recordState1.Reference), recordsResponses[1].Reference)
		require.Equal(t, []byte(recordState2.Reference), recordsResponses[2].Reference)
	})

	t.Run("count", func(t *testing.T) {
		var recordsResponses []*GetRecordsResponse
		request := GetRecordsRequest{
			PulseNumber:  pulse.PulseNumber,
			Prototypes:   [][]byte{accountPrototypeReference.Bytes()},
			RecordNumber: 0,
			Count:        2,
		}
		stream, err := client.GetRecords(ctx, &request)
		require.Nil(t, err)
		for {
			pulseResp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			recordsResponses = append(recordsResponses, pulseResp)
		}
		require.Len(t, recordsResponses, 2)
		require.Equal(t, []byte(recordResult.Reference), recordsResponses[0].Reference)
		require.Equal(t, []byte(recordState1.Reference), recordsResponses[1].Reference)
	})

	t.Run("recordNumber", func(t *testing.T) {
		var recordsResponses []*GetRecordsResponse
		request := GetRecordsRequest{
			PulseNumber:  pulse.PulseNumber,
			Prototypes:   [][]byte{accountPrototypeReference.Bytes()},
			RecordNumber: 1,
			Count:        1,
		}
		stream, err := client.GetRecords(ctx, &request)
		require.Nil(t, err)
		for {
			pulseResp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			recordsResponses = append(recordsResponses, pulseResp)
		}
		require.Len(t, recordsResponses, 1)
		require.Equal(t, []byte(recordState1.Reference), recordsResponses[0].Reference)
	})

	t.Run("no protoref", func(t *testing.T) {
		var recordsResponses []*GetRecordsResponse
		request := GetRecordsRequest{
			PulseNumber:  pulse.PulseNumber,
			Prototypes:   [][]byte{gen.Reference().Bytes()},
			RecordNumber: 1,
			Count:        1,
		}
		stream, err := client.GetRecords(ctx, &request)
		require.Nil(t, err)
		for {
			pulseResp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			recordsResponses = append(recordsResponses, pulseResp)
		}
		require.Len(t, recordsResponses, 0)
	})
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
