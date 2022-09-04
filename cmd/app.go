package main

import (
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"

	tgBotAPI "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/ic2hrmk/promtail"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type App struct {
	Config     *Config
	LogRus     *logrus.Logger
	HTTPClient *http.Client
	TGM        *tgBotAPI.BotAPI
	DB         *sqlx.DB
	Mongo      *mongo.Client
	PromTail   promtail.Client
	Fiber      *fiber.App
	Metrics    *Metrics
}
