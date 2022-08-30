package controllers

import (
	"net/url"

	tgmBotAPI "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate mockery --case=snake --name=ClientCtrl
//go:generate mockery --case=snake --name=CryptoCtrl
//go:generate mockery --case=snake --name=TgmCtrl

type ClientCtrl interface {
	Send(method string, url *url.URL, body []byte, useApiKey bool) ([]byte, error)
}

type CryptoCtrl interface {
	GetSignature(query string) string
}

type TgmCtrl interface {
	Send(text string) error
	CheckChatID(chatID int64) bool
	Update(msgID int, text string) error
	GetUpdates() tgmBotAPI.UpdatesChannel
}
