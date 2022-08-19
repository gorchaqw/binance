package main

import (
	tgmBotAPI "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (a *App) initTgBot() error {
	bot, err := tgmBotAPI.NewBotAPI(a.Config.TelegramApiToken)
	if err != nil {
		return err
	}
	bot.Debug = false

	a.TGM = bot

	return nil
}
