package main

import (
	"binance/internal/usecasees/structs"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	Order map[structs.MetricConst]prometheus.Counter
}

func (a *App) InitMetrics() {
	metrics := Metrics{Order: map[structs.MetricConst]prometheus.Counter{}}

	metrics.Order[structs.MetricOrderComplete] = promauto.NewCounter(prometheus.CounterOpts{
		Name: structs.MetricOrderComplete.ToString(),
		Help: structs.MetricOrderComplete.ToString(),
	})

	metrics.Order[structs.MetricOrderStopLossLimitFilled] = promauto.NewCounter(prometheus.CounterOpts{
		Name: structs.MetricOrderStopLossLimitFilled.ToString(),
		Help: structs.MetricOrderStopLossLimitFilled.ToString(),
	})

	metrics.Order[structs.MetricOrderLimitMaker] = promauto.NewCounter(prometheus.CounterOpts{
		Name: structs.MetricOrderLimitMaker.ToString(),
		Help: structs.MetricOrderLimitMaker.ToString(),
	})

	a.Metrics = &metrics
}
