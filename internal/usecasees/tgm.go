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
	for update := range u.tgmController.GetUpdates() {
		switch update.Message.Command() {
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
						order.CreatedAt.Format(time.RFC822),
					)); err != nil {
					u.logger.Debug(err)
					continue
				}
			}
		}
	}
}
