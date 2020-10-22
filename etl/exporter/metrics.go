// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter

import (
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/block-explorer/instrumentation/metrics"
)

type GRPCMetrics struct {
	ServerMetrics *grpc_prometheus.ServerMetrics
}

func NewGRPCMetrics() *GRPCMetrics {
	grpc_prometheus.EnableHandlingTimeHistogram()
	return &GRPCMetrics{ServerMetrics: grpc_prometheus.DefaultServerMetrics}
}

func (s *GRPCMetrics) Refresh() {
	// nothing to refresh
}

func (s *GRPCMetrics) Metrics(p *metrics.Prometheus) []prometheus.Collector {
	return []prometheus.Collector{
		s.ServerMetrics,
	}
}
