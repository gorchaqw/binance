package controllers

import tgmBotAPI "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type TgmController struct {
	tgmBot *tgmBotAPI.BotAPI
	chatID int64
}

func NewTgmController(
	tgmBot *tgmBotAPI.BotAPI,
	chatID int64,
) *TgmController {
	return &TgmController{
		tgmBot: tgmBot,
		chatID: chatID,
	}
}

func (c *TgmController) Send(text string) error {
	msg := tgmBotAPI.NewMessage(c.chatID, text)
	msg.DisableWebPagePreview = true

	if _, err := c.tgmBot.Send(msg); err != nil {
		return err
	}

	return nil
}

func (c *TgmController) CheckChatID(chatID int64) bool {
	if chatID == c.chatID {
		return true
	}
	return false
}

func (c *TgmController) Update(msgID int, text string) error {
	msg := tgmBotAPI.EditMessageTextConfig{
		BaseEdit: tgmBotAPI.BaseEdit{
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

func (c *TgmController) GetUpdates() tgmBotAPI.UpdatesChannel {
	u := tgmBotAPI.NewUpdate(0)
	u.Timeout = 60

	return c.tgmBot.GetUpdatesChan(u)
}
