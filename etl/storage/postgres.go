// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/insolar/block-explorer/instrumentation/metrics"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/client_golang/prometheus"
)

type PostgresCollector struct {
	*PostgresCollectorConfig

	status map[string]prometheus.Gauge
	*gorm.DB
	Logger log.Logger
}

type PostgresCollectorConfig struct {
	Prefix   string
	Interval time.Duration
	Labels   map[string]string
}

func NewPostgresCollector(config *PostgresCollectorConfig, db *gorm.DB) *PostgresCollector {
	if config == nil {
		config = &PostgresCollectorConfig{}
	}
	return &PostgresCollector{
		PostgresCollectorConfig: config,
		DB:                      db,
		Logger:                  belogger.FromContext(context.Background()).WithField("postgres_collector", ""),
	}
}

func (pg *PostgresCollector) Metrics(p *metrics.Prometheus) []prometheus.Collector {
	if pg.Prefix == "" {
		pg.Prefix = "gbe_gorm_status_"
	}

	if pg.Interval == 0 {
		pg.Interval = p.RefreshInterval
	}

	if pg.status == nil {
		pg.status = map[string]prometheus.Gauge{}
	}

	go func() {
		for range time.Tick(pg.Interval) {
			pg.collect()
		}
	}()

	pg.collect()
	collectors := make([]prometheus.Collector, 0, len(pg.status))

	for _, v := range pg.status {
		collectors = append(collectors, v)
	}

	return collectors
}

func (pg *PostgresCollector) collect() {
	sql := "select " +
		"(select count(*) from jet_drops) as jetdrops, " +
		"(select count(*)from pulses) as pulses, " +
		"(select count(*) from records) as records"
	rows, err := pg.Raw(sql).Rows()
	if err != nil {
		pg.Logger.Error("gorm:prometheus query error: %v", err)
	}

	if rows == nil {
		return
	}

	var jetdrops, pulses, records int
	for rows.Next() {
		err = rows.Scan(&jetdrops, &pulses, &records)
		if err != nil {
			pg.Logger.Error("gorm scan got error: %v", err)
			continue
		}
		pg.setStats("all_jetdrop_count", jetdrops)
		pg.setStats("all_pulse_count", pulses)
		pg.setStats("all_record_count", records)
	}
}

func (pg *PostgresCollector) setStats(variableName string, variableValue int) {
	gauge, ok := pg.status[variableName]
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        pg.Prefix + variableName,
			ConstLabels: pg.Labels,
		})

		pg.status[variableName] = gauge
		_ = prometheus.Register(gauge)
	}

	gauge.Set(float64(variableValue))
}
