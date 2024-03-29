package betestsetup

import (
	"context"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/testutils/clients"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/jinzhu/gorm"

	"github.com/insolar/block-explorer/etl/controller"
	"github.com/insolar/block-explorer/etl/extractor"
	"github.com/insolar/block-explorer/etl/processor"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/etl/transformer"
)

// used in tests to quickly initialize block-explorer processes
type BlockExplorerTestSetUp struct {
	ExporterClient exporter.RecordExporterClient
	DB             *gorm.DB
	PulseClient    *clients.TestPulseClient

	extr *extractor.PlatformExtractor
	cont *controller.Controller
	proc *processor.Processor
	trsf *transformer.MainNetTransformer
	strg *storage.Storage
	ctx  context.Context
}

func NewBlockExplorer(exporterClient exporter.RecordExporterClient, db *gorm.DB) BlockExplorerTestSetUp {
	return BlockExplorerTestSetUp{
		ExporterClient: exporterClient,
		DB:             db,
		PulseClient:    clients.GetTestPulseClient(1, nil),
	}
}

var cfg = configuration.Controller{
	PulsePeriod:       10,
	SequentialPeriod:  1,
	ReloadPeriod:      10,
	ReloadCleanPeriod: 1,
}

// start Extractor, Transformer, Controller and Processor
func (b *BlockExplorerTestSetUp) Start() error {
	b.ctx = context.Background()

	pulseExtractor := extractor.NewPlatformPulseExtractor(b.PulseClient)
	b.extr = extractor.NewPlatformExtractor(100, 0, 100, 100, pulseExtractor, b.ExporterClient, func() {})
	err := b.extr.Start(b.ctx)
	if err != nil {
		return err
	}

	b.trsf = transformer.NewMainNetTransformer(b.extr.GetJetDrops(b.ctx), 100)
	err = b.trsf.Start(b.ctx)
	if err != nil {
		return err
	}

	b.strg = storage.NewStorage(b.DB)
	b.cont, err = controller.NewController(cfg, b.extr, b.strg, 2)
	if err != nil {
		return err
	}
	b.proc = processor.NewProcessor(b.trsf, b.strg, b.cont, 1)
	err = b.proc.Start(b.ctx)
	if err != nil {
		return err
	}

	return nil
}

// Stop Transformer and Processor
func (b *BlockExplorerTestSetUp) Stop() error {
	if err := b.extr.Stop(b.ctx); err != nil {
		return err
	}
	if err := b.trsf.Stop(b.ctx); err != nil {
		return err
	}
	if err := b.proc.Stop(b.ctx); err != nil {
		return err
	}
	return nil
}

func (b *BlockExplorerTestSetUp) Extractor() *extractor.PlatformExtractor {
	return b.extr
}

func (b *BlockExplorerTestSetUp) Controller() *controller.Controller {
	return b.cont
}

func (b *BlockExplorerTestSetUp) Processor() *processor.Processor {
	return b.proc
}

func (b *BlockExplorerTestSetUp) Transformer() *transformer.MainNetTransformer {
	return b.trsf
}

func (b *BlockExplorerTestSetUp) Storage() *storage.Storage {
	return b.strg
}
