// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"net/http"

	echoPrometheus "github.com/globocom/echo-prometheus"
	"github.com/insolar/block-explorer/instrumentation/metrics"
	"github.com/insolar/insconfig"
	"github.com/insolar/spec-insolar-block-explorer-api/v1/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stackimpact/stackimpact-go"

	"github.com/insolar/block-explorer/api"
	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/dbconn"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
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

	_ = stackimpact.Start(stackimpact.Options{
		AgentKey: "5256279e53f4aa857af6ee782a4c53e72034b0da",
		AppName:  "api",
	})

	router := api.NewRouter()
	err := router.Start(ctx)
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
