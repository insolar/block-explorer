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

	"github.com/insolar/block-explorer/api"
	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/etl/controller"
	"github.com/insolar/block-explorer/etl/dbconn/reconnect"
	"github.com/insolar/block-explorer/etl/extractor"
	"github.com/insolar/block-explorer/etl/processor"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/dbconn"
	"github.com/insolar/block-explorer/etl/storage"

	"github.com/insolar/block-explorer/configuration"
)

var stop = make(chan os.Signal, 1)

func main() {
	cfg := &configuration.BlockExplorer{}
	params := insconfig.Params{
		EnvPrefix:        "block_explorer",
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	fmt.Println("Starts with configuration:\n", insConfigurator.ToYaml(cfg))
	ctx, logger := belogger.InitLogger(context.Background(), cfg.Log, "block_explorer")
	logger.Info("Config and logger were initialized")

	router := api.NewRouter()
	_ = router.Start(ctx)
	defer func() {
		err := router.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop pprof: ", err)
		}
	}()

	client, err := connection.NewGRPCClientConnection(ctx, cfg.Replicator)
	if err != nil {
		logger.Fatal("cannot connect to GRPC server: ", err)
	}
	defer client.GetGRPCConn().Close()

	pulseExtractor := extractor.NewPlatformPulseExtractor(exporter.NewPulseExporterClient(client.GetGRPCConn()))
	platformExtractor := extractor.NewPlatformExtractor(100, pulseExtractor, exporter.NewRecordExporterClient(client.GetGRPCConn()))
	err = platformExtractor.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start platformExtractor: ", err)
	}
	defer func() {
		err := platformExtractor.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop platformExtractor: ", err)
		}
	}()

	mainNetTransformer := transformer.NewMainNetTransformer(platformExtractor.GetJetDrops(ctx))
	err = mainNetTransformer.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start transformer: ", err)
	}
	defer func() {
		err := mainNetTransformer.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop transformer: ", err)
		}
	}()

	connectFn := dbconn.ConnectFn(cfg.DB)
	db, err := connectFn()
	if err != nil {
		logger.Fatalf("Error while connecting to database: %s", err.Error())
	}
	defer func() {
		err := db.DB().Close()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to close database").Error())
		}
	}()

	r := reconnect.New(cfg.DB.Reconnect, connectFn)
	r.Apply(db)

	storage := storage.NewStorage(db)

	controller, err := controller.NewController(cfg.Controller, platformExtractor, storage)
	if err != nil {
		logger.Fatal("cannot initialize controller: ", err)
	}
	err = controller.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start controller: ", err)
	}
	defer func() {
		err := controller.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop controller: ", err)
		}
	}()

	proc := processor.NewProcessor(mainNetTransformer, storage, controller, cfg.Processor.Workers)
	err = proc.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start processor: ", err)
	}
	defer func() {
		err := proc.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop processor: ", err)
		}
	}()

	graceful(ctx)
}

func graceful(ctx context.Context) {
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger := belogger.FromContext(ctx)
	logger.Infof("gracefully stopping...")
}
