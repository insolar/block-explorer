// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"context"

	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

type PlatformPulseExtractor struct {
	client exporter.PulseExporterClient
}

func NewPlatformPulseExtractor(client exporter.PulseExporterClient) *PlatformPulseExtractor {
	return &PlatformPulseExtractor{
		client: client,
	}
}

func (ppe *PlatformPulseExtractor) GetCurrentPulse(ctx context.Context) (uint32, error) {
	return ppe.fetchCurrentPulse(ctx)
}

// fetchCurrentPulse returns the current pulse number
func (ppe *PlatformPulseExtractor) fetchCurrentPulse(ctx context.Context) (uint32, error) {
	client := ppe.client
	request := &exporter.GetTopSyncPulse{}
	log := belogger.FromContext(ctx)
	log.Debug("Fetching top sync pulse")

	tsp, err := client.TopSyncPulse(ctx, request)
	if err != nil {
		log.WithField("request", request).Error(errors.Wrapf(err, "failed to get TopSyncPulse"))
		return 0, err
	}

	log.Debug("Received top sync pulse ", tsp.PulseNumber)
	return tsp.PulseNumber, nil
}

func (ppe *PlatformPulseExtractor) GetNextFinalizedPulse(ctx context.Context, p int64) (*exporter.FullPulse, error) {
	c := ppe.client
	req := &exporter.GetNextFinalizedPulse{p}

	log := belogger.FromContext(ctx)
	log.Debug("GetNextFinalizedPulse")

	// fatal error: signal_recv: inconsistent state
	ret, err := c.NextFinalizedPulse(ctx, req)
	if err != nil {
		log.WithField("request", req).Error(errors.Wrap(err, "failed to GetNextFinalizedPulse()"))
		return nil, err
	}
	return ret, nil
}
