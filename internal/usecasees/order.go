package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/internal/usecasees/structs"
	"binance/models"
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

	BTCRUB  = "BTCRUB"
	ETHRUB  = "ETHRUB"
	BTCBUSD = "BTCBUSD"
	ETHBUSD = "ETHBUSD"
	USDTRUB = "USDTRUB"
	BTCUSDT = "BTCUSDT"
	BNBRUB  = "BNBRUB"
	LTCRUB  = "LTCRUB"
)

var (
	SymbolList = []string{
		ETHRUB,
		ETHBUSD,
		BTCUSDT,
		BTCBUSD,
		BTCRUB,
		USDTRUB,
		BNBRUB,
		LTCRUB,
	}

	SpotURLs = map[string]string{
		ETHRUB:  "https://www.binance.com/ru/trade/ETH_RUB?theme=dark&type=spot",
		ETHBUSD: "https://www.binance.com/ru/trade/ETH_BUSD?theme=dark&type=spot",
		BTCUSDT: "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",
		BTCBUSD: "https://www.binance.com/ru/trade/BTC_BUSD?theme=dark&type=spot",
		BTCRUB:  "https://www.binance.com/ru/trade/BTC_RUB?theme=dark&type=spot",
		USDTRUB: "https://www.binance.com/ru/trade/USDT_RUB?theme=dark&type=spot",
		BNBRUB:  "https://www.binance.com/ru/trade/BNB_RUB?theme=dark&type=spot",
		LTCRUB:  "https://www.binance.com/ru/trade/LTC_RUB?theme=dark&type=spot",
	}

	QuantityList = map[string]float64{
		ETHRUB:  0.02,
		ETHBUSD: 0.02,
		BTCUSDT: 0.004,
		BTCBUSD: 0.002,
		BTCRUB:  0.002,
		USDTRUB: 60,
		BNBRUB:  0.04,
		LTCRUB:  0.09,
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

func (u *orderUseCase) Monitoring(symbol string) error {
	sTime := time.Now()

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
					u.logger.Debugf("orderRepo.GetLast %+v", err)
					continue
				}

				max, maxT, err := u.priceRepo.GetMaxByCreatedByInterval(symbol, sTime, time.Now())
				if err != nil {
					u.logger.Debugf("priceRepo.GetMaxByCreatedByInterval %+v", err)
					continue
				}

				min, minT, err := u.priceRepo.GetMinByCreatedByInterval(symbol, sTime, time.Now())
				if err != nil {
					u.logger.Debugf("priceRepo.GetMinByCreatedByInterval %+v", err)
					continue
				}

				actualPrice, err := u.priceUseCase.GetPrice(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				deltaMax := max - actualPrice
				deltaMin := actualPrice - min

				avr := (max + min) / 2
				delta := 0.00075 * avr

				avrMAX := max - avr
				avrMAXActual := actualPrice - avr

				avrMIN := avr - min
				avrMINActual := avr - actualPrice

				//deltaMax := 100 * (max - actualPrice) / actualPrice
				//deltaMin := 100 * (actualPrice - min) / actualPrice

				//msgId := 0
				//switch symbol {
				//case ETHRUB:
				//	msgId = msgID_ETHRUB
				//case BTCRUB:
				//	msgId = msgID_BTCRUB
				//case BTCBUSD:
				//	msgId = msgID_BTCBUSD
				//}

				//if err := u.tgmController.Send(
				//	fmt.Sprintf(
				//		"[ Monitoring ]\n"+
				//			"Symbol:\t%s\n"+
				//			"Max Price:\t%.2f\n"+
				//			"Min Price:\t%.2f\n"+
				//			"Delta Max/Min:\t%.2f\n"+
				//			"Actual Price:\t%.2f\n"+
				//			"Order Price:\t%.2f\n"+
				//			"Order Side:\t%s\n"+
				//			"Order Time:\t%s\n"+
				//			"DeltaMax:\t%.2f\n"+
				//			"DeltaMin:\t%.2f\n"+
				//			"STime:\t%s\n",
				//		symbol,
				//		max,
				//		min,
				//		max-min,
				//		actualPrice,
				//		lastOrder.Price,
				//		lastOrder.Side,
				//		lastOrder.CreatedAt.Format(time.RFC822),
				//		100*(max-actualPrice)/actualPrice,
				//		100*(actualPrice-min)/actualPrice,
				//		sTime.Format(time.RFC822),
				//	)); err != nil {
				//	u.logger.Debug(err)
				//	continue
				//}

				var side, orderType string

				switch lastOrder.Side {
				case "SELL":
					if lastOrder.Price-actualPrice > delta &&
						deltaMin > delta {
						side = "BUY" // купить
						orderType = "MARKET"
					} else {
						continue
					}
					sTime = minT
				case "BUY":
					if actualPrice-lastOrder.Price > delta &&
						deltaMax > delta {
						side = "SELL" // продать
						orderType = "MARKET"
					} else {
						continue
					}
					sTime = maxT
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
							"Delta:\t%.2f\n"+
							"Delta MAX:\t%.2f\n"+
							"Delta MIN:\t%.2f\n",
						side,
						symbol,
						actualPrice,
						lastOrder.Price,
						delta,
						100*(avrMAX-avrMAXActual)/avrMAX,
						100*(avrMIN-avrMINActual)/avrMIN,
					)); err != nil {
					u.logger.Debug(err)
					continue
				}
			}
		}
	}()

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
	fmt.Printf("%s\n%s\n%s\n", order.Symbol, order.Side, orderType)

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
