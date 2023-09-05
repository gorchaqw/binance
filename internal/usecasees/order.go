package usecasees

import (
	"errors"
	"github.com/sirupsen/logrus"

	"binance/internal/controllers"
	"binance/internal/repository/mongo"
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/repository/postgres"
	"binance/internal/usecasees/structs"
)

const (
	orderUrlPath     = "/api/v3/order"
	orderList        = "/api/v3/orderList"
	orderAllUrlPath  = "/api/v3/allOrders"
	orderOpenUrlPath = "/api/v3/openOrders"
	orderOCO         = "/api/v3/order/oco"

	// featureURL = "https://fapi.binance.com"
	// featureURL          = "https://testnet.binancefuture.com"
	featureOrder        = "/fapi/v1/order"
	featurePositionInfo = "/fapi/v2/positionRisk"
	featureBatchOrders  = "/fapi/v1/batchOrders"
	featureSymbolPrice  = "/fapi/v1/ticker/price"
	featureTicker24hr   = "/fapi/v1/ticker/24hr"
	featureDepth        = "/fapi/v1/depth"
	featureTrades       = "/fapi/v1/trades"

	BNB  = "BNB"
	BTC  = "BTC"
	ETH  = "ETH"
	RUB  = "RUB"
	BUSD = "BUSD"
	USDT = "USDT"
	BCH  = "BCH"
	SOL  = "SOL"

	ETHRUB = ETH + RUB

	ETHBUSD = ETH + BUSD
	BTCBUSD = BTC + BUSD
	BNBBUSD = BNB + BUSD
	SOLBUSD = SOL + BUSD

	BTCUSDT = BTC + USDT
	ETHUSDT = ETH + USDT
	BCHUSDT = BCH + USDT

	SideSell = "SELL"
	SideBuy  = "BUY"

	OrderStatusNew        = "NEW"
	OrderStatusCanceled   = "CANCELED"
	OrderStatusFilled     = "FILLED"
	OrderStatusExpired    = "EXPIRED"
	OrderStatusNotFound   = "NOT_FOUND"
	OrderStatusInProgress = "IN PROGRESS"
	OrderStatusError      = "ERROR"

	OrderTypeLimit = "LIMIT"
	//OrderTypeMarket     = "MARKET"

	OrderTypeCurrentTakeProfit = OrderTypeTakeProfitLimit
	OrderTypeCurrentStopLoss   = OrderTypeStopLossLimit

	OrderTypeMarket           = "MARKET"
	OrderTypeTakeProfitMarket = "TAKE_PROFIT_MARKET"
	OrderTypeStopLossMarket   = "STOP_MARKET"

	OrderTypeTakeProfitLimit = "TAKE_PROFIT"
	OrderTypeStopLossLimit   = "STOP"

	OrderTypeDelta = "DELTA"
	OrderTypeOCO   = "OCO"
	OrderTypeBatch = "BATCH"

	OrderTypeLimitID      = 0
	OrderTypeTakeProfitID = 1
	OrderTypeStopLossID   = 2
	OrderTypeDeltaID      = 3
)

var (
	SymbolList = []string{
		//BTCBUSD,
		//SOLBUSD,
		//ETHBUSD,
		//BNBBUSD,
		BTCUSDT,
		//ETHUSDT,
	}
)

type orderUseCase struct {
	clientController controllers.ClientCtrl
	cryptoController controllers.CryptoCtrl
	tgmController    controllers.TgmCtrl

	settingsRepo mongo.SettingsRepo
	orderRepo    postgres.OrderRepo

	priceUseCase *priceUseCase

	url string

	logRus *logrus.Logger
}

func NewOrderUseCase(
	client controllers.ClientCtrl,
	crypto controllers.CryptoCtrl,
	tgm controllers.TgmCtrl,
	settingsRepo mongo.SettingsRepo,
	orderRepo postgres.OrderRepo,
	priceUseCase *priceUseCase,
	url string,
	logger *logrus.Logger,
) *orderUseCase {
	return &orderUseCase{
		clientController: client,
		cryptoController: crypto,
		tgmController:    tgm,
		settingsRepo:     settingsRepo,
		orderRepo:        orderRepo,
		priceUseCase:     priceUseCase,
		url:              url,
		logRus:           logger,
	}
}

func (u *orderUseCase) fillPricePlan(orderType string, symbol string, actualPrice float64, settings *mongoStructs.Settings, status *structs.Status, actualDepth *structs.DepthInfo, actualTreads *structs.TradeInfo) (*structs.PricePlan, error) {
	var out structs.PricePlan

	out.Status = status
	out.Symbol = symbol
	out.SafeDelta = float64(settings.Delta)
	out.TriggerDelta = float64(settings.Delta) / 100 * 1

	switch orderType {
	case OrderTypeLimit:
		if actualDepth.DeltaBids > settings.DepthLimit {

			out.ActualPrice = actualPrice
			out.Side = SideSell
			out.PositionSide = "SHORT"
			out.Price = actualPrice - (out.TriggerDelta / 2)

			//out.ActualPrice = actualPrice
			//out.Side = SideBuy
			//out.PositionSide = "LONG"
			//out.Price = actualPrice + (out.TriggerDelta / 2)

			status.SetBottomLevel(actualPrice)

			return &out, nil
		}

		if actualDepth.DeltaAsks > settings.DepthLimit {
			status.SetTopLevel(actualPrice)

			//out.ActualPrice = actualPrice
			//out.Side = SideSell
			//out.PositionSide = "SHORT"
			//out.Price = actualPrice - (out.TriggerDelta / 2)
			//
			//return &out, nil
		}
	}

	//switch orderType {
	//case OrderTypeLimit:
	//	stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
	//	if err != nil {
	//		u.logRus.Error(err)
	//
	//		return nil
	//	}
	//
	//	highPrice, err := strconv.ParseFloat(stat.HighPrice, 64)
	//	if err != nil {
	//		u.logRus.Error(err)
	//
	//		return nil
	//	}
	//
	//	lowPrice, err := strconv.ParseFloat(stat.LowPrice, 64)
	//	if err != nil {
	//		u.logRus.Error(err)
	//
	//		return nil
	//	}
	//
	//	out.AvgPrice = (highPrice + lowPrice) / 2
	//	out.DeltaPrice = out.AvgPrice / 100 * 0.2
	//
	//	out.AvgPriceHigh = (highPrice + out.AvgPrice) / 2
	//	out.AvgPriceLow = (lowPrice + out.AvgPrice) / 2
	//
	//	out.HighPrice = out.AvgPrice + out.DeltaPrice
	//	out.LowPrice = out.AvgPrice - out.DeltaPrice
	//
	//	out.ActualPricePercent = out.ActualPrice / 100 * (float64(settings.Delta) * 1.2)
	//	out.ActualStopPricePercent = out.ActualPrice / 100 * (float64(settings.Delta) * 1.2)
	//
	//case OrderTypeOCO:
	//	out.ActualPricePercent = out.ActualPrice / 100 * float64(settings.Delta)
	//	out.ActualStopPricePercent = out.ActualPrice / 100 * float64(settings.Delta)
	//case OrderTypeBatch:
	//	out.ActualPricePercent = float64(settings.Delta)
	//	out.ActualStopPricePercent = float64(settings.Delta)
	//}

	//out.ActualPricePercent = out.ActualPrice / 100 * (settings.Delta + (settings.DeltaStep * float64(orderTry)))

	//out.StopPriceBUY = out.ActualPrice + out.ActualStopPricePercent
	//out.StopPriceSELL = out.ActualPrice - out.ActualStopPricePercent
	//
	//out.HighPrice = out.ActualPrice - out.ActualPricePercent
	//out.LowPrice = out.ActualPrice + out.ActualPricePercent

	return &out, errors.New("test fpp")
}
