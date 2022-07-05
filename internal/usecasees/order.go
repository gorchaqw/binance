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

	BTCBUSD = "BTCBUSD"
	BTCRUB  = "BTCRUB"

	ETHRUB  = "ETHRUB"
	ETHBUSD = "ETHBUSD"

	BUSDRUB = "BUSDRUB"
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
	var delta float64

	switch symbol {
	case BTCBUSD:
		quantity = "0.0005"
		delta = 30
	case BTCRUB:
		quantity = "0.004"
		delta = 500
	case ETHRUB:
		quantity = "0.03"
		delta = 500
	}

	ticker := time.NewTicker(5 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				max, min, err := u.priceRepo.GetMaxMinByCreatedByInterval(symbol, lastOrder.CreatedAt, time.Now())
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				actualPrice, err := u.priceUseCase.GetPrice(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				if err := u.tgmController.Send(
					fmt.Sprintf(
						"[ Monitoring ]\n"+
							"Symbol:\t%s\n"+
							"Max Price:\t%.0f\n"+
							"Min Price:\t%.0f\n"+
							"Delta Max/Min:\t%.0f\n"+
							"Actual Price:\t%.0f\n"+
							"Order Price:\t%.0f\n"+
							"Order Side:\t%s\n"+
							"DeltaMax:\t%.0f\n"+
							"DeltaMin:\t%.0f\n",
						symbol,
						max,
						min,
						max-min,
						actualPrice,
						lastOrder.Price,
						lastOrder.Side,
						max-actualPrice,
						min-actualPrice,
					)); err != nil {
					u.logger.Debug(err)
					continue
				}

				orders, err := u.GetOpenOrders(symbol)
				if err != nil {
					u.logger.Debug(err)
					continue
				}

				if len(orders) == 0 {
					lastOrder, err := u.orderRepo.GetLast(symbol)
					if err != nil {
						u.logger.Debug(err)
						continue
					}

					if err := u.tgmController.Send(fmt.Sprintf("[ Order Monitoring, Last Order ]\n%+v", lastOrder)); err != nil {
						u.logger.Debug(err)
						continue
					}

					var side, orderType string

					switch lastOrder.Side {
					case "SELL":
						if min-actualPrice > -delta && actualPrice > lastOrder.Price {
							side = "BUY"
							orderType = "MARKET"
						} else {
							continue
						}
					case "BUY":
						if max-actualPrice > delta && actualPrice < lastOrder.Price {
							side = "SELL"
							orderType = "MARKET"
						} else {
							continue
						}
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
							"[ New Orders ]\nSide:\t%s\nSymbol:\t%s\nPrice:\t%.0f\n",
							side,
							symbol,
							actualPrice,
						)); err != nil {
						u.logger.Debug(err)
						continue
					}
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

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (u *orderUseCase) GetAllOrders() error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderAllUrlPath)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
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
	q.Set("timestamp", fmt.Sprintf("%d", time.Now().Unix()*1000))

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

	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return err
	}

	if out.OrderId != 0 {
		price, err := strconv.ParseFloat(out.Price, 64)
		if err != nil {
			return err
		}

		if err := u.orderRepo.Store(&models.Order{
			OrderId:  out.OrderId,
			Symbol:   out.Symbol,
			Side:     out.Side,
			Quantity: fmt.Sprintf("%.5f", qu),
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
