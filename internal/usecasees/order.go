package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees/structs"
	"binance/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const (
	orderUrlPath     = "/api/v3/order"
	orderAllUrlPath  = "/api/v3/allOrders"
	orderOpenUrlPath = "/api/v3/openOrders"

	ETH  = "ETH"
	RUB  = "RUB"
	BUSD = "BUSD"
	USDT = "USDT"
	BTC  = "BTC"
	BNB  = "BNB"

	ETHRUB  = "ETHRUB"
	ETHBUSD = "ETHBUSD"
	ETHUSDT = "ETHUSDT"

	BTCRUB  = "BTCRUB"
	BTCBUSD = "BTCBUSD"
	BTCUSDT = "BTCUSDT"

	BNBBUSD = "BNBBUSD"

	SIDE_SELL = "SELL"
	SIDE_BUY  = "BUY"

	TIME_FRAME    = 15 * time.Minute
	CRON_JOB      = "0,15,30,45 * * * *"
	TEST_CRON_JOB = "*/1 * * * *"
)

var (
	Symbols = map[string][]string{
		ETHRUB:  {ETH, RUB},
		ETHBUSD: {ETH, BUSD},
		ETHUSDT: {ETH, USDT},

		BTCRUB:  {BTC, RUB},
		BTCBUSD: {BTC, BUSD},
		BTCUSDT: {BTC, USDT},

		BNBBUSD: {BNB, BUSD},
	}

	SymbolList = []string{
		ETHRUB,
		//ETHBUSD,
		//ETHUSDT,
		//
		//BTCRUB,
		//BTCBUSD,
		//BTCUSDT,
		//
		//BNBBUSD,
	}

	SpotURLs = map[string]string{
		ETHRUB:  "https://www.binance.com/ru/trade/ETH_RUB?theme=dark&type=spot",
		ETHBUSD: "https://www.binance.com/ru/trade/ETH_BUSD?theme=dark&type=spot",
		ETHUSDT: "https://www.binance.com/ru/trade/ETH_USDT?theme=dark&type=spot",

		BTCRUB:  "https://www.binance.com/ru/trade/BTC_RUB?theme=dark&type=spot",
		BTCBUSD: "https://www.binance.com/ru/trade/BTC_BUSD?theme=dark&type=spot",
		BTCUSDT: "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",

		BNBBUSD: "https://www.binance.com/ru/trade/BNB_BUSD?theme=dark&type=spot",
	}

	QuantityList = map[string]float64{
		ETHRUB:  0.02,
		ETHBUSD: 0.02,
		ETHUSDT: 0.02,

		BTCRUB:  0.002,
		BTCBUSD: 0.002,
		BTCUSDT: 0.002,

		BNBBUSD: 0.3,
	}
)

type orderUseCase struct {
	clientController *controllers.ClientController
	cryptoController *controllers.CryptoController
	tgmController    *controllers.TgmController

	orderRepo  *sqlite.OrderRepository
	priceRepo  *sqlite.PriceRepository
	candleRepo *sqlite.CandleRepository

	priceUseCase *priceUseCase

	cron *cron.Cron

	url string

	logger *logrus.Logger
}

func NewOrderUseCase(
	client *controllers.ClientController,
	crypto *controllers.CryptoController,
	tgmController *controllers.TgmController,
	orderRepo *sqlite.OrderRepository,
	priceRepo *sqlite.PriceRepository,
	candleRepo *sqlite.CandleRepository,
	priceUseCase *priceUseCase,
	cron *cron.Cron,
	url string,
	logger *logrus.Logger,
) *orderUseCase {
	return &orderUseCase{
		clientController: client,
		cryptoController: crypto,
		tgmController:    tgmController,
		orderRepo:        orderRepo,
		priceRepo:        priceRepo,
		candleRepo:       candleRepo,
		priceUseCase:     priceUseCase,
		cron:             cron,
		url:              url,
		logger:           logger,
	}
}

func (u *orderUseCase) Monitoring(symbol string) error {
	sTime := time.Now()

	if _, err := u.cron.AddFunc(CRON_JOB, func() {
		eTime := time.Now()

		candle, err := u.candleRepo.GetLast(symbol)
		if err != nil {
			u.logger.
				WithField("func", "GetLast").
				WithField("useCase", "order").
				WithField("method", "Monitoring").
				Debug(err)
		}

		if err := u.tgmController.Send(
			fmt.Sprintf(
				"[ Сandle ]\n"+
					"ID:\t%d\n"+
					"Symbol:\t%s\n"+
					"MaxPrice:\t%.2f\n"+
					"MinPrice:\t%.2f\n"+
					"OpenPrice:\t%.2f\n"+
					"ClosePrice:\t%.2f\n"+
					"AvgPrice:\t%.2f\n"+
					"OpenTime:\t%s\n"+
					"CloseTime:\t%s\n"+
					"Upper Shadow Weight:\t%.2f\n"+
					"Body Weight:\t%.2f\n"+
					"Lower Shadow Weight:\t%.2f\n"+
					"Upper Shadow Weight Percent:\t%.2f\n"+
					"Body Weight Percent:\t%.2f\n"+
					"Lower Shadow Weight Percent:\t%.2f\n\n",
				candle.ID,
				candle.Symbol,
				candle.MaxPrice,
				candle.MinPrice,
				candle.OpenPrice,
				candle.ClosePrice,
				(candle.MaxPrice+candle.MinPrice)/2,
				candle.OpenTime.Format(time.RFC822),
				candle.CloseTime.Format(time.RFC822),
				candle.UpperShadow().Weight,
				candle.Body().Weight,
				candle.LowerShadow().Weight,
				candle.UpperShadow().WeightPercent,
				candle.Body().WeightPercent,
				candle.LowerShadow().WeightPercent,
			)); err != nil {
			u.logger.
				WithField("func", "tgmController.Send").
				WithField("useCase", "order").
				WithField("method", "Monitoring").
				Debug(err)
		}

		lastOrder, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			if err == sql.ErrNoRows {
				if err := u.initSymbol(symbol); err != nil {
					u.logger.
						WithField("func", "initSymbol").
						WithField("useCase", "order").
						WithField("method", "Monitoring").
						Debug(err)
				}
			} else {
				u.logger.
					WithField("func", "orderRepo.GetLast").
					WithField("useCase", "order").
					WithField("method", "Monitoring").
					Debug(err)
			}
		}

		//avr := (candle.MaxPrice + candle.MinPrice) / 2
		//delta := avr / 100 * 0.15

		var side string

		switch lastOrder.Side {
		case "SELL":
			//if lastOrder.Price-candle.ClosePrice > delta && structs.BUYPatterns(candle) {
			if structs.BUYPatterns(candle) {
				side = SIDE_BUY
				if err := u.GetOrder(&structs.Order{
					Symbol: symbol,
					Side:   side,
				}, QuantityList[symbol], "MARKET"); err != nil {
					u.logger.
						WithField("func", "u.GetOrder").
						WithField("useCase", "order").
						WithField("method", "Monitoring").
						Debug(err)
				}
			}
		case "BUY":
			//if candle.ClosePrice-lastOrder.Price > delta && structs.SELLPatterns(candle) {
			if structs.SELLPatterns(candle) {
				side = SIDE_SELL
				if err := u.GetOrder(&structs.Order{
					Symbol: symbol,
					Side:   side,
				}, QuantityList[symbol], "MARKET"); err != nil {
					u.logger.
						WithField("func", "u.GetOrder").
						WithField("useCase", "order").
						WithField("method", "Monitoring").
						Debug(err)
				}
			}
		}

		max, min, err := u.priceRepo.GetMaxMinByCreatedByInterval(symbol, sTime, eTime)
		if err != nil {
			u.logger.
				WithField("func", "priceRepo.GetMaxMinByCreatedByInterval").
				WithField("useCase", "order").
				WithField("method", "Monitoring").
				Debug(err)
		}

		lastPrice, err := u.priceRepo.GetLast(symbol)
		if err != nil {
			u.logger.
				WithField("func", "priceRepo.GetLast").
				WithField("useCase", "order").
				WithField("method", "Monitoring").
				Debug(err)
		}

		if err := u.candleRepo.Store(&models.Candle{
			Symbol:     symbol,
			OpenPrice:  candle.ClosePrice,
			ClosePrice: lastPrice.Price,
			MaxPrice:   max,
			MinPrice:   min,
			TimeFrame:  TIME_FRAME.String(),
			OpenTime:   sTime,
			CloseTime:  time.Now(),
		}); err != nil {
			u.logger.
				WithField("func", "candleRepo.Store").
				WithField("useCase", "order").
				WithField("method", "Monitoring").
				Debug(err)
		}

		sTime = time.Now()

	}); err != nil {
		u.logger.
			WithField("func", "c.AddFunc").
			WithField("useCase", "order").
			WithField("method", "Monitoring").
			Debug(err)
	}

	return nil
}

func (u *orderUseCase) initSymbol(symbol string) error {
	stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
	if err != nil {
		return err
	}

	weightedAvgPrice, err := strconv.ParseFloat(stat.WeightedAvgPrice, 64)
	if err != nil {
		return err
	}

	if err := u.orderRepo.Store(&models.Order{
		OrderId:  777,
		Symbol:   symbol,
		Side:     SIDE_BUY,
		Quantity: fmt.Sprintf("%.5f", QuantityList[symbol]),
		Price:    weightedAvgPrice,
	}); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) GetOpenOrders(symbol string) ([]structs.Order, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(orderOpenUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out []structs.Order

	fmt.Printf("%s", req)

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (u *orderUseCase) GetAllOrders(symbol string) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderAllUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return err
	}

	type reqJson struct {
		Symbol              string `json:"symbol"`
		OrderId             int64  `json:"orderId"`
		OrderListId         int    `json:"orderListId"`
		ClientOrderId       string `json:"clientOrderId"`
		Price               string `json:"price"`
		OrigQty             string `json:"origQty"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
		Status              string `json:"status"`
		TimeInForce         string `json:"timeInForce"`
		Type                string `json:"type"`
		Side                string `json:"side"`
		StopPrice           string `json:"stopPrice"`
		IcebergQty          string `json:"icebergQty"`
		Time                int64  `json:"time"`
		UpdateTime          int64  `json:"updateTime"`
		IsWorking           bool   `json:"isWorking"`
		OrigQuoteOrderQty   string `json:"origQuoteOrderQty"`
	}

	var out []reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return err
	}

	for _, order := range out {
		if order.Status == "NEW" {
			if err := u.tgmController.Send(fmt.Sprintf("[ Open Orders ]\n%s\n%s\n%s\n%d", order.Symbol, order.Side, order.Price, order.OrderId)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *orderUseCase) GetOrder(order *structs.Order, quantity float64, orderType string) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("symbol", order.Symbol)
	q.Set("side", order.Side)
	q.Set("type", orderType)
	//q.Set("type", "LIMIT")
	//q.Set("timeInForce", "GTC")
	q.Set("quantity", fmt.Sprintf("%.5f", quantity))
	//q.Set("price", order.Price)
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	u.logger.WithField("method", "GetOrder").Debugf("%s", req)

	type reqJson struct {
		Symbol              string `json:"symbol"`
		OrderID             int64  `json:"orderId"`
		OrderListID         int    `json:"orderListId"`
		ClientOrderID       string `json:"clientOrderId"`
		TransactTime        int64  `json:"transactTime"`
		Price               string `json:"price"`
		OrigQty             string `json:"origQty"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
		Status              string `json:"status"`
		TimeInForce         string `json:"timeInForce"`
		Type                string `json:"type"`
		Side                string `json:"side"`
		Fills               []struct {
			Price           string `json:"price"`
			Qty             string `json:"qty"`
			Commission      string `json:"commission"`
			CommissionAsset string `json:"commissionAsset"`
			TradeID         int    `json:"tradeId"`
		} `json:"fills"`
	}

	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return err
	}

	if out.OrderID != 0 {
		price, err := strconv.ParseFloat(out.Fills[0].Price, 64)
		if err != nil {
			return err
		}

		if err := u.orderRepo.Store(&models.Order{
			OrderId:  out.OrderID,
			Symbol:   out.Symbol,
			Side:     out.Side,
			Quantity: out.Fills[0].Qty,
			Price:    price,
		}); err != nil {
			return err
		}

		if err := u.tgmController.Send(fmt.Sprintf("%s", req)); err != nil {
			return err
		}

		return nil
	}

	var errStruct struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(req, &errStruct); err != nil {
		return err
	}

	if err := u.tgmController.Send(fmt.Sprintf("[ Get Order ]\nError\n%s", errStruct.Msg)); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) Cancel(symbol, orderId string) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", orderId)
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d", time.Now().Unix()*1000))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodDelete, baseURL, nil, true)
	if err != nil {
		return err
	}

	type reqJson struct {
		Symbol              string `json:"symbol"`
		OrigClientOrderId   string `json:"origClientOrderId"`
		OrderId             int    `json:"orderId"`
		OrderListId         int    `json:"orderListId"`
		ClientOrderId       string `json:"clientOrderId"`
		Price               string `json:"price"`
		OrigQty             string `json:"origQty"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
		Status              string `json:"status"`
		TimeInForce         string `json:"timeInForce"`
		Type                string `json:"type"`
		Side                string `json:"side"`
	}

	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return err
	}

	if out.OrderId != 0 {
		if err := u.tgmController.Send(fmt.Sprintf("%s", req)); err != nil {
			return err
		}

		return nil
	}

	var errStruct struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(req, &errStruct); err != nil {
		return err
	}

	if err := u.tgmController.Send(fmt.Sprintf("[ Cancel Order ]\nError\n%s", errStruct.Msg)); err != nil {
		return err
	}

	return nil
}
