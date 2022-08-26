package usecasees

import (
	"binance/internal/repository/mongo"
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

	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
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

	ETH  = "ETH"
	RUB  = "RUB"
	BUSD = "BUSD"

	ETHRUB  = ETH + RUB
	ETHBUSD = ETH + BUSD

	SideSell = "SELL"
	SideBuy  = "BUY"

	OrderStatusNew      = "NEW"
	OrderStatusCanceled = "CANCELED"
	OrderStatusFilled   = "FILLED"
)

var (
	SymbolList = []string{
		ETHBUSD,
	}
)

type orderUseCase struct {
	clientController *controllers.ClientController
	cryptoController *controllers.CryptoController
	tgmController    *controllers.TgmController

	settingsRepo *mongo.SettingsRepository

	orderRepo  *sqlite.OrderRepository
	priceRepo  *sqlite.PriceRepository
	candleRepo *sqlite.CandleRepository

	priceUseCase *priceUseCase

	url string

	logger *logrus.Logger
}

func NewOrderUseCase(
	client *controllers.ClientController,
	crypto *controllers.CryptoController,
	tgm *controllers.TgmController,
	settingsRepo *mongo.SettingsRepository,
	orderRepo *sqlite.OrderRepository,
	priceRepo *sqlite.PriceRepository,
	candleRepo *sqlite.CandleRepository,
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
		priceRepo:        priceRepo,
		candleRepo:       candleRepo,
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

	sendOrderInfo := func(order *models.Order) {
		if err := u.tgmController.Send(fmt.Sprintf("[ Last Order ]\n"+
			"orderId:\t%d\n"+
			"status:\t%s\n"+
			"side:\t%s\n",
			order.OrderId,
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
			"quantity:\t%.5f\n"+
			"actualPrice:\t%.5f\n"+
			"actualPricePercent:\t%.5f\n"+
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
				if err := u.settingsRepo.ReLoad(settings); err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
				}

				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					switch err {
					case sql.ErrNoRows:
						if err := u.initOrder(sendStat, symbol, quantity, settings.Delta, orderTry); err != nil {
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

				orderList, err := u.getOrderList(lastOrder.OrderId)
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
						}
					}

				}

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

				pricePlan := u.fillPricePlan(actualPrice, quantity, settings.Delta, orderTry)

				openOrders, err := u.getOpenOrders(symbol)
				if err != nil {
					u.logger.
						WithError(err).
						Error(string(debug.Stack()))
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
						}, quantity, actualPrice, orderTry); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}

					case SideSell:
						go sendStat(pricePlan)

						if err := u.createOrder(&structs.Order{
							Symbol:    symbol,
							Side:      SideBuy,
							Price:     fmt.Sprintf("%.0f", pricePlan.PriceBUY),
							StopPrice: fmt.Sprintf("%.0f", pricePlan.StopPriceBUY),
						}, quantity, actualPrice, orderTry); err != nil {
							u.logger.
								WithError(err).
								Error(string(debug.Stack()))
						}

					}
				}
			}
		}
	}()

	return nil
}

func (u *orderUseCase) fillPricePlan(actualPrice, quantity, deltaOrder float64, orderTry int) *structs.PricePlan {
	var out structs.PricePlan

	out.ActualPricePercent = actualPrice / 100 * (deltaOrder + (0.025 * float64(orderTry)))
	out.ActualStopPricePercent = out.ActualPricePercent

	out.StopPriceBUY = actualPrice + out.ActualStopPricePercent
	out.StopPriceSELL = actualPrice - out.ActualStopPricePercent

	out.PriceBUY = actualPrice - out.ActualPricePercent
	out.PriceSELL = actualPrice + out.ActualPricePercent

	out.Quantity = quantity

	return &out
}

func (u *orderUseCase) initOrder(sendStat func(stat *structs.PricePlan), symbol string, quantity, deltaOrder float64, orderTry int) error {
	actualPrice, err := u.priceUseCase.GetPrice(symbol)
	if err != nil {
		return err
	}

	pricePlan := u.fillPricePlan(actualPrice, quantity, deltaOrder, orderTry)

	if err := u.createOrder(&structs.Order{
		Symbol:    symbol,
		Side:      SideBuy,
		Price:     fmt.Sprintf("%.0f", pricePlan.PriceBUY),
		StopPrice: fmt.Sprintf("%.0f", pricePlan.StopPriceBUY),
	}, quantity, actualPrice, orderTry); err != nil {
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

func (u *orderUseCase) createOrder(order *structs.Order, quantity, actualPrice float64, try int) error {
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

	var oList OrderList
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
		OrderId:     oList.OrderListID,
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
