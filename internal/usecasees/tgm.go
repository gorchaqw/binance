package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type tgmUseCase struct {
	tgmController *controllers.TgmController
	orderRepo     *sqlite.OrderRepository
	logger        *logrus.Logger
}

func NewTgmUseCase(
	tgmController *controllers.TgmController,
	orderRepo *sqlite.OrderRepository,
	logger *logrus.Logger,
) *tgmUseCase {
	return &tgmUseCase{
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
		case "ping":
			if err := u.tgmController.Send(
				fmt.Sprintf(
					"PONG [ %s ]",
					time.Now().In(loc).Format(time.RFC822),
				)); err != nil {
				u.logger.Debug(err)
				continue
			}
		case "quantity":
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
		case "last":
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
	}
}
