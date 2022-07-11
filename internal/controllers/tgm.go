package controllers

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type TgmController struct {
	tgmBot *tgbotapi.BotAPI
	chatID int64
}

func NewTgmController(
	tgmBot *tgbotapi.BotAPI,
	chatID int64,
) *TgmController {
	return &TgmController{
		tgmBot: tgmBot,
		chatID: chatID,
	}
}

func (c *TgmController) Send(text string) error {
	msg := tgbotapi.NewMessage(c.chatID, text)

	if _, err := c.tgmBot.Send(msg); err != nil {
		return err
	}

	return nil
}

func (c *TgmController) Update(msgID int, text string) error {
	msg := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    c.chatID,
			MessageID: msgID,
		},
		Text: text,
	}

	if _, err := c.tgmBot.Send(msg); err != nil {
		return err
	}

	return nil
}
