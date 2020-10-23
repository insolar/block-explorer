// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultRefreshInterval = time.Second * 15 // the prometheus default pull metrics every 15 seconds
	defaultHTTPServerPort  = 8080             // default pull port
)

// Collector represents the methods for collecting metrics
type Collector interface {
	Refresher
	Metrics(*Prometheus) []prometheus.Collector
}

// Refresher represents the methods for refreshing metrics
type Refresher interface {
	Refresh()
}

type Prometheus struct {
	*Config
	Logger      log.Logger
	refreshOnce sync.Once
	Labels      map[string]string
	collectors  []prometheus.Collector
}

type Config struct {
	DBName            string        // use DBName as metrics label
	RefreshInterval   time.Duration // refresh metrics interval.
	StartServer       bool          // if true, create http server to expose metrics
	HTTPServerPort    uint32        // http server port
	MetricsCollectors []Collector   // collector
}

func New(config Config) *Prometheus {
	if config.RefreshInterval == 0 {
		config.RefreshInterval = defaultRefreshInterval
	}

	if config.HTTPServerPort == 0 {
		config.HTTPServerPort = defaultHTTPServerPort
	}

	logger := belogger.FromContext(context.Background()).WithField("gorm:prometheus", nil)
	return &Prometheus{Config: &config, Logger: logger, Labels: make(map[string]string)}
}

func (p *Prometheus) Initialize() error { // can be called repeatedly
	if p.Config.DBName != "" {
		p.Labels["db_name"] = p.Config.DBName
	}

	p.refreshOnce.Do(func() {
		for _, mc := range p.MetricsCollectors {
			p.collectors = append(p.collectors, mc.Metrics(p)...)
		}

		prometheus.MustRegister(p.collectors...)

		go func() {
			for range time.Tick(p.Config.RefreshInterval) {
				for _, v := range p.MetricsCollectors {
					go v.Refresh()
				}
			}
		}()
	})

	if p.Config.StartServer {
		go p.startServer()
	}

	return nil
}

var httpServerOnce sync.Once

func (p *Prometheus) startServer() {
	// only start once
	httpServerOnce.Do(func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(fmt.Sprintf(":%d", p.Config.HTTPServerPort), mux)
		if err != nil {
			p.Logger.Error("gorm:prometheus listen and serve err: ", err)
		}
	})
}
