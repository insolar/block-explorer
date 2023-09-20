package main

import (
	"context"
	"fmt"
	"net/http"

	echoPrometheus "github.com/globocom/echo-prometheus"
	"github.com/insolar/block-explorer/api"
	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/dbconn"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/instrumentation/metrics"
	"github.com/insolar/block-explorer/instrumentation/profefe"
	"github.com/insolar/insconfig"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := &configuration.API{}
	params := insconfig.Params{
		EnvPrefix:        "block_explorer_api",
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	fmt.Println("Starts with configuration:\n", insConfigurator.ToYaml(cfg))
	ctx := context.Background()
	ctx, logger := belogger.InitLogger(ctx, cfg.Log, "block_explorer_api")
	logger.Info("Config and logger were initialized")

	pfefe := profefe.New(cfg.Profefe, "block_explorer_api")
	err := pfefe.Start(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		err := pfefe.Stop(ctx)
		if err != nil {
			logger.Error(err)
		}
	}()

	router := api.NewRouter()
	err = router.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start pprof: ", err)
	}

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatalf("Error while connecting to database: %s", err.Error())
		return
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echoPrometheus.MetricsMiddleware())

	metricConfig := metrics.Config{
		RefreshInterval: cfg.Metrics.RefreshInterval,
		StartServer:     cfg.Metrics.StartServer,
		HTTPServerPort:  cfg.Metrics.HTTPServerPort,
		MetricsCollectors: []metrics.Collector{
			storage.NewStatsCollector(db, nil),
			storage.Metrics{},
		},
	}

	_ = metrics.New(metricConfig).Initialize()

	s := storage.NewStorage(db)

	apiServer := api.NewServer(ctx, s, *cfg)
	server.RegisterHandlers(e, apiServer)

	srv := &http.Server{
		Addr:         cfg.Listen,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	e.Logger.Fatal(e.StartServer(srv))
}
