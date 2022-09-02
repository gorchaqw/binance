package main

import (
	"github.com/ic2hrmk/promtail"
)

func (a *App) initLoki() error {
	identifiers := map[string]string{
		"instanceId": a.Name,
	}

	promTail, err := promtail.NewJSONv1Client("loki:3100", identifiers)
	if err != nil {
		return err
	}

	a.PromTail = promTail

	return nil
}
