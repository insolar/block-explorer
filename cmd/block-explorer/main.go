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
	"github.com/insolar/block-explorer/etl/dbconn"
	"github.com/insolar/block-explorer/etl/dbconn/plugins"
	"github.com/insolar/block-explorer/etl/extractor"
	"github.com/insolar/block-explorer/etl/processor"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/instrumentation/metrics"
	"github.com/insolar/block-explorer/instrumentation/profefe"
	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/block-explorer/configuration"
)

var stop = make(chan os.Signal, 1)

// stopChannel is the global channel where you have to send signal to stop application
var stopChannel = make(chan struct{})

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

	pfefe := profefe.New(cfg.Profefe, "block-explorer")
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
	client.NotifyShutdown(ctx, stopChannel, cfg.Replicator.WaitForConnectionRecoveryTimeout)

	pulseExtractor := extractor.NewPlatformPulseExtractor(exporter.NewPulseExporterClient(client.GetGRPCConn()))
	platformExtractor := extractor.NewPlatformExtractor(
		100,
		cfg.Replicator.ContinuousPulseRetrievingHalfPulseSeconds,
		int32(cfg.Replicator.ParallelConnections),
		cfg.Replicator.QueueLen,
		pulseExtractor,
		exporter.NewRecordExporterClient(client.GetGRPCConn()),
		shutdownBE,
	)
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

	mainNetTransformer := transformer.NewMainNetTransformer(
		platformExtractor.GetJetDrops(ctx),
		cfg.Transformer.QueueLen,
	)
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

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatalf("Error while connecting to database: %s", err.Error())
	}
	defer func() {
		err := db.DB().Close()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to close database").Error())
		}
	}()

	db.SetLogger(belogger.NewGORMLogAdapter(logger))

	r := plugins.NewDefaultShutdownPlugin(stopChannel)
	r.Apply(db)

	repository := storage.NewStorage(db)

	gbeController, err := controller.NewController(cfg.Controller, platformExtractor, repository, cfg.Replicator.PlatformVersion)
	if err != nil {
		logger.Fatal("cannot initialize gbeController: ", err)
	}
	err = gbeController.Start(ctx)
	if err != nil {
		logger.Fatal("cannot start gbeController: ", err)
	}
	defer func() {
		err := gbeController.Stop(ctx)
		if err != nil {
			logger.Fatal("cannot stop gbeController: ", err)
		}
	}()

	proc := processor.NewProcessor(mainNetTransformer, repository, gbeController, cfg.Processor.Workers)
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

	metricConfig := metrics.Config{
		RefreshInterval: cfg.Metrics.RefreshInterval,
		StartServer:     cfg.Metrics.StartServer,
		HTTPServerPort:  cfg.Metrics.HTTPServerPort,
		MetricsCollectors: []metrics.Collector{
			storage.NewPostgresCollector(nil, db),
			storage.NewStatsCollector(db, nil),
			storage.Metrics{},
			extractor.Metrics{},
			transformer.Metrics{},
			controller.Metrics{},
		},
	}

	_ = metrics.New(metricConfig).Initialize()

	graceful(ctx)
}

func graceful(ctx context.Context) {
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	logger := belogger.FromContext(ctx)
	select {
	case <-stopChannel:
		logger.Info("gracefully stopping by channel")
	case <-stop:
		logger.Info("gracefully stopping by signal")
	}
}

func shutdownBE() {
	stopChannel <- struct{}{}
}
