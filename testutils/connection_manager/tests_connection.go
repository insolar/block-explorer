// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package connection_manager

import (
	"context"
	"testing"

	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
)

// struct that represents all connections that can be used throughout integration tests
type ConnectionManager struct {
	grpcServer     *testutils.TestGRPCServer
	grpcClientConn *connection.GrpcClientConnection
	ExporterClient exporter.RecordExporterClient
	ImporterClient heavymock.HeavymockImporterClient
	Importer       *heavymock.ImporterServer
	DB             *gorm.DB
	dbPoolCleaner  func()
}

// Starts GRPC server and initializes connection from GRPC clients
func (c *ConnectionManager) Start(t *testing.T) {
	var err error
	c.grpcServer = testutils.CreateTestGRPCServer(t)
	c.Importer = heavymock.NewHeavymockImporter()
	heavymock.RegisterHeavymockImporterServer(c.grpcServer.Server, c.Importer)
	exporter.RegisterRecordExporterServer(c.grpcServer.Server, heavymock.NewRecordExporter(c.Importer))
	c.grpcServer.Serve(t)

	ctx := context.Background()
	cfg := connection.GetClientConfiguration(c.grpcServer.Address)

	c.grpcClientConn, err = connection.NewGrpcClientConnection(ctx, cfg)
	require.NoError(t, err)

	c.ExporterClient = exporter.NewRecordExporterClient(c.grpcClientConn.GetGRPCConn())
	c.ImporterClient = heavymock.NewHeavymockImporterClient(c.grpcClientConn.GetGRPCConn())
}

// run postgres in docker and perform migrations
func (c *ConnectionManager) StartDB(t *testing.T) {
	db, poolCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	c.DB = db
	c.dbPoolCleaner = poolCleaner
}

// close all opened connections
func (c *ConnectionManager) Stop() {
	c.grpcServer.Server.Stop()
	c.grpcClientConn.GetGRPCConn().Close()
	if f := c.dbPoolCleaner; f != nil {
		f()
	}
}
