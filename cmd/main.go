package main

import (
	"binance/internal/controllers"
	"binance/internal/usecasees"
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

	app.initHTTPClient()

	clientController := controllers.NewClientController(app.HTTPClient, app.Config.BinanceApiKey)
	cryptoController := controllers.NewCryptoController(app.Config.BinanceSecretKey)

	orderUseCase := usecasees.NewOrderUseCase(clientController, cryptoController, app.Logger)

	if err := orderUseCase.GetOrder(); err != nil {
		app.Logger.Debug(err)
	}

	//u := tgbotapi.NewUpdate(0)
	//u.Timeout = 60

	//updates := app.TGM.GetUpdatesChan(u)
	//
	//for update := range updates {
	//	if update.Message != nil {
	//		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	//
	//		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
	//		msg.ReplyToMessageID = update.Message.MessageID
	//
	//		if _, err := app.TGM.Send(msg); err != nil {
	//			app.Logger.Debug(err)
	//		}
	//	}
	//}
}
