package main

import (
	"binance/internal/usecasees"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const OrderComplete = "order_complete"

type Metrics struct {
	Order map[string]prometheus.Counter
}

func (a *App) InitMetrics() {
	metrics := Metrics{Order: map[string]prometheus.Counter{}}

	metrics.Order[usecasees.MetricOrderComplete] = promauto.NewCounter(prometheus.CounterOpts{
		Name: OrderComplete,
		Help: OrderComplete,
	})

	a.Metrics = &metrics
}
