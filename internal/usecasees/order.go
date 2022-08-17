package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees/structs"
	"binance/models"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	orderUrlPath     = "/api/v3/order"
	orderList        = "/api/v3/orderList"
	orderAllUrlPath  = "/api/v3/allOrders"
	orderOpenUrlPath = "/api/v3/openOrders"
	orderOCO         = "/api/v3/order/oco"

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
)

var (
	TimeFrames = map[string]time.Duration{
		"15min": 15 * time.Minute,
		"1h":    1 * time.Hour,
		"1min":  time.Minute,
	}

	CRONJobs = map[string]string{
		"15min": "0,15,30,45 * * * *",
		"1h":    "0 * * * *",
		"4h":    "0 0,4,8,12,16,20 * * *",
		"1min":  "* * * * *",
	}

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
		//ETHRUB,
		//ETHBUSD,
		//ETHUSDT,
		//
		//BTCRUB,
		BTCBUSD,
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

	StepList = map[string]float64{
		BTCBUSD: 0.0005,
		ETHBUSD: 8,
	}

	QuantityList = map[string]float64{
		//ETHRUB: 0.25,
		//ETHBUSD: 0.02,
		//ETHUSDT: 0.02,
		//
		//BTCRUB:  0.002,
		BTCBUSD: 0.014,
		ETHBUSD: 0.1,
		//
		//BNBBUSD: 0.3,
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
	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)

	quantity := StepList[symbol]
	orderTry := 1

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				orderList, err := u.GetOrderList(lastOrder.OrderId)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				lastOrderStatus := "NEW"
				for _, o := range orderList.Orders {
					orderInfo, err := u.GetOrderInfo(o.OrderID, symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					switch true {
					case orderInfo.Type == "STOP_LOSS_LIMIT" && orderInfo.Status == "FILLED":
						lastOrderStatus = "CANCELED"

						if orderInfo.Side == SIDE_SELL {
							orderTry++
							quantity = StepList[symbol] * float64(orderTry) * 2
						}

					case orderInfo.Type == "LIMIT_MAKER" && orderInfo.Status == "FILLED":
						lastOrderStatus = "FILLED"

						if orderInfo.Side == SIDE_SELL {
							orderTry = 1
							quantity = StepList[symbol]
						}
					}

				}

				if lastOrder.Status != lastOrderStatus {
					if err := u.orderRepo.SetStatus(lastOrder.ID, lastOrderStatus); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

				openOrders, err := u.GetOpenOrders(symbol)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				type sendStatStruct struct {
					Side          string
					Quantity      float64
					ActualPrice   float64
					StopPriceBUY  float64
					StopPriceSELL float64
					PriceBUY      float64
					PriceSELL     float64
					OpenOrders    []structs.OrderList
					LastOrder     *models.Order
				}

				sendStat := func(stat *sendStatStruct) {
					if err := u.tgmController.Send(fmt.Sprintf("[ Stat ]\n"+
						"side:\t%s\n"+
						"quantity:\t%.5f\n"+
						"actualPrice:\t%.2f\n"+
						"stopPriceBUY:\t%.2f\n"+
						"stopPriceSELL:\t%.2f\n"+
						"priceBUY:\t%.2f\n"+
						"priceSELL:\t%.2f\n"+
						"openOrders:\t%+v\n"+
						"lastOrder:\t%+v\n",
						stat.Side,
						stat.Quantity,
						stat.ActualPrice,
						stat.StopPriceBUY,
						stat.StopPriceSELL,
						stat.PriceBUY,
						stat.PriceSELL,
						stat.OpenOrders,
						stat.LastOrder)); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

				actualPrice, err := u.priceUseCase.GetPrice(symbol)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				actualPricePercent := float64(10)
				actualStopPricePercent := actualPricePercent

				stopPriceBUY := actualPrice + actualStopPricePercent
				stopPriceSELL := actualPrice - actualStopPricePercent

				priceBUY := actualPrice - actualPricePercent
				priceSELL := actualPrice + actualPricePercent

				if len(openOrders) == 0 {
					switch lastOrder.Side {
					case SIDE_BUY:
						if err := u.GetOrder(&structs.Order{
							Symbol:    symbol,
							Side:      SIDE_SELL,
							Price:     fmt.Sprintf("%.0f", priceSELL),
							StopPrice: fmt.Sprintf("%.0f", stopPriceSELL),
						}, quantity, orderTry); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}

						go sendStat(&sendStatStruct{
							Side:          SIDE_SELL,
							Quantity:      quantity,
							ActualPrice:   actualPrice,
							StopPriceBUY:  stopPriceBUY,
							StopPriceSELL: stopPriceSELL,
							PriceBUY:      priceBUY,
							PriceSELL:     priceSELL,
							OpenOrders:    openOrders,
							LastOrder:     lastOrder,
						})

					case SIDE_SELL:
						if err := u.GetOrder(&structs.Order{
							Symbol:    symbol,
							Side:      SIDE_BUY,
							Price:     fmt.Sprintf("%.0f", priceBUY),
							StopPrice: fmt.Sprintf("%.0f", stopPriceBUY),
						}, quantity, orderTry); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}

						go sendStat(&sendStatStruct{
							Side:          SIDE_BUY,
							Quantity:      quantity,
							ActualPrice:   actualPrice,
							StopPriceBUY:  stopPriceBUY,
							StopPriceSELL: stopPriceSELL,
							PriceBUY:      priceBUY,
							PriceSELL:     priceSELL,
							OpenOrders:    openOrders,
							LastOrder:     lastOrder,
						})
					}
				}

				//switch lastOrder.Side {
				//case SIDE_BUY:
				//	if actualPrice > lastOrder.StopPrice {
				//		if err := cancelOrder(lastOrder, symbol, SIDE_BUY, stopPriceBUY, true); err != nil {
				//			u.logger.
				//				WithError(err).
				//				Error(string(debug.Stack()))
				//			continue
				//		}
				//	}
				//case SIDE_SELL:
				//	if actualPrice < lastOrder.StopPrice {
				//		if err := cancelOrder(lastOrder, symbol, SIDE_SELL, stopPriceSELL, true); err != nil {
				//			u.logger.
				//				WithError(err).
				//				Error(string(debug.Stack()))
				//			continue
				//		}
				//	}
				//}
			}
		}
	}()

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
		Status:   sqlite.ORDER_STATUS_NEW,
		Try:      1,
	}); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) GetOpenOrders(symbol string) ([]structs.OrderList, error) {
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

	var out []structs.OrderList

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
func (u *orderUseCase) GetOrderList(orderListID int64) (*structs.OrderList, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(orderList)

	q := baseURL.Query()
	q.Set("orderListId", fmt.Sprintf("%d", orderListID))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out structs.OrderList

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *orderUseCase) GetOrderInfo(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", fmt.Sprintf("%d", orderID))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *orderUseCase) GetOrder(order *structs.Order, quantity float64, try int) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderOCO)

	q := baseURL.Query()
	q.Set("symbol", order.Symbol)
	q.Set("side", order.Side)
	q.Set("quantity", fmt.Sprintf("%.5f", quantity))
	q.Set("price", order.Price)
	q.Set("stopPrice", order.StopPrice)
	q.Set("stopLimitPrice", order.StopPrice)
	q.Set("recvWindow", "60000")
	q.Set("stopLimitTimeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	type OrderList struct {
		OrderListID       int64  `json:"orderListId"`
		ContingencyType   string `json:"contingencyType"`
		ListStatusType    string `json:"listStatusType"`
		ListOrderStatus   string `json:"listOrderStatus"`
		ListClientOrderID string `json:"listClientOrderId"`
		TransactionTime   int64  `json:"transactionTime"`
		Symbol            string `json:"symbol"`
		Orders            []struct {
			Symbol        string `json:"symbol"`
			OrderID       int64  `json:"orderId"`
			ClientOrderID string `json:"clientOrderId"`
		} `json:"orders"`
		OrderReports []struct {
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
			StopPrice           string `json:"stopPrice,omitempty"`
		} `json:"orderReports"`
	}

	fmt.Printf("%s", req)

	var oList OrderList

	if err := json.Unmarshal(req, &oList); err != nil {
		return err
	}

	stopPrice, err := strconv.ParseFloat(order.StopPrice, 64)
	if err != nil {
		return err
	}

	o := models.Order{
		OrderId:   oList.OrderListID,
		Symbol:    oList.Symbol,
		Side:      order.Side,
		StopPrice: stopPrice,
		Quantity:  fmt.Sprintf("%.5f", quantity),
		Type:      "OCO",
		Status:    "NEW",
		Try:       try,
	}

	if err := u.orderRepo.Store(&o); err != nil {
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
	//q.Set("timeInForce", "GTC")
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
		return nil
	}

	var errStruct struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(req, &errStruct); err != nil {
		return err
	}

	return nil
}
