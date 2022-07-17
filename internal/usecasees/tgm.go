package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees/structs"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

type tgmUseCase struct {
	priceUseCase  *priceUseCase
	orderUseCase  *orderUseCase
	tgmController *controllers.TgmController
	orderRepo     *sqlite.OrderRepository
	logger        *logrus.Logger
}

func NewTgmUseCase(
	priceUseCase *priceUseCase,
	orderUseCase *orderUseCase,
	tgmController *controllers.TgmController,
	orderRepo *sqlite.OrderRepository,
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
		u.logger.Debug(err)
	}

	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
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
			case "quantity":
				u.quantityProc()
			case "last":
				u.lastProc(loc)
			case "sell_all":
				u.massOrderProc(SIDE_SELL)
			case "buy_all":
				u.massOrderProc(SIDE_BUY)
			case "order":
				order := strings.Split(update.Message.CommandArguments(), " ")
				if (order[0] == SIDE_BUY || order[0] == SIDE_SELL) && contains(SymbolList, order[1]) {
					u.orderProc(order[0], order[1])
				}
			case "statistics":
				u.statisticsProc()
			}
		}
	}
}

func (u *tgmUseCase) statisticsProc() {
	msg := "[ Statistics ]\n\n"

	for _, symbol := range SymbolList {
		stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		weightedAvgPrice, err := strconv.ParseFloat(stat.WeightedAvgPrice, 64)
		if err != nil {
			u.logger.Debug(err)
		}

		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"AvgPrice:\t%.5f\n"+
				"Delta:\t%.5f\n"+
				"PriceChange:\t%s\n"+
				"PriceChangePercent:\t%s\n\n",
			symbol,
			weightedAvgPrice,
			weightedAvgPrice*DeltaRatio,
			stat.PriceChange,
			stat.PriceChangePercent,
		)
	}

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.Debug(err)
	}
}

func (u *tgmUseCase) massOrderProc(side string) {
	for _, symbol := range SymbolList {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		if order.Side != side {
			if err := u.orderUseCase.GetOrder(&structs.Order{
				Symbol: symbol,
				Side:   side,
			}, QuantityList[symbol], "MARKET"); err != nil {
				u.logger.Debug(err)
				continue
			}
		}
	}
}

func (u *tgmUseCase) setAvgProc(loc *time.Location) {

	for _, symbol := range SymbolList {
		stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		weightedAvgPrice, err := strconv.ParseFloat(stat.WeightedAvgPrice, 64)
		if err != nil {
			u.logger.Debug(err)
		}

		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		if err := u.orderRepo.SetActualPrice(order.ID, weightedAvgPrice); err != nil {
			u.logger.Debug(err)
		}
	}

	if err := u.tgmController.Send(
		fmt.Sprintf(
			"Orders updated [ %s ]",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.Debug(err)
	}
}

func (u *tgmUseCase) setActualProc(loc *time.Location) {
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

	if err := u.tgmController.Send(
		fmt.Sprintf(
			"Orders updated [ %s ]",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.Debug(err)
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
	var msg string
	for symbol, quantity := range QuantityList {
		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"Raio:\t%.5f\n"+
				"Quantity:\t%.5f\n\n",
			symbol,
			DeltaRatios[symbol],
			quantity,
		)
	}

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.Debug(err)
	}
}

func (u *tgmUseCase) orderProc(side, symbol string) {
	order, err := u.orderRepo.GetLast(symbol)
	if err != nil {
		u.logger.Debug(err)
	}

	if order.Side != side {
		if err := u.orderUseCase.GetOrder(&structs.Order{
			Symbol: symbol,
			Side:   side,
		}, QuantityList[symbol], "MARKET"); err != nil {
			u.logger.Debug(err)
		}
	}
}

func (u *tgmUseCase) lastProc(loc *time.Location) {
	var msg string
	for _, symbol := range SymbolList {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"Order Price:\t%.2f\n"+
				"Order Side:\t%s\n"+
				"URL:\t%s\n"+
				"Order CreatedAt:\t%s\n\n",
			symbol,
			order.Price,
			order.Side,
			SpotURLs[symbol],
			order.CreatedAt.In(loc).Format(time.RFC822),
		)
	}

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.Debug(err)
	}

}
