package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func RegisterHTTPEndpoints(f *fiber.App, l *logrus.Logger) {
	h := NewHandler(f, l)
	router := f.Group("api")
	router.Get("/healthcheck", h.HealthCheck)
}
