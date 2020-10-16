// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/etl/dbconn"
	"github.com/insolar/block-explorer/etl/exporter"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/instrumentation/metrics"
	"github.com/insolar/block-explorer/instrumentation/profefe"
	"github.com/insolar/insconfig"
)

var stop = make(chan os.Signal, 1)

func main() {
	cfg := &configuration.Exporter{}
	params := insconfig.Params{
		EnvPrefix:        "block_explorer_exporter_api",
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	fmt.Println("Starts with configuration:\n", insConfigurator.ToYaml(cfg))
	ctx := context.Background()
	ctx, logger := belogger.InitLogger(ctx, cfg.Log, "block_explorer_exporter_api")
	logger.Info("Config and logger were initialized")

	pfefe := profefe.New(cfg.Profefe, "block_explorer_exporter_api")
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

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatalf("Error while connecting to database: %s", err.Error())
		return
	}
	s := storage.NewStorage(db)

	metricConfig := metrics.Config{
		RefreshInterval:   cfg.Metrics.RefreshInterval,
		StartServer:       cfg.Metrics.StartServer,
		HTTPServerPort:    cfg.Metrics.HTTPServerPort,
		MetricsCollectors: []metrics.Collector{},
	}

	_ = metrics.New(metricConfig).Initialize()

	var (
		recordExporter *exporter.RecordServer
		pulseExporter  *exporter.PulseServer
	)

	recordExporter = exporter.NewRecordServer()
	pulseExporter = exporter.NewPulseServer(s, cfg.PulsePeriod, &logger)

	grpcMetrics := grpc_prometheus.NewServerMetrics()
	grpcMetrics.EnableHandlingTimeHistogram()

	grpcServer, err := connection.NewGRPCServer(*cfg, grpcMetrics)
	if err != nil {
		logger.Fatal("failed to initiate a GRPC server: ", err)
	}
	exporter.RegisterRecordExporterServer(grpcServer, recordExporter)
	exporter.RegisterPulseExporterServer(grpcServer, pulseExporter)

	grpcMetrics.InitializeMetrics(grpcServer)

	exporterServer := exporter.NewServer(cfg.Listen, grpcServer)
	err = exporterServer.Start(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		err := exporterServer.Stop(ctx)
		if err != nil {
			logger.Error(err)
		}
	}()

	graceful(ctx)
}

func graceful(ctx context.Context) {
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	logger := belogger.FromContext(ctx)
	<-stop
	logger.Info("gracefully stopping by signal")
}
