// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/block-explorer/instrumentation/metrics"
)

var (
	DataQueue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_transformer_data_queue",
		Help: "The number of jetdrops in transformer export data queue",
	})
	TransformedPulses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_transformer_pulses",
		Help: "The number of transformed pulses",
	})
	TransformedRecords = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_transformer_records",
		Help: "The number of transformed records",
	})
	Errors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_transformer_errors",
		Help: "The number of errors received during data transforming",
	})
)

type Metrics struct{}

func (s Metrics) Refresh() {
	// nothing to refresh
}

func (s Metrics) Metrics(p *metrics.Prometheus) []prometheus.Collector {
	_ = prometheus.Register(DataQueue)
	_ = prometheus.Register(TransformedPulses)
	_ = prometheus.Register(TransformedRecords)
	_ = prometheus.Register(Errors)

	return []prometheus.Collector{}
}
