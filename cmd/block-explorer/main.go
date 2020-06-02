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

	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/etl/connection"
	"github.com/insolar/block-explorer/etl/controller"
	"github.com/insolar/block-explorer/etl/extractor"
	"github.com/insolar/block-explorer/etl/processor"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/instrumentation/belogger"

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

	client, err := connection.NewGrpcClientConnection(ctx, cfg.Replicator)
	if err != nil {
		logger.Fatal("cannot connect to GRPC server", err)
	}
	defer client.GetGRPCConn().Close()

	extractor := extractor.NewMainNetExtractor(100, exporter.NewRecordExporterClient(client.GetGRPCConn()))
	err = extractor.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start extractor", err)
	}
	defer func() {
		err := extractor.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop extractor", err)
		}
	}()

	trn := transformer.NewMainNetTransformer(extractor.GetJetDrops(ctx))
	err = trn.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start transformer", err)
	}
	defer func() {
		err := trn.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop transformer", err)
		}
	}()

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatalf("Error while connecting to database: %s", err.Error())
	}
	defer func() {
		err := db.DB().Close()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to close database"))
		}
	}()

	s := storage.NewStorage(db)

	contr, err := controller.NewController(extractor, s)
	if err != nil {
		logger.Fatal("cannot initialize controller", err)
	}
	err = contr.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start controller", err)
	}
	defer func() {
		err := contr.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop controller", err)
		}
	}()

	proc := processor.NewProcessor(trn, s, contr, 1)
	err = proc.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start processor", err)
	}
	defer func() {
		err := proc.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop processor", err)
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
