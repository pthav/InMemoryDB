package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type metrics struct {
	dbHttpRequestCounter *prometheus.CounterVec   // Requests labeled by uri, method, and status.
	dbLatency            *prometheus.HistogramVec // Latency labeled by uri, method, and status.
	dbSubscriptions      prometheus.Gauge         // Number of active subscriptions
	dbPublishedMessages  prometheus.Counter       // Number of cumulative published messages.
}

func newPromHandler() (http.Handler, *metrics) {
	m := &metrics{
		dbHttpRequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_http_requests_total",
			Help: "Total number of DB http requests, labelled by uri, method, and status.",
		}, []string{"method", "uri", "status"}),
		dbLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "db_latency",
			Help:    "Histogram of DB latency in seconds, labelled by uri, method, and status.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "uri", "status"}),
		dbSubscriptions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_subscriptions",
			Help: "Total number of subscriptions",
		}),
		dbPublishedMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_published_messages",
			Help: "Cumulative number of published messages",
		}),
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(m.dbHttpRequestCounter)
	reg.MustRegister(m.dbLatency)
	reg.MustRegister(m.dbSubscriptions)
	reg.MustRegister(m.dbPublishedMessages)

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	return handler, m
}
