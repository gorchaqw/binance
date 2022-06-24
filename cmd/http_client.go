package main

import "net/http"

func (a *App) initHTTPClient() {
	a.HTTPClient = &http.Client{}
}
