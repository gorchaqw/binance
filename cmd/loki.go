package main

import (
	"fmt"
	"time"

	"github.com/afiskon/promtail-client/promtail"
)

func (a *App) initLoki() error {
	labels := "123"

	conf := promtail.ClientConfig{
		PushURL:            fmt.Sprintf("http://%s:3100/api/prom/push", "binance-loki"),
		Labels:             labels,
		BatchWait:          5 * time.Second,
		BatchEntriesNumber: 10000,
		SendLevel:          promtail.INFO,
		PrintLevel:         promtail.ERROR,
	}

	loki, err := promtail.NewClientProto(conf)
	if err != nil {
		return err
	}

	a.Loki = loki

	return nil
}
