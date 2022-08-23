package main

import "binance/internal/api/http"

func (a *App) registerHTTPEndpoints() {
	http.RegisterHTTPEndpoints(a.Fiber, a.Logger)
}
