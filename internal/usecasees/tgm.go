package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/postgres"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type tgmUseCase struct {
	priceUseCase  *priceUseCase
	orderUseCase  *orderUseCase
	tgmController *controllers.TgmController
	orderRepo     *postgres.OrderRepository
	logger        *logrus.Logger
}

func NewTgmUseCase(
	priceUseCase *priceUseCase,
	orderUseCase *orderUseCase,
	tgmController *controllers.TgmController,
	orderRepo *postgres.OrderRepository,
	logger *logrus.Logger,
) *tgmUseCase {
	return &tgmUseCase{
		priceUseCase:  priceUseCase,
		orderUseCase:  orderUseCase,
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
			case "set_actual":
				u.setActualProc(loc)
			case "set_avg_price":
				u.setAvgProc(loc)
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

func (u *tgmUseCase) setAvgProc(loc *time.Location) {

	for _, symbol := range SymbolList {
		stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
		if err != nil {
			u.logger.WithField("method", "setAvgProc").Debug(err)
		}

		weightedAvgPrice, err := strconv.ParseFloat(stat.WeightedAvgPrice, 64)
		if err != nil {
			u.logger.WithField("method", "setAvgProc").Debug(err)
		}

		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.WithField("method", "setAvgProc").Debug(err)
		}

		if err := u.orderRepo.SetActualPrice(order.ID, weightedAvgPrice); err != nil {
			u.logger.WithField("method", "setAvgProc").Debug(err)
		}
	}

	if err := u.tgmController.Send(
		fmt.Sprintf(
			"[ Orders updated ]\n"+
				"Time:\t%s\n",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.WithField("method", "setAvgProc").Debug(err)
	}
}

func (u *tgmUseCase) setActualProc(loc *time.Location) {
	for _, symbol := range SymbolList {
		actualPrice, err := u.priceUseCase.GetPrice(symbol)
		if err != nil {
			u.logger.WithField("method", "setActualProc").Debug(err)
		}

		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.WithField("method", "setActualProc").Debug(err)
		}

		if err := u.orderRepo.SetActualPrice(order.ID, actualPrice); err != nil {
			u.logger.WithField("method", "setActualProc").Debug(err)
		}
	}

	if err := u.tgmController.Send(
		fmt.Sprintf(
			"[ Orders updated ]\n"+
				"Time:\t%s\n",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.WithField("method", "setActualProc").Debug(err)
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
