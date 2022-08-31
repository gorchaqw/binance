package main

import (
	"github.com/sirupsen/logrus"
)

func (a *App) initLogger() {
	a.Logger = logrus.New()

	switch a.Config.LogLevel {
	case "DEBUG":
		a.Logger.SetLevel(logrus.DebugLevel)
	case "ERROR":
		a.Logger.SetLevel(logrus.ErrorLevel)
	default:
		a.Logger.SetLevel(logrus.InfoLevel)
	}
}
