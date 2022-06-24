package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"

	"net/http"
)

type App struct {
	Config     *Config
	Logger     *logrus.Logger
	HTTPClient *http.Client
	TGM        *tgbotapi.BotAPI
}
