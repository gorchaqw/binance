package main

import (
	"net/http"
	"time"
)

func (a *App) initHTTPClient() {
	a.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
}
