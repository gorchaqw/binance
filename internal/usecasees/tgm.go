package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees/structs"
	"binance/models"
	"fmt"
	"github.com/sirupsen/logrus"
	"runtime/debug"
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
		u.logger.WithField("method", "CommandProcessor").Debug(err)
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
			case "last":
				u.lastProc(loc)
			case "sell_all":
				u.massOrderProc(SIDE_SELL, loc)
			case "buy_all":
				u.massOrderProc(SIDE_BUY, loc)
			case "order":
				order := strings.Split(update.Message.CommandArguments(), " ")
				if (order[0] == SIDE_BUY || order[0] == SIDE_SELL) && contains(SymbolList, order[1]) {
					u.orderProc(order[0], order[1], loc)
				}
			case "statistics":
				u.statisticsProc()
			case "calc_balance":
				u.calculateBalanceProc()
			case "open_orders":
				u.openOrdersProc()

			}
		}
	}
}

func (u *tgmUseCase) openOrdersProc() {
	for _, symbol := range SymbolList {
		openOrders, err := u.orderUseCase.GetOpenOrders(symbol)
		if err != nil {
			u.logger.WithField("method", "openOrdersProc").Debug(err)
		}

		actualPrice, err := u.priceUseCase.GetPrice(symbol)
		if err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}

		price, err := strconv.ParseFloat(openOrders[0].Price, 64)
		if err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}

		actualPricePercent := actualPrice / 100 * 0.75
		actualStopPricePercent := actualPricePercent / 2

		stopPriceBUY := actualPrice + actualStopPricePercent
		stopPriceSELL := actualPrice - actualStopPricePercent

		o := models.Order{
			OrderId:  openOrders[0].OrderId,
			Symbol:   openOrders[0].Symbol,
			Side:     openOrders[0].Side,
			Price:    price,
			Quantity: fmt.Sprintf("%.5f", QuantityList[openOrders[0].Symbol]),
			Status:   sqlite.ORDER_STATUS_NEW,
		}

		switch openOrders[0].Side {
		case SIDE_BUY:
			o.StopPrice = stopPriceBUY
		case SIDE_SELL:
			o.StopPrice = stopPriceSELL
		}

		if err := u.orderRepo.Store(&o); err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}

		if err := u.tgmController.Send(fmt.Sprintf("%+v", openOrders)); err != nil {
			u.logger.WithField("method", "openOrdersProc").Debug(err)
		}
	}
}

func (u *tgmUseCase) calculateBalanceProc() {
	msg := "[ Calculate Balance ]\n\n"
	total := make(map[string]float64)
	var totalInRUB float64

	for _, symbol := range SymbolList {
		actualPrice, err := u.priceUseCase.GetPrice(symbol)
		if err != nil {
			u.logger.WithField("method", "calculateBalanceProc").Debug(err)
		}

		sum := actualPrice * QuantityList[symbol]
		s := Symbols[symbol][1]

		RUBSymbol := fmt.Sprintf("%s%s", Symbols[symbol][0], RUB)
		actualPriceInRUB, err := u.priceUseCase.GetPrice(RUBSymbol)
		if err != nil {
			u.logger.WithField("method", "calculateBalanceProc").Debug(err)
		}

		capacityInRUB := actualPriceInRUB * QuantityList[symbol]

		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"Quantity:\t%.5f\n"+
				"Capacity:\t%.2f %s\n"+
				"Capacity RUB:\t%.2f RUB\n\n",
			symbol,
			QuantityList[symbol],
			sum,
			s,
			capacityInRUB,
		)

		totalInRUB += actualPriceInRUB * QuantityList[symbol]
		total[s] += sum
	}

	msg += fmt.Sprintf(
		"Total in RUB:\t%.2f\n",
		totalInRUB,
	)

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.WithField("method", "calculateBalanceProc").Debug(err)
	}

	msg = "[ Total ]\n\n"
	for symbol, weight := range total {
		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"weight:\t%.2f\n\n",
			symbol,
			weight,
		)
	}

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.WithField("method", "calculateBalanceProc").Debug(err)
	}
}

func (u *tgmUseCase) statisticsProc() {
	msg := "[ Statistics ]\n\n"

	for _, symbol := range SymbolList {
		stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
		if err != nil {
			u.logger.WithField("method", "statisticsProc").Debug(err)
		}

		weightedAvgPrice, err := strconv.ParseFloat(stat.WeightedAvgPrice, 64)
		if err != nil {
			u.logger.WithField("method", "statisticsProc").Debug(err)
		}

		msg += fmt.Sprintf(
			"Symbol:\t%s\n"+
				"Quantity:\t%.5f\n"+
				"AvgPrice:\t%.5f\n"+
				"PriceChange:\t%s\n"+
				"PriceChangePercent:\t%s\n\n",
			symbol,
			QuantityList[symbol],
			weightedAvgPrice,
			stat.PriceChange,
			stat.PriceChangePercent,
		)
	}

	if err := u.tgmController.Send(msg); err != nil {
		u.logger.WithField("method", "statisticsProc").Debug(err)
	}
}

func (u *tgmUseCase) massOrderProc(side string, loc *time.Location) {
	for _, symbol := range SymbolList {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.WithField("method", "massOrderProc").Debug(err)
		}

		if order.Side != side {
			if err := u.orderUseCase.GetOrder(&structs.Order{
				Symbol: symbol,
				Side:   side,
			}, QuantityList[symbol], "MARKET", 1); err != nil {
				u.logger.WithField("method", "massOrderProc").Debug(err)
				continue
			}
		}
	}

	if err := u.tgmController.Send(
		fmt.Sprintf(
			"[ Orders updated ]\n"+
				"Time:\t%s\n",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.WithField("method", "massOrderProc").Debug(err)
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

func (u *tgmUseCase) orderProc(side, symbol string, loc *time.Location) {
	order, err := u.orderRepo.GetLast(symbol)
	if err != nil {
		u.logger.WithField("method", "orderProc").Debug(err)
	}

	if order.Side != side {
		if err := u.orderUseCase.GetOrder(&structs.Order{
			Symbol: symbol,
			Side:   side,
		}, QuantityList[symbol], "MARKET", 1); err != nil {
			u.logger.WithField("method", "orderProc").Debug(err)
		}
	}

	if err := u.tgmController.Send(
		fmt.Sprintf(
			"[ Orders updated ]\n"+
				"Time:\t%s\n",
			time.Now().In(loc).Format(time.RFC822),
		)); err != nil {
		u.logger.WithField("method", "orderProc").Debug(err)
	}
}

func (u *tgmUseCase) lastProc(loc *time.Location) {
	var msg string
	for _, symbol := range SymbolList {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			u.logger.WithField("method", "lastProc").Debug(err)
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
		u.logger.WithField("method", "lastProc").Debug(err)
	}

}
