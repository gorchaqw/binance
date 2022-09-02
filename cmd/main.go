package main

import (
	"binance/internal/controllers"
	"binance/internal/repository/mongo"
	"binance/internal/repository/postgres"
	"flag"
	"fmt"
	"strconv"

	"binance/internal/usecasees"
)

func main() {
	var app App
	var confFileName string

	flag.StringVar(&confFileName, "config", ".env", "")
	flag.Parse()

	if err := app.loadConfig(confFileName); err != nil {
		panic(err)
	}

	app.initLogger()

	app.initLoki()

	if err := app.initMongo(); err != nil {
		panic(err)
	}

	if err := app.initTgBot(); err != nil {
		panic(err)
	}

	if err := app.InitDB(app.Config.DB); err != nil {
		panic(err)
	}

	app.initFiber()
	app.initHTTPClient()

	chatId, err := strconv.ParseInt(app.Config.TelegramChatID, 10, 64)
	if err != nil {
		panic(err)
	}

	// Init Repository
	priceRepo := postgres.NewPriceRepository(app.DB)
	orderRepo := postgres.NewOrderRepository(app.DB)

	mongoRepo := mongo.NewSettingsRepository(app.Mongo)

	if err := mongoRepo.SetDefault(); err != nil {
		panic(err)
	}

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
		mongoRepo,
		orderRepo,
		priceUseCase,
		app.Config.BinanceUrl,
		app.Logger,
		app.PromTail,
	)
	tgmUseCase := usecasees.NewTgmUseCase(
		priceUseCase,
		orderUseCase,
		mongoRepo,
		orderRepo,
		tgmController,
		app.Logger,
	)

	go tgmUseCase.CommandProcessor()

	for _, symbol := range usecasees.SymbolList {
		if err := orderUseCase.Monitoring(symbol); err != nil {
			app.Logger.Error(err)
		}
	}

	if err := tgmController.Send(fmt.Sprintf("[ Started ]")); err != nil {
		app.Logger.Error(err)
	}

	app.registerHTTPEndpoints()

	if err := app.Fiber.Listen(fmt.Sprintf(":%s", app.Config.AppPort)); err != nil {
		panic(err)
	}
}
