package main

import (
	"binance/internal/usecasees"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	Order map[string]prometheus.Counter
}

func (a *App) InitMetrics() {
	metrics := Metrics{Order: map[string]prometheus.Counter{}}

	metrics.Order[usecasees.MetricOrderComplete] = promauto.NewCounter(prometheus.CounterOpts{
		Name: usecasees.MetricOrderComplete,
		Help: usecasees.MetricOrderComplete,
	})

	metrics.Order[usecasees.StopLossLimitFilled] = promauto.NewCounter(prometheus.CounterOpts{
		Name: usecasees.StopLossLimitFilled,
		Help: usecasees.StopLossLimitFilled,
	})

	metrics.Order[usecasees.LimitMaker] = promauto.NewCounter(prometheus.CounterOpts{
		Name: usecasees.LimitMaker,
		Help: usecasees.LimitMaker,
	})

	a.Metrics = &metrics
}
