package connectionmanager

import (
	"context"
	"net"
	"testing"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/api"
	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/test/heavymock"
	"github.com/insolar/block-explorer/testutils"
)

const DefaultAPIPort = ":8081"

// struct that represents all connections that can be used throughout integration tests
type ConnectionManager struct {
	grpcServer     *testutils.TestGRPCServer
	grpcClientConn *connection.GRPCClientConnection
	ExporterClient exporter.RecordExporterClient
	ImporterClient heavymock.HeavymockImporterClient
	Importer       *heavymock.ImporterServer
	DB             *gorm.DB
	dbPoolCleaner  func()
	echo           *echo.Echo
	ctx            context.Context
}

// Starts GRPC server and initializes connection from GRPC clients
func (c *ConnectionManager) Start(t testing.TB) {
	var err error
	// todo get config ConnectionConfig from test and insert into CreateTestGRPCServer
	c.grpcServer = testutils.CreateTestGRPCServer(t, nil)
	c.Importer = heavymock.NewHeavymockImporter()
	heavymock.RegisterHeavymockImporterServer(c.grpcServer.Server, c.Importer)
	exporter.RegisterRecordExporterServer(c.grpcServer.Server, heavymock.NewRecordExporter(c.Importer))
	c.grpcServer.Serve(t)

	ctx := context.Background()
	c.ctx = ctx
	cfg := connection.GetClientConfiguration(c.grpcServer.Address)

	c.grpcClientConn, err = connection.NewGRPCClientConnection(c.ctx, cfg)
	require.NoError(t, err)

	c.ExporterClient = exporter.NewRecordExporterClient(c.grpcClientConn.GetGRPCConn())
	c.ImporterClient = heavymock.NewHeavymockImporterClient(c.grpcClientConn.GetGRPCConn())
}

// run postgres in docker and perform migrations
func (c *ConnectionManager) StartDB(t testing.TB) {
	db, poolCleaner, err := testutils.SetupDB()
	require.NoError(t, err)
	c.DB = db
	c.dbPoolCleaner = poolCleaner
}

// start API server
func (c *ConnectionManager) StartAPIServer(t testing.TB) {
	e := echo.New()
	c.echo = e

	if c.DB == nil {
		t.Fatal("DB not initialized")
	}
	s := storage.NewStorage(c.DB)

	cfg := configuration.API{
		Listen: DefaultAPIPort,
	}
	apiServer := api.NewServer(c.ctx, s, cfg)
	server.RegisterHandlers(e, apiServer)

	l, err := net.Listen("tcp", cfg.Listen)
	require.NoError(t, err, "can't start listen")
	c.echo.Listener = l
	go func() {
		err := c.echo.Start(cfg.Listen)
		if err != nil {
			require.Contains(t, err.Error(), "http: Server closed", "HTTP server stopped unexpected")
		}
	}()
}

// close all opened connections
func (c *ConnectionManager) Stop() {
	c.grpcServer.Server.Stop()
	c.grpcClientConn.GetGRPCConn().Close()
	if f := c.dbPoolCleaner; f != nil {
		f()
	}
	if e := c.echo; e != nil {
		_ = c.echo.Shutdown(c.ctx)
	}
}
