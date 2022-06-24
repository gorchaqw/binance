package main

import (
	"github.com/sirupsen/logrus"
)

func (a *App) initLogger() {
	a.Logger = logrus.New()
	a.Logger.SetLevel(logrus.DebugLevel)
}
