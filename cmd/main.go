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

	app.initLogRus()

	if err := app.initPromTail(); err != nil {
		panic(err)
	}

	if err := app.initMongo(); err != nil {
		panic(err)
	}

	if err := app.InitDB(app.Config.DB); err != nil {
		panic(err)
	}

	app.initFiber()
	app.InitMetrics()
	app.initHTTPClient()

	chatId, err := strconv.ParseInt(app.Config.TelegramChatID, 10, 64)
	if err != nil {
		panic(err)
	}

	// Init Repository
	priceRepo := postgres.NewPriceRepository(app.DB)
	//orderRepoSpot := postgres.NewOrderRepository(app.DB, postgres.Spot)
	orderRepoFeatures := postgres.NewOrderRepository(app.DB, postgres.Features)

	mongoRepo := mongo.NewSettingsRepository(app.Mongo)

	if err := mongoRepo.SetDefault(); err != nil {
		panic(err)
	}

	// Init Controllers
	clientController := controllers.NewClientController(
		app.HTTPClient,
		app.Config.BinanceApiKey,
		app.LogRus,
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
		app.LogRus,
	)

	//orderUseCaseSpot := usecasees.NewOrderUseCase(
	//	clientController,
	//	cryptoController,
	//	mongoRepo,
	//	orderRepoSpot,
	//	priceUseCase,
	//	app.Config.BinanceUrl,
	//	app.LogRus,
	//	app.PromTail,
	//	app.Metrics.Order,
	//)

	orderUseCaseFeatures := usecasees.NewOrderUseCase(
		clientController,
		cryptoController,
		mongoRepo,
		orderRepoFeatures,
		priceUseCase,
		app.Config.BinanceUrl,
		app.LogRus,
		app.PromTail,
		app.Metrics.Order,
	)

	//tgmUseCase := usecasees.NewTgmUseCase(
	//	priceUseCase,
	//	orderUseCase,
	//	mongoRepo,
	//	orderRepo,
	//	tgmController,
	//	app.LogRus,
	//)

	//go tgmUseCase.CommandProcessor()

	//for _, symbol := range usecasees.SymbolList {
	//	if err := orderUseCase.Monitoring(symbol); err != nil {
	//		app.LogRus.Error(err)
	//	}
	//}

	for _, symbol := range usecasees.SymbolList {
		if err := orderUseCaseFeatures.FeaturesMonitoring(symbol); err != nil {
			app.LogRus.Error(err)
		}
	}

	app.registerHTTPEndpoints()

	if err := app.Fiber.Listen(fmt.Sprintf(":%s", app.Config.AppPort)); err != nil {
		panic(err)
	}
}
