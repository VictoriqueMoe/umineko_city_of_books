package db

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

type (
	statsCollector struct {
		db *sql.DB

		maxOpen           *prometheus.Desc
		open              *prometheus.Desc
		inUse             *prometheus.Desc
		idle              *prometheus.Desc
		waitCount         *prometheus.Desc
		waitDuration      *prometheus.Desc
		maxIdleClosed     *prometheus.Desc
		maxIdleTimeClosed *prometheus.Desc
		maxLifetimeClosed *prometheus.Desc
	}
)

func RegisterStatsCollector(db *sql.DB) {
	_ = prometheus.Register(newStatsCollector(db))
}

func newStatsCollector(db *sql.DB) *statsCollector {
	return &statsCollector{
		db:                db,
		maxOpen:           prometheus.NewDesc("db_pool_max_open_connections", "Maximum number of open connections allowed to the database.", nil, nil),
		open:              prometheus.NewDesc("db_pool_open_connections", "Number of established connections both in use and idle.", nil, nil),
		inUse:             prometheus.NewDesc("db_pool_in_use_connections", "Number of connections currently in use.", nil, nil),
		idle:              prometheus.NewDesc("db_pool_idle_connections", "Number of idle connections in the pool.", nil, nil),
		waitCount:         prometheus.NewDesc("db_pool_wait_count_total", "Total number of connections waited for.", nil, nil),
		waitDuration:      prometheus.NewDesc("db_pool_wait_duration_seconds_total", "Total time blocked waiting for a new connection.", nil, nil),
		maxIdleClosed:     prometheus.NewDesc("db_pool_max_idle_closed_total", "Total number of connections closed due to SetMaxIdleConns.", nil, nil),
		maxIdleTimeClosed: prometheus.NewDesc("db_pool_max_idle_time_closed_total", "Total number of connections closed due to SetConnMaxIdleTime.", nil, nil),
		maxLifetimeClosed: prometheus.NewDesc("db_pool_max_lifetime_closed_total", "Total number of connections closed due to SetConnMaxLifetime.", nil, nil),
	}
}

func (c *statsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpen
	ch <- c.open
	ch <- c.inUse
	ch <- c.idle
	ch <- c.waitCount
	ch <- c.waitDuration
	ch <- c.maxIdleClosed
	ch <- c.maxIdleTimeClosed
	ch <- c.maxLifetimeClosed
}

func (c *statsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(c.maxOpen, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.open, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUse, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.idle, prometheus.GaugeValue, float64(stats.Idle))

	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDuration, prometheus.CounterValue, stats.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.maxIdleClosed, prometheus.CounterValue, float64(stats.MaxIdleClosed))
	ch <- prometheus.MustNewConstMetric(c.maxIdleTimeClosed, prometheus.CounterValue, float64(stats.MaxIdleTimeClosed))
	ch <- prometheus.MustNewConstMetric(c.maxLifetimeClosed, prometheus.CounterValue, float64(stats.MaxLifetimeClosed))
}
