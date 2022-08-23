package main

import "github.com/gofiber/fiber/v2"

func (a *App) initFiber() {
	a.Fiber = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
}
