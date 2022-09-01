package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/mongo"
	"binance/internal/repository/postgres"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
)

type tgmUseCase struct {
	priceUseCase  *priceUseCase
	orderUseCase  *orderUseCase
	settingsRepo  mongo.SettingsRepo
	orderRepo     postgres.OrderRepo
	tgmController *controllers.TgmController
	logger        *logrus.Logger
}

func NewTgmUseCase(
	priceUseCase *priceUseCase,
	orderUseCase *orderUseCase,
	settingsRepo mongo.SettingsRepo,
	orderRepo postgres.OrderRepo,
	tgmController *controllers.TgmController,
	logger *logrus.Logger,
) *tgmUseCase {
	return &tgmUseCase{
		priceUseCase:  priceUseCase,
		orderUseCase:  orderUseCase,
		settingsRepo:  settingsRepo,
		orderRepo:     orderRepo,
		tgmController: tgmController,
		logger:        logger,
	}
}

func (u *tgmUseCase) CommandProcessor() {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		u.logger.WithField("method", "CommandProcessor").Debug(err)
	}

	for update := range u.tgmController.GetUpdates() {
		if u.tgmController.CheckChatID(update.Message.Chat.ID) {

			switch update.Message.Command() {
			case "ping":
				u.pingProc(loc)
			case "stat":
				u.orderStatProc()
			}
		}
	}
}

func (u *tgmUseCase) orderStatProc() {
	msg := "[ Orders Stat ]\n"

	eTime := time.Now()
	sTime := eTime.Add(-24 * time.Hour)

	for _, symbol := range SymbolList {
		orders, err := u.orderRepo.GetLastWithInterval(symbol, sTime, eTime)
		if err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}

		var canceled, filled float64

		for _, order := range orders {
			switch order.Status {
			case "FILLED":
				filled++
			case "CANCELED":
				canceled++
			}
		}

		total := canceled + filled

		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"Total:\t%.0f\n"+
				"Filled:\t%.0f\n"+
				"Canceled:\t%.0f\n"+
				"Filled/Canceled:\t%.0f/%.0f\n",
			symbol,
			total,
			filled,
			canceled,
			filled/total*100,
			canceled/total*100,
		)
	}

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.
			WithError(err).
			Error(string(debug.Stack()))
	}
}

func (u *tgmUseCase) pingProc(loc *time.Location) {
	if err := u.tgmController.Send(
		fmt.Sprintf(
			"PONG [ %s ]",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.WithField("method", "pingProc").Debug(err)
	}
}
