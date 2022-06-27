package main

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees"
	"strconv"
	"sync"
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
	orderRepo := sqlite.NewOrderRepository(app.DB)

	clientController := controllers.NewClientController(app.HTTPClient, app.Config.BinanceApiKey)
	cryptoController := controllers.NewCryptoController(app.Config.BinanceSecretKey)
	tgmController := controllers.NewTgmController(app.TGM, chatId)

	orderUseCase := usecasees.NewOrderUseCase(
		clientController,
		cryptoController,
		tgmController,
		orderRepo,
		priceRepo,
		app.Config.BinanceUrl,
		app.Logger,
	)

	priceUseCase := usecasees.NewPriceUseCase(
		clientController,
		tgmController,
		priceRepo,
		app.Config.BinanceUrl,
		app.Logger,
	)

	//walletUseCase := usecasees.NewWalletUseCase(clientController, cryptoController, tgmController, app.Config.BinanceUrl, app.Logger)

	if err := orderUseCase.Monitoring(); err != nil {
		app.Logger.Error(err)
	}

	//if err := orderUseCase.GetOrder(&structs.Order{
	//	Symbol: "BTCBUSD",
	//	Side:   "SELL",
	//	Price:  "21300",
	//}, "0.003"); err != nil {
	//	app.Logger.Error(err)
	//}

	//if _, err := orderUseCase.GetOpenOrders(); err != nil {
	//	app.Logger.Error(err)
	//}

	if err := orderUseCase.Monitoring(); err != nil {
		app.Logger.Error(err)
	}

	if err := priceUseCase.Monitoring(); err != nil {
		app.Logger.Error(err)
	}

	//if err := priceUseCase.GetAverage(); err != nil {
	//	app.Logger.Error(err)
	//}

	//if err := walletUseCase.GetAllCoins(); err != nil {
	//	app.Logger.Debug(err)
	//}

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
