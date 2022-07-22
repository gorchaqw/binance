package main

import (
	"github.com/robfig/cron/v3"
)

func (a *App) initCron() {
	a.Cron = cron.New()
}
