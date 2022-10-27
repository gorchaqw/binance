package main

import (
	"net/http"
	"time"
)

func (a *App) initHTTPClient() {
	a.HTTPClient = &http.Client{
		Timeout: 1 * time.Second,
	}
}
