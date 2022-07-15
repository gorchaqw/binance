package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type tgmUseCase struct {
	priceUseCase  *priceUseCase
	tgmController *controllers.TgmController
	orderRepo     *sqlite.OrderRepository
	logger        *logrus.Logger
}

func NewTgmUseCase(
	priceUseCase *priceUseCase,
	tgmController *controllers.TgmController,
	orderRepo *sqlite.OrderRepository,
	logger *logrus.Logger,
) *tgmUseCase {
	return &tgmUseCase{
		priceUseCase:  priceUseCase,
		orderRepo:     orderRepo,
		tgmController: tgmController,
		logger:        logger,
	}
}

func (u *tgmUseCase) CommandProcessor() {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		u.logger.Debug(err)
	}

	for update := range u.tgmController.GetUpdates() {
		switch update.Message.Command() {
		case "set_actual":
			u.setActualProc()
		case "ping":
			u.pingProc(loc)
		case "quantity":
			u.quantityProc()
		case "last":
			u.lastProc(loc)
		}
	}
}

func (u *tgmUseCase) setActualProc() {
	for _, symbol := range SymbolList {
		actualPrice, err := u.priceUseCase.GetPrice(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		if err := u.orderRepo.SetActualPrice(order.ID, actualPrice); err != nil {
			u.logger.Debug(err)
		}
	}
}

func (u *tgmUseCase) pingProc(loc *time.Location) {
	if err := u.tgmController.Send(
		fmt.Sprintf(
			"PONG [ %s ]",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.Debug(err)
	}
}

func (u *tgmUseCase) quantityProc() {
	for symbol, quantity := range QuantityList {
		if err := u.tgmController.Send(
			fmt.Sprintf(
				"Symbol:\t%s\n"+
					"Quantity:\t%.5f\n",
				symbol,
				quantity,
			)); err != nil {
			u.logger.Debug(err)
			continue
		}
	}
}

func (u *tgmUseCase) lastProc(loc *time.Location) {
	for _, symbol := range SymbolList {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		if err := u.tgmController.Send(
			fmt.Sprintf(
				"Symbol:\t%s\n"+
					"Order Price:\t%.2f\n"+
					"Order Side:\t%s\n"+
					"URL:\t%s\n"+
					"Order CreatedAt:\t%s\n",
				symbol,
				order.Price,
				order.Side,
				SpotURLs[symbol],
				order.CreatedAt.In(loc).Format(time.RFC822),
			)); err != nil {
			u.logger.Debug(err)
			continue
		}
	}
}
