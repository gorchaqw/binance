package main

import (
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/afiskon/promtail-client/promtail"
	tgBotAPI "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type App struct {
	Config     *Config
	Logger     *logrus.Logger
	HTTPClient *http.Client
	TGM        *tgBotAPI.BotAPI
	DB         *sqlx.DB
	Mongo      *mongo.Client
	Loki       promtail.Client
	Fiber      *fiber.App
}
