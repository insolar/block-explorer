// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package extractor

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/block-explorer/instrumentation/metrics"
)

const LabelType = "type"

var (
	ErrorTypeNotFound          = prometheus.Labels{LabelType: "not_found"}
	ErrorTypeOnRecordExport    = prometheus.Labels{LabelType: "record_export"}
	ErrorTypeRateLimitExceeded = prometheus.Labels{LabelType: "rate_limit_exceeded"}

	ExtractProcessCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_extractor_process_count",
		Help: "The number of processes fetching data from heavy",
	})
	FromExtractorDataQueue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_extractor_data_queue",
		Help: "The number of elements in extractor export data queue",
	})
	LastPulseFetched = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_extractor_last_pulse",
		Help: "The number of last pulse fetched by NextFinalizedPulse",
	})
	Errors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gbe_extractor_errors",
		Help: "The number of errors received during data fetching",
	},
		[]string{LabelType},
	)
	ReceivedPulses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_extractor_received_pulses",
		Help: "The number of pulses received",
	})
	ReceivedRecords = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gbe_extractor_received_records",
		Help: "The number of records received",
	})

	RetrievePulsesCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_extractor_retrieve_pulses_count",
		Help: "The number of retrievePulses goroutines",
	})
	RetrieveRecordsCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gbe_extractor_retrieve_records_count",
		Help: "The number of retrievePulses goroutines",
	})
)

type Metrics struct{}

func (s Metrics) Refresh() {
	// nothing to refresh
}

func (s Metrics) Metrics(p *metrics.Prometheus) []prometheus.Collector {
	return []prometheus.Collector{
		ExtractProcessCount,
		FromExtractorDataQueue,
		LastPulseFetched,
		Errors,
		ReceivedRecords,
		ReceivedPulses,
		RetrievePulsesCount,
		RetrieveRecordsCount,
	}
}
