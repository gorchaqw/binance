package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees/structs"
	"binance/models"
	"database/sql"
	"encoding/json"
	"fmt"
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

	TIME_FRAME = 1 * time.Hour
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
		ETHBUSD,
		ETHUSDT,

		BTCRUB,
		BTCBUSD,
		BTCUSDT,

		BNBBUSD,
	}

	DeltaRatios = map[string]float64{
		ETHRUB:  3,
		ETHBUSD: 3,
		ETHUSDT: 3,

		BTCRUB:  3,
		BTCBUSD: 3,
		BTCUSDT: 3,

		BNBBUSD: 3,
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

	orderRepo *sqlite.OrderRepository
	priceRepo *sqlite.PriceRepository

	priceUseCase *priceUseCase

	url string

	logger *logrus.Logger
}

func NewOrderUseCase(
	client *controllers.ClientController,
	crypto *controllers.CryptoController,
	tgmController *controllers.TgmController,
	orderRepo *sqlite.OrderRepository,
	priceRepo *sqlite.PriceRepository,
	priceUseCase *priceUseCase,
	url string,
	logger *logrus.Logger,
) *orderUseCase {
	return &orderUseCase{
		clientController: client,
		cryptoController: crypto,
		tgmController:    tgmController,
		orderRepo:        orderRepo,
		priceRepo:        priceRepo,
		priceUseCase:     priceUseCase,
		url:              url,
		logger:           logger,
	}
}

func (u *orderUseCase) updateRatio() {
	for _, symbol := range SymbolList {
		stat, err := u.priceUseCase.GetPriceChangeStatistics(symbol)
		if err != nil {
			u.logger.Debug(err)
		}

		priceChangePercent, err := strconv.ParseFloat(stat.PriceChangePercent, 64)
		if err != nil {
			u.logger.Debug(err)
		}

		var ratio float64

		if priceChangePercent < 0 {
			ratio = priceChangePercent * -0.15
		} else {
			ratio = priceChangePercent * 0.15
		}

		DeltaRatios[symbol] = ratio
	}
}

func (u *orderUseCase) Monitoring(symbol string) error {
	lastPrice, err := u.priceUseCase.GetPrice(symbol)
	if err != nil {
		u.logger.Debug(err)
	}

	sTime := time.Now()

	timeFrame := time.NewTicker(TIME_FRAME)
	timeFrameDone := make(chan bool)

	go func() {
		for {
			select {
			case <-timeFrameDone:
				return
			case _ = <-timeFrame.C:
				sTime = time.Now()
				p, err := u.priceUseCase.GetPrice(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				lastPrice = p
			}
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					if err == sql.ErrNoRows {
						if err := u.initSymbol(symbol); err != nil {
							u.logger.Debug(err)
						}
						continue
					} else {
						u.logger.Debug(err)
						continue
					}
				}

				max, _, err := u.priceRepo.GetMaxByCreatedByInterval(symbol, sTime, time.Now())
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				min, _, err := u.priceRepo.GetMinByCreatedByInterval(symbol, sTime, time.Now())
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				actualPrice, err := u.priceUseCase.GetPrice(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				avr := (max + min) / 2
				delta := avr / 100 * 0.2

				avgMAXMIN := max - min

				avgMAX := max - actualPrice
				avgMIN := actualPrice - min

				lastMax := max - lastPrice
				lastMin := lastPrice - min

				if avgMAXMIN == 0 ||
					avgMAX <= 0 ||
					avgMIN <= 0 ||
					lastMax <= 0 ||
					lastMin <= 0 {
					continue
				}

				deltaMIN := avgMIN * 100 / avgMAXMIN
				deltaMAX := avgMAX * 100 / avgMAXMIN

				deltaLastMIN := lastMin * 100 / avgMAXMIN
				deltaLastMAX := lastMax * 100 / avgMAXMIN

				//avrMAX := max - avr
				//avrMAXActual := actualPrice - avr

				//avrMIN := avr - min
				//avrMINActual := avr - actualPrice

				var side, orderType string

				//if err := u.tgmController.Send(
				//	fmt.Sprintf(
				//		"[ Monitoring ]\n"+
				//			"Symbol:\t%s\n"+
				//			"Price:\t%.2f\n"+
				//			"Last order price:\t%.2f\n"+
				//			"Delta price:\t%.2f\n"+
				//			"Delta:\t%.2f\n"+
				//			"Max:\t%.2f\n"+
				//			"Min:\t%.2f\n"+
				//			"AvgMAXMIN:\t%.2f\n"+
				//			"AvgMAX:\t%.2f\n"+
				//			"AvgMIN:\t%.2f\n"+
				//			"Avg:\t%.2f\n"+
				//			"Last MAX:\t%.2f\n"+
				//			"Last MIN:\t%.2f\n"+
				//			"Delta Last MAX:\t%.2f\n"+
				//			"Delta Last MIN:\t%.2f\n"+
				//			"Delta MAX:\t%.2f\n"+
				//			"Delta MIN:\t%.2f\n",
				//		symbol,
				//		actualPrice,
				//		lastOrder.Price,
				//		lastOrder.Price-actualPrice,
				//		delta,
				//		max,
				//		min,
				//		avgMAXMIN,
				//		avgMAX,
				//		avgMIN,
				//		avr,
				//		lastMax,
				//		lastMin,
				//		deltaLastMAX,
				//		deltaLastMIN,
				//		deltaMAX,
				//		deltaMIN,
				//	)); err != nil {
				//	u.logger.Debug(err)
				//	continue
				//}

				switch lastOrder.Side {
				case "SELL":
					if lastOrder.Price-actualPrice > delta &&
						(structs.PatternShootingStar(deltaLastMIN, deltaMAX) ||
							structs.PatternGallows(deltaLastMIN, deltaMAX)) {
						side = SIDE_BUY // купить
						orderType = "MARKET"
					} else {
						continue
					}
				case "BUY":
					if actualPrice-lastOrder.Price > delta &&
						structs.PatternBaseBUY(deltaMAX, deltaLastMAX) {
						side = SIDE_SELL // продать
						orderType = "MARKET"
					} else {
						continue
					}
				}

				if err := u.GetOrder(&structs.Order{
					Symbol: symbol,
					Side:   side,
				}, QuantityList[symbol], orderType); err != nil {
					u.logger.Debug(err)
					continue
				}

				if err := u.tgmController.Send(
					fmt.Sprintf(
						"[ New Orders ]\n"+
							"Side:\t%s\n"+
							"Symbol:\t%s\n"+
							"Price:\t%.2f\n"+
							"Last order price:\t%.2f\n"+
							"Delta price:\t%.2f\n"+
							"Delta:\t%.2f\n"+
							"Delta Last MIN:\t%.2f\n"+
							"Delta Last MAX:\t%.2f\n"+
							"Delta MIN:\t%.2f\n"+
							"Delta MAX:\t%.2f\n",
						side,
						symbol,
						actualPrice,
						lastOrder.Price,
						lastOrder.Price-actualPrice,
						delta,
						deltaLastMIN,
						deltaLastMAX,
						deltaMIN,
						deltaMAX,
					)); err != nil {
					u.logger.Debug(err)
					continue
				}
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

	u.logger.Debugf("%s", req)

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
