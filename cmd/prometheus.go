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

	for _, m := range []structs.MetricConst{
		structs.MetricOrderComplete,
		structs.MetricOrderStopLossLimitFilled,
		structs.MetricOrderLimitMaker,
		structs.MetricOrderOSONewPricePlan,
		structs.MetricOrderLimitNewPricePlan,
	} {
		metrics.Order[m] = promauto.NewCounter(prometheus.CounterOpts{
			Name: m.ToString(),
			Help: m.ToString(),
		})
	}

	a.Metrics = &metrics
}
