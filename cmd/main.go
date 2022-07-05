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

	clientController := controllers.NewClientController(
		app.HTTPClient,
		app.Config.BinanceApiKey,
		app.Logger,
	)
	cryptoController := controllers.NewCryptoController(
		app.Config.BinanceSecretKey,
	)
	tgmController := controllers.NewTgmController(
		app.TGM,
		chatId,
	)

	priceUseCase := usecasees.NewPriceUseCase(
		clientController,
		tgmController,
		priceRepo,
		app.Config.BinanceUrl,
		app.Logger,
	)

	orderUseCase := usecasees.NewOrderUseCase(
		clientController,
		cryptoController,
		tgmController,
		orderRepo,
		priceRepo,
		priceUseCase,
		app.Config.BinanceUrl,
		app.Logger,
	)

	for _, symbol := range []string{
		usecasees.BTCBUSD,
		usecasees.ETHBUSD,
		usecasees.BUSDRUB,
		usecasees.BNBBUSD,
	} {
		if err := orderUseCase.Monitoring(symbol); err != nil {
			app.Logger.Error(err)
		}

		if err := priceUseCase.Monitoring(symbol); err != nil {
			app.Logger.Error(err)
		}
	}

	//if err := priceUseCase.Monitoring(usecasees.ETHRUB); err != nil {
	//	app.Logger.Error(err)
	//}
	//
	//if err := priceUseCase.Monitoring(usecasees.BTCRUB); err != nil {
	//	app.Logger.Error(err)
	//}

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
