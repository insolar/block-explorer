// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package controller

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/block-explorer/instrumentation/metrics"
)

var (
	IncompletePulsesQueue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_controller_data_queue",
		Help: "The number of pulses in controller's incomplete pulses queue",
	})
	CurrentSeqPulse = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_controller_current_seq_pulse",
		Help: "Current sequentual pulse rerequested from platform",
	})
	CurrentIncompletePulse = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_controller_current_incomplete_pulse",
		Help: "Current incomplete pulse that records are rerequested from platform",
	})
	PulseCompleteCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_controller_pulse_complete_counter",
		Help: "How many pulses is completed by 'pulseIsComplete' check",
	})
	PulseNotCompleteCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_controller_pulse_not_complete_counter",
		Help: "How many pulses is not completed by 'pulseIsComplete' check",
	})
)

type Metrics struct{}

func (s Metrics) Refresh() {
	// nothing to refresh
}

func (s Metrics) Metrics(p *metrics.Prometheus) []prometheus.Collector {
	return []prometheus.Collector{
		IncompletePulsesQueue,
		CurrentSeqPulse,
		CurrentIncompletePulse,
		PulseNotCompleteCounter,
		PulseCompleteCounter,
	}
}
