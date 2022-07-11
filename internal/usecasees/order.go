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

	msgID_BTCRUB  = 41277
	msgID_ETHRUB  = 41275
	msgID_BTCBUSD = 41275
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
	var quantity string

	var sTime time.Time
	var delta float64

	switch symbol {
	case ETHRUB:
		quantity = "0.03"
		delta = 250
	case BTCRUB:
		quantity = "0.002"
		delta = 5000
	case BTCBUSD:
		quantity = "0.002"
		delta = 50
	}

	sTime = time.Now()
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

				average := (max + min) / 2
				deltaMaxMin := (max - min) * 0.2

				actualPrice, err := u.priceUseCase.GetPrice(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				deltaMax := max - actualPrice
				deltaMin := actualPrice - min

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
				//			"STime:\t%s\n"+
				//			"DeltaMaxMin:\t%.2f\n",
				//		symbol,
				//		max,
				//		min,
				//		max-min,
				//		actualPrice,
				//		lastOrder.Price,
				//		lastOrder.Side,
				//		lastOrder.CreatedAt.Format(time.RFC822),
				//		max-actualPrice,
				//		min-actualPrice,
				//		sTime.Format(time.RFC822),
				//		deltaMaxMin,
				//	)); err != nil {
				//	u.logger.Debug(err)
				//	continue
				//}

				var side, orderType string

				switch lastOrder.Side {
				case "SELL":
					if deltaMax >= deltaMax*0.25 &&
						deltaMaxMin != 0 &&
						lastOrder.Price-actualPrice > delta {
						side = "BUY" // купить
						orderType = "MARKET"
					} else {
						continue
					}
					sTime = minT
				case "BUY":
					if deltaMin >= deltaMin*0.25 &&
						deltaMaxMin != 0 &&
						actualPrice-lastOrder.Price > delta {
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
				}, quantity, orderType); err != nil {
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
							"Average:\t%.2f\n"+
							"DeltaMaxMin:\t%.2f\n",

						side,
						symbol,
						actualPrice,
						lastOrder.Price,
						average,
						deltaMaxMin,
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

func (u *orderUseCase) GetOrder(order *structs.Order, quantity, orderType string) error {
	qu, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		return err
	}

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
	q.Set("quantity", fmt.Sprintf("%.5f", qu))
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
