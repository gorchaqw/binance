package http

import (
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
)

type Middleware struct {
	appName string
	fiber   *fiber.App
}

func NewMiddleware(fiber *fiber.App) *Middleware {
	return &Middleware{
		fiber: fiber,
	}
}

func (m *Middleware) useMetrics() {
	prometheus := fiberprometheus.New("binance")
	prometheus.RegisterAt(m.fiber, "/metrics")
	m.fiber.Use(prometheus.Middleware)
}
