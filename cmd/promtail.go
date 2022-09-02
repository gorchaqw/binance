package main

import (
	"github.com/ic2hrmk/promtail"
)

func (a *App) initPromTail() error {
	identifiers := map[string]string{
		"instanceId": a.Config.AppName,
	}

	promTail, err := promtail.NewJSONv1Client("loki:3100", identifiers)
	if err != nil {
		return err
	}

	a.PromTail = promTail

	return nil
}
