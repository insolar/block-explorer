// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/jinzhu/gorm"
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
	ctx := context.Background()
	ctx, logger := belogger.InitLogger(ctx, cfg.Log, "block_explorer")
	logger.Info("Config and logger were initialized")

	client, err := connection.NewGrpcClientConnection(ctx, cfg.Replicator)
	if err != nil {
		// TODO: change to logger after PENV-279
		log.Fatal("cannot connect to GRPC server", err)
	}
	defer client.GetGRPCConn().Close()

	extractor := extractor.NewMainNetExtractor(100, exporter.NewRecordExporterClient(client.GetGRPCConn()))

	trn := transformer.NewMainNetTransformer(extractor.GetJetDrops(ctx))
	err = trn.Start(ctx)
	if err != nil {
		// TODO: change to logger after PENV-279
		log.Fatal("cannot connect to GRPC server", err)
	}
	defer trn.Stop(ctx)

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		// TODO: change to logger after PENV-279
		// logger.Fatalf("Error while connecting to database: %s", err.Error())
		fmt.Printf("Error while connecting to database: %s\n", err.Error())
		return
	}

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

	graceful(ctx, makeStopper(ctx, db))
}

func graceful(ctx context.Context, that func()) {
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger := belogger.FromContext(ctx)
	logger.Infof("gracefully stopping...")
	that()
}

func makeStopper(ctx context.Context, db *gorm.DB) func() {
	logger := belogger.FromContext(ctx)
	return func() {
		err := db.DB().Close()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to close database"))
		}
	}
}
