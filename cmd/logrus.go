package main

import (
	"github.com/sirupsen/logrus"
)

func (a *App) initLogRus() {
	a.LogRus = logrus.New()

	switch a.Config.LogLevel {
	case "DEBUG":
		a.LogRus.SetLevel(logrus.DebugLevel)
	case "ERROR":
		a.LogRus.SetLevel(logrus.ErrorLevel)
	default:
		a.LogRus.SetLevel(logrus.InfoLevel)
	}
}
