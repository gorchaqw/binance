package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/afiskon/promtail-client/promtail"

	tgBotAPI "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type App struct {
	Config     *Config
	Logger     *logrus.Logger
	HTTPClient *http.Client
	TGM        *tgBotAPI.BotAPI
	DB         *sqlx.DB
	Loki       *promtail.Client
}

func loki(a *App) error {
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

	a.Loki = &loki

	return nil
}
