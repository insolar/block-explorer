// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"github.com/insolar/block-explorer/instrumentation/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	quntitile = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}

	SaveJetDropDataDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_SaveJetDropDataDuration",
		Help:       "The duration of the SaveJetDropData function execution",
		Objectives: quntitile,
	})
	SavePulseDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_SavePulseDuration",
		Help:       "The duration of the SavePulse function execution",
		Objectives: quntitile,
	})
	SavePulseExecutionDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_SavePulseExecutionDuration",
		Help:       "The duration of the SavePulse function execution without mutexes",
		Objectives: quntitile,
	})
	CompletePulseDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_CompletePulseDuration",
		Help:       "The duration of the CompletePulse function execution",
		Objectives: quntitile,
	})
	SequencePulseDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_SequencePulseDuration",
		Help:       "The duration of the SequencePulse function execution",
		Objectives: quntitile,
	})
	GetRecordDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetRecordDuration",
		Help:       "The duration of the GetRecord function execution",
		Objectives: quntitile,
	})
	GetLifelineDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetLifelineDuration",
		Help:       "The duration of the GetLifeline function execution",
		Objectives: quntitile,
	})
	GetPulseDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetPulseDuration",
		Help:       "The duration of the GetPulse function execution",
		Objectives: quntitile,
	})
	GetPulsesDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetPulsesDuration",
		Help:       "The duration of the GetPulses function execution",
		Objectives: quntitile,
	})
	GetRecordsByJetDropDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetRecordsByJetDropDuration",
		Help:       "The duration of the GetRecordsByJetDrop function execution",
		Objectives: quntitile,
	})
	GetIncompletePulsesDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetIncompletePulsesDuration",
		Help:       "The duration of the GetIncompletePulses function execution",
		Objectives: quntitile,
	})
	GetPulseByPrevDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetPulseByPrevDuration",
		Help:       "The duration of the GetPulseByPrev function execution",
		Objectives: quntitile,
	})
	GetSequentialPulseDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetSequentialPulseDuration",
		Help:       "The duration of the GetSequentialPulse function execution",
		Objectives: quntitile,
	})
	GetNextSavedPulseDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetNextSavedPulseDuration",
		Help:       "The duration of the GetNextSavedPulse function execution",
		Objectives: quntitile,
	})
	GetJetDropsDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetJetDropsDuration",
		Help:       "The duration of the GetJetDrops function execution",
		Objectives: quntitile,
	})
	GetJetDropsWithParamsDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetJetDropsWithParamsDuration",
		Help:       "The duration of the GetJetDropsWithParams function execution",
		Objectives: quntitile,
	})
	GetJetDropByIDDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetJetDropByIDDuration",
		Help:       "The duration of the GetJetDropByID function execution",
		Objectives: quntitile,
	})
	GetJetDropsByJetIDDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "gbe_storage_stats_GetJetDropsByJetIDDuration",
		Help:       "The duration of the GetJetDropsByJetID function execution",
		Objectives: quntitile,
	})
)

// The storage function metrics
type Metrics struct{}

func (s Metrics) Refresh() {
	// nothing to refresh
}

func (s Metrics) Metrics(p *metrics.Prometheus) []prometheus.Collector {
	return []prometheus.Collector{
		SaveJetDropDataDuration,
		SavePulseDuration,
		SavePulseExecutionDuration,
		CompletePulseDuration,
		SequencePulseDuration,
		GetRecordDuration,
		GetLifelineDuration,
		GetPulseDuration,
		GetPulsesDuration,
		GetRecordsByJetDropDuration,
		GetIncompletePulsesDuration,
		GetPulseByPrevDuration,
		GetSequentialPulseDuration,
		GetNextSavedPulseDuration,
		GetJetDropsDuration,
		GetJetDropsWithParamsDuration,
		GetJetDropByIDDuration,
		GetJetDropsByJetIDDuration,
	}
}
