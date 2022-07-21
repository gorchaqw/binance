package main

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees"
	"flag"
	"fmt"
	"strconv"
	"sync"
)

func main() {
	var app App
	var confFileName, dbFileName string

	flag.StringVar(&confFileName, "config", ".env", "")
	flag.StringVar(&dbFileName, "db", "./store.db", "")
	flag.Parse()

	app.initLogger()

	if err := app.loadConfig(confFileName); err != nil {
		panic(err)
	}

	if err := app.initTgBot(); err != nil {
		panic(err)
	}

	if err := app.InitDB(dbFileName); err != nil {
		panic(err)
	}

	app.initHTTPClient()

	chatId, err := strconv.ParseInt(app.Config.TelegramChatID, 10, 64)
	if err != nil {
		panic(err)
	}

	// Init Repository
	priceRepo := sqlite.NewPriceRepository(app.DB)
	orderRepo := sqlite.NewOrderRepository(app.DB)
	candleRepo := sqlite.NewCandlesRepository(app.DB)

	// Init Controllers
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

	// Init UseCases
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
		candleRepo,
		priceUseCase,
		app.Config.BinanceUrl,
		app.Logger,
	)
	tgmUseCase := usecasees.NewTgmUseCase(
		priceUseCase,
		orderUseCase,
		tgmController,
		orderRepo,
		app.Logger,
	)

	//go orderUseCase.UpdateRatio()
	go tgmUseCase.CommandProcessor()

	for _, symbol := range usecasees.SymbolList {
		if err := orderUseCase.Monitoring(symbol); err != nil {
			app.Logger.Error(err)
		}

		if err := priceUseCase.Monitoring(symbol); err != nil {
			app.Logger.Error(err)
		}
	}

	if err := tgmController.Send(fmt.Sprintf("[ Started ]")); err != nil {
		app.Logger.Error(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
