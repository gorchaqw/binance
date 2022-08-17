package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
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
			case "statistics":
				u.statisticsProc()
			case "calc_balance":
				u.calculateBalanceProc()
			}
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
