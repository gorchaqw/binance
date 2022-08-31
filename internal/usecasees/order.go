package usecasees

import (
	"binance/internal/repository/mongo"
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/repository/postgres"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/google/uuid"

	"binance/internal/controllers"
	"binance/internal/usecasees/structs"
	"binance/models"

	"github.com/sirupsen/logrus"
)

const (
	orderUrlPath     = "/api/v3/order"
	orderList        = "/api/v3/orderList"
	orderAllUrlPath  = "/api/v3/allOrders"
	orderOpenUrlPath = "/api/v3/openOrders"
	orderOCO         = "/api/v3/order/oco"

	BTC  = "BTC"
	ETH  = "ETH"
	RUB  = "RUB"
	BUSD = "BUSD"
	USDT = "USDT"

	ETHRUB  = ETH + RUB
	ETHBUSD = ETH + BUSD
	BTCUSDT = BTC + USDT

	SideSell = "SELL"
	SideBuy  = "BUY"

	OrderStatusNew      = "NEW"
	OrderStatusCanceled = "CANCELED"
	OrderStatusFilled   = "FILLED"
)

var (
	SymbolList = []string{
		BTCUSDT,
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

	logger *logrus.Logger
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
		logger:           logger,
	}
}

func (u *orderUseCase) Monitoring(symbol string) error {
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)

	settings, err := u.settingsRepo.Load(symbol)
	if err != nil {
		return err
	}

	orderTry := 1
	quantity := settings.Step
	sessionID := uuid.New().String()

	sendOrderInfo := func(order *models.Order) {
		if err := u.tgmController.Send(fmt.Sprintf("[ Last Order ]\n"+
			"orderId:\t%d\n"+
			"status:\t%s\n"+
			"side:\t%s\n",
			order.OrderID,
			order.Status,
			order.Side)); err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}
	}
	sendLimit := func(quantity float64) {
		if err := u.tgmController.Send(fmt.Sprintf("[ Limit ]\n"+
			"quantity:\t%.5f\n",
			quantity)); err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}
	}
	sendStat := func(stat *structs.PricePlan) {
		if err := u.tgmController.Send(fmt.Sprintf("[ Stat ]\n"+
			"quantity:\t%.4f\n"+
			"actualPrice:\t%.2f\n"+
			"actualPricePercent:\t%.2f\n"+
			"stopPriceBUY:\t%.2f\n"+
			"stopPriceSELL:\t%.2f\n"+
			"priceBUY:\t%.2f\n"+
			"priceSELL:\t%.2f\n",
			stat.Quantity,
			stat.ActualPrice,
			stat.ActualPricePercent,
			stat.StopPriceBUY,
			stat.StopPriceSELL,
			stat.PriceBUY,
			stat.PriceSELL)); err != nil {
			u.logger.
				WithError(err).
				Error(string(debug.Stack()))
		}
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				settings, err := u.settingsRepo.Load(symbol)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				u.logger.
					WithField("settings", settings.Status).
					Debug("settings")

				switch settings.Status {
				case mongoStructs.New.ToString():
					lastOrder, err := u.orderRepo.GetLast(symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if orderInfo.Status == OrderStatusFilled {
						orderTry = 1
						quantity = settings.Step
						sessionID = uuid.New().String()

						if err := u.initOrder(sendStat, symbol, quantity, settings, orderTry, sessionID); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}
					}

					if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.Enabled); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}
				case mongoStructs.LiquidationSELL.ToString():
					lastOrder, err := u.orderRepo.GetLast(symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if orderInfo.Status == OrderStatusFilled {
						sendOrderInfo(lastOrder)

						priceSELL := lastOrder.Price + ((lastOrder.Price) / 100 * 1.5)

						if err := u.CreateLimitOrder(&structs.Order{
							Symbol: symbol,
							Side:   SideSell,
							Price:  fmt.Sprintf("%.0f", priceSELL),
						}, settings.Limit, lastOrder.StopPrice, orderTry, sessionID); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.New); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}
					}
				case mongoStructs.LiquidationBUY.ToString():
					lastOrder, err := u.orderRepo.GetLast(symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					sendOrderInfo(lastOrder)

					var liquidDelta float64
					for i := 0; i < lastOrder.Try; i++ {
						actualPricePercent := lastOrder.StopPrice / 100 * (settings.Delta + (settings.DeltaStep * float64(i)))
						liquidDelta += actualPricePercent
					}

					priceBUY := lastOrder.StopPrice - (liquidDelta / 2)

					if err := u.CreateLimitOrder(&structs.Order{
						Symbol: symbol,
						Side:   SideBuy,
						Price:  fmt.Sprintf("%.0f", priceBUY),
					}, settings.Limit, lastOrder.StopPrice, orderTry, sessionID); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.LiquidationSELL); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}
				case mongoStructs.Disabled.ToString():

					continue
				}

				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					switch err {
					case sql.ErrNoRows:
						if err := u.initOrder(sendStat, symbol, quantity, settings, orderTry, sessionID); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}
						continue
					default:
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

				orderTry = lastOrder.Try
				quantity = lastOrder.Quantity

				orderList, err := u.getOrderList(lastOrder.OrderID)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				lastOrderStatus := lastOrder.Status

				for _, o := range orderList.Orders {
					orderInfo, err := u.getOrderInfo(o.OrderID, symbol)
					if err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					switch true {
					case orderInfo.Type == "STOP_LOSS_LIMIT" && orderInfo.Status == OrderStatusFilled:
						lastOrderStatus = OrderStatusCanceled
						orderTry++

						if orderInfo.Side == SideSell {
							quantity = settings.Step * math.Pow(2, float64(orderTry-1))
						}

					case orderInfo.Type == "LIMIT_MAKER" && orderInfo.Status == OrderStatusFilled:
						lastOrderStatus = OrderStatusFilled

						if orderInfo.Side == SideSell {
							orderTry = 1
							quantity = settings.Step
							sessionID = uuid.New().String()
						}
					}

				}

				u.logger.
					WithField("lastOrderStatus", lastOrder.Status).
					WithField("newOrderStatus", lastOrderStatus).
					WithField("orderTry", orderTry).
					WithField("quantity", quantity).
					WithField("sessionID", sessionID).
					Debug("update status")

				if lastOrder.Status != lastOrderStatus {
					if err := u.orderRepo.SetStatus(lastOrder.ID, lastOrderStatus); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if lastOrder, err = u.orderRepo.GetByID(lastOrder.ID); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

				if quantity > settings.Limit {
					sendLimit(quantity)

					if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.LiquidationBUY); err != nil {
						u.logger.
							WithError(err).
							Error(string(debug.Stack()))
					}

					continue
				}

				var actualPrice float64

				switch lastOrder.Status {
				case OrderStatusFilled:
					actualPrice = lastOrder.Price
				case OrderStatusCanceled:
					actualPrice = lastOrder.StopPrice
				case OrderStatusNew:
					continue
				}

				go sendOrderInfo(lastOrder)

				pricePlan := u.fillPricePlan(actualPrice, quantity, settings, orderTry)

				openOrders, err := u.getOpenOrders(symbol)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
					continue
				}

				if len(openOrders) == 0 {
					switch lastOrder.Side {
					case SideBuy:
						go sendStat(pricePlan)

						if err := u.createOrder(&structs.Order{
							Symbol:    symbol,
							Side:      SideSell,
							Price:     fmt.Sprintf("%.0f", pricePlan.PriceSELL),
							StopPrice: fmt.Sprintf("%.0f", pricePlan.StopPriceSELL),
						}, quantity, actualPrice, orderTry, sessionID); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
							continue
						}

					case SideSell:
						go sendStat(pricePlan)

						if err := u.createOrder(&structs.Order{
							Symbol:    symbol,
							Side:      SideBuy,
							Price:     fmt.Sprintf("%.0f", pricePlan.PriceBUY),
							StopPrice: fmt.Sprintf("%.0f", pricePlan.StopPriceBUY),
						}, quantity, actualPrice, orderTry, sessionID); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
							continue
						}

					}
				}
			}
		}
	}()

	return nil
}

func (u *orderUseCase) fillPricePlan(actualPrice, quantity float64, settings *mongoStructs.Settings, orderTry int) *structs.PricePlan {
	var out structs.PricePlan

	out.ActualPrice = actualPrice

	out.ActualPricePercent = out.ActualPrice / 100 * settings.Delta
	//out.ActualPricePercent = out.ActualPrice / 100 * (settings.Delta + (settings.DeltaStep * float64(orderTry)))
	out.ActualStopPricePercent = out.ActualPricePercent

	out.StopPriceBUY = out.ActualPrice + out.ActualStopPricePercent
	out.StopPriceSELL = out.ActualPrice - out.ActualStopPricePercent

	out.PriceBUY = out.ActualPrice - out.ActualPricePercent
	out.PriceSELL = out.ActualPrice + out.ActualPricePercent

	out.Quantity = quantity

	return &out
}
func (u *orderUseCase) initOrder(sendStat func(stat *structs.PricePlan), symbol string, quantity float64, settings *mongoStructs.Settings, orderTry int, sessionID string) error {
	actualPrice, err := u.priceUseCase.GetPrice(symbol)
	if err != nil {
		return err
	}

	pricePlan := u.fillPricePlan(actualPrice, quantity, settings, orderTry)

	if err := u.createOrder(&structs.Order{
		Symbol:    symbol,
		Side:      SideBuy,
		Price:     fmt.Sprintf("%.0f", pricePlan.PriceBUY),
		StopPrice: fmt.Sprintf("%.0f", pricePlan.StopPriceBUY),
	}, quantity, actualPrice, orderTry, sessionID); err != nil {
		return err
	}

	sendStat(pricePlan)

	return nil
}
func (u *orderUseCase) liquidOrder(sendStat func(stat *structs.PricePlan), symbol, sessionID string, quantity float64, settings *mongoStructs.Settings, orderTry int) error {
	actualPrice, err := u.priceUseCase.GetPrice(symbol)
	if err != nil {
		return err
	}

	pricePlan := u.fillPricePlan(actualPrice, quantity, settings, orderTry)

	if err := u.createOrder(&structs.Order{
		Symbol:    symbol,
		Side:      SideSell,
		Price:     fmt.Sprintf("%.0f", pricePlan.PriceBUY),
		StopPrice: fmt.Sprintf("%.0f", pricePlan.StopPriceBUY),
	}, quantity, actualPrice, orderTry, sessionID); err != nil {
		return err
	}

	sendStat(pricePlan)

	return nil
}
func (u *orderUseCase) getOpenOrders(symbol string) ([]structs.Order, error) {
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
func (u *orderUseCase) getAllOrders(symbol string) error {
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
		if order.Status == OrderStatusNew {
			if err := u.tgmController.Send(fmt.Sprintf("[ Open Orders ]\n%s\n%s\n%s\n%d", order.Symbol, order.Side, order.Price, order.OrderId)); err != nil {
				return err
			}
		}
	}

	return nil
}
func (u *orderUseCase) getOrderList(orderListID int64) (*structs.OrderList, error) {
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
func (u *orderUseCase) getOrderInfo(orderID int64, symbol string) (*structs.Order, error) {
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
func (u *orderUseCase) CreateLimitOrder(order *structs.Order, quantity, actualPrice float64, try int, sessionID string) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("symbol", order.Symbol)
	q.Set("side", "BUY")
	q.Set("type", "LIMIT")
	q.Set("quantity", fmt.Sprintf("%.5f", quantity))
	q.Set("price", order.Price)
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	var o structs.LimitOrder
	if err := json.Unmarshal(req, &o); err != nil {
		return err
	}

	if o.OrderID == 0 {
		var errMsg structs.Err

		if err := json.Unmarshal(req, &errMsg); err != nil {
			return err
		}

		if err := u.tgmController.Send(fmt.Sprintf("[ Err createOrder ]\n"+
			"Code:\t%d\n"+
			"Msg:\t%s",
			errMsg.Code,
			errMsg.Msg,
		)); err != nil {
			return err
		}

		return errors.New(errMsg.Msg)
	}

	price, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		return err
	}

	orderModel := models.Order{
		OrderID:     o.OrderID,
		SessionID:   sessionID,
		Symbol:      o.Symbol,
		ActualPrice: actualPrice,
		Price:       price,
		Side:        o.Side,
		StopPrice:   0,
		Quantity:    quantity,
		Type:        "LIMIT",
		Status:      "NEW",
		Try:         try,
	}

	if err := u.orderRepo.Store(&orderModel); err != nil {
		return err
	}

	return nil
}
func (u *orderUseCase) createOrder(order *structs.Order, quantity, actualPrice float64, try int, sessionID string) error {
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

	type Err struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	var oList structs.OrderList
	var errMsg Err

	if err := json.Unmarshal(req, &oList); err != nil {
		return err
	}

	if oList.OrderListID == 0 {
		if err := json.Unmarshal(req, &errMsg); err != nil {
			return err
		}

		if err := u.tgmController.Send(fmt.Sprintf("[ Err createOrder ]\n"+
			"Code:\t%d\n"+
			"Msg:\t%s",
			errMsg.Code,
			errMsg.Msg,
		)); err != nil {
			return err
		}

		return errors.New(errMsg.Msg)
	}

	stopPrice, err := strconv.ParseFloat(order.StopPrice, 64)
	if err != nil {
		return err
	}

	price, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		return err
	}

	o := models.Order{
		OrderID:     oList.OrderListID,
		SessionID:   sessionID,
		Symbol:      oList.Symbol,
		ActualPrice: actualPrice,
		Price:       price,
		Side:        order.Side,
		StopPrice:   stopPrice,
		Quantity:    quantity,
		Type:        "OCO",
		Status:      "NEW",
		Try:         try,
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return err
	}

	return nil
}
func (u *orderUseCase) cancelOrder(symbol, orderId string) error {
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
