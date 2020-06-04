// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package betestsetup

import (
	"context"

	"github.com/insolar/block-explorer/etl/controller"
	"github.com/insolar/block-explorer/etl/extractor"
	"github.com/insolar/block-explorer/etl/interfaces"
	"github.com/insolar/block-explorer/etl/processor"
	"github.com/insolar/block-explorer/etl/storage"
	"github.com/insolar/block-explorer/etl/transformer"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/jinzhu/gorm"
)

// used in tests to quickly initialize block-explorer processes
type BlockExplorerTestSetUp struct {
	ExporterClient exporter.RecordExporterClient
	DB             *gorm.DB

	extr *extractor.PlatformExtractor
	cont *controller.Controller
	proc *processor.Processor
	trsf *transformer.MainNetTransformer
	strg interfaces.Storage
	ctx  context.Context
}

func NewBlockExplorer(exporterClient exporter.RecordExporterClient, db *gorm.DB) BlockExplorerTestSetUp {
	return BlockExplorerTestSetUp{
		ExporterClient: exporterClient,
		DB:             db,
	}
}

// start Extractor, Transformer, Controller and Processor
func (b *BlockExplorerTestSetUp) Start() error {
	b.ctx = context.Background()

	b.extr = extractor.NewPlatformExtractor(100, b.ExporterClient)
	err := b.extr.Start(b.ctx)
	if err != nil {
		return err
	}

	b.trsf = transformer.NewMainNetTransformer(b.extr.GetJetDrops(b.ctx))
	err = b.trsf.Start(b.ctx)
	if err != nil {
		return err
	}

	b.strg = storage.NewStorage(b.DB)
	b.cont, err = controller.NewController(b.extr, b.strg)
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

func (b *BlockExplorerTestSetUp) Storage() interfaces.Storage {
	return b.strg
}
