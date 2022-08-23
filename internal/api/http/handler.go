package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	fiber  *fiber.App
	logger *logrus.Logger
}

func NewHandler(f *fiber.App, l *logrus.Logger) *Handler {
	return &Handler{
		fiber:  f,
		logger: l,
	}
}

func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	body := struct {
		Status bool `json:"status"`
	}{
		Status: true,
	}

	if err := c.JSON(body); err != nil {
		return err
	}

	return nil
}
