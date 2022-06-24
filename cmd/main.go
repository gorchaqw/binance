package main

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees"
	"strconv"
)

func main() {
	var app App

	app.initLogger()

	if err := app.loadConfig(); err != nil {
		panic(err)
	}

	if err := app.initTgBot(); err != nil {
		panic(err)
	}

	if err := app.InitDB(); err != nil {
		panic(err)
	}

	app.initHTTPClient()

	chatId, err := strconv.ParseInt(app.Config.TelegramChatID, 10, 64)
	if err != nil {
		panic(err)
	}

	priceRepo := sqlite.NewPriceRepository(app.DB)

	clientController := controllers.NewClientController(app.HTTPClient, app.Config.BinanceApiKey)
	cryptoController := controllers.NewCryptoController(app.Config.BinanceSecretKey)
	tgmController := controllers.NewTgmController(app.TGM, chatId)

	orderUseCase := usecasees.NewOrderUseCase(clientController, cryptoController, tgmController, app.Config.BinanceUrl, app.Logger)

	if err := orderUseCase.GetOrder(); err != nil {
		app.Logger.Debug(err)
	}

	priceUseCase := usecasees.NewPriceUseCase(clientController, tgmController, priceRepo, app.Config.BinanceUrl, app.Logger)

	if err := priceUseCase.GetPrice(); err != nil {
		app.Logger.Debug(err)
	}

}
