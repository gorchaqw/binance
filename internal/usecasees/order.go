package usecasees

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"

	"binance/internal/controllers"
	"binance/internal/repository/mongo"
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/repository/postgres"
	"binance/internal/usecasees/structs"
	"binance/models"
)

const (
	orderUrlPath     = "/api/v3/order"
	orderList        = "/api/v3/orderList"
	orderAllUrlPath  = "/api/v3/allOrders"
	orderOpenUrlPath = "/api/v3/openOrders"
	orderOCO         = "/api/v3/order/oco"

	featureURL = "https://fapi.binance.com"
	//featureURL          = "https://testnet.binancefuture.com"
	featureOrder        = "/fapi/v1/order"
	featurePositionInfo = "/fapi/v2/positionRisk"
	featureBatchOrders  = "/fapi/v1/batchOrders"
	featureSymbolPrice  = "/fapi/v1/ticker/price"

	BTC  = "BTC"
	ETH  = "ETH"
	RUB  = "RUB"
	BUSD = "BUSD"
	USDT = "USDT"

	ETHRUB  = ETH + RUB
	ETHBUSD = ETH + BUSD
	BTCBUSD = BTC + BUSD
	BTCUSDT = BTC + USDT

	SideSell = "SELL"
	SideBuy  = "BUY"

	OrderStatusNew        = "NEW"
	OrderStatusCanceled   = "CANCELED"
	OrderStatusFilled     = "FILLED"
	OrderStatusExpired    = "EXPIRED"
	OrderStatusNotFound   = "NOT_FOUND"
	OrderStatusInProgress = "IN PROGRESS"
	OrderStatusError      = "ERROR"

	OrderTypeLimit      = "LIMIT"
	OrderTypeMarket     = "MARKET"
	OrderTypeTakeProfit = "TAKE_PROFIT_MARKET"
	OrderTypeStopLoss   = "STOP_MARKET"
	OrderTypeOCO        = "OCO"
	OrderTypeBatch      = "BATCH"

	OrderTypeMarketID     = 0
	OrderTypeTakeProfitID = 1
	OrderTypeStopLossID   = 2
)

var (
	SymbolList = []string{
		BTCBUSD,
	}
)

type orderUseCase struct {
	clientController controllers.ClientCtrl
	cryptoController controllers.CryptoCtrl

	settingsRepo mongo.SettingsRepo
	orderRepo    postgres.OrderRepo

	priceUseCase *priceUseCase

	url string

	logRus *logrus.Logger
}

func NewOrderUseCase(
	client controllers.ClientCtrl,
	crypto controllers.CryptoCtrl,
	settingsRepo mongo.SettingsRepo,
	orderRepo postgres.OrderRepo,
	priceUseCase *priceUseCase,
	url string,
	logger *logrus.Logger,
) *orderUseCase {
	return &orderUseCase{
		clientController: client,
		cryptoController: crypto,
		settingsRepo:     settingsRepo,
		orderRepo:        orderRepo,
		priceUseCase:     priceUseCase,
		url:              url,
		logRus:           logger,
	}
}

func (u *orderUseCase) Monitoring(symbol string) error {
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)

	settings, err := u.settingsRepo.Load(symbol)
	if err != nil {
		return err
	}

	var status structs.Status
	status.Reset(settings.Step)

	//sendOrderInfo := func(order *models.Order) {
	//	if err := u.tgmController.Send(fmt.Sprintf("[ Last Order ]\n"+
	//		"orderId:\t%d\n"+
	//		"status:\t%s\n"+
	//		"side:\t%s\n"+
	//		"order:\t%+v\n",
	//		order.OrderID,
	//		order.Status,
	//		order.Side,
	//		order)); err != nil {
	//		u.logRus.
	//			WithError(err).
	//			Error(string(debug.Stack()))
	//		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
	//	}
	//}
	//sendLimit := func(quantity float64) {
	//	if err := u.tgmController.Send(fmt.Sprintf("[ Limit ]\n"+
	//		"quantity:\t%.5f\n",
	//		quantity)); err != nil {
	//		u.logRus.
	//			WithError(err).
	//			Error(string(debug.Stack()))
	//		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
	//	}
	//}
	//sendStat := func(stat *structs.PricePlan) {
	//	if err := u.tgmController.Send(fmt.Sprintf("[ Stat ]\n"+
	//		"actualPrice:\t%.2f\n"+
	//		"actualPricePercent:\t%.2f\n"+
	//		"stopPriceBUY:\t%.2f\n"+
	//		"stopPriceSELL:\t%.2f\ncreateFeaturesLimitOrder"+
	//		"priceBUY:\t%.2f\n"+
	//		"priceSELL:\t%.2f\n",
	//		stat.ActualPrice,
	//		stat.ActualPricePercent,
	//		stat.StopPriceBUY,
	//		stat.StopPriceSELL,
	//		stat.PriceBUY,
	//		stat.PriceSELL)); err != nil {
	//		u.logRus.
	//			WithError(err).
	//			Error(string(debug.Stack()))
	//		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
	//	}
	//}

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				settings, err = u.settingsRepo.Load(symbol)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
				}

				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					switch err {
					case sql.ErrNoRows:
						status.Reset(settings.Step)

						actualPrice, err := u.priceUseCase.GetPrice(symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))

							continue
						}
					default:
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

				u.logRus.
					WithField("settings", settings.Status).
					Debug("settings")

				switch settings.Status {
				case mongoStructs.New.ToString():
					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if orderInfo.Status == OrderStatusFilled && orderInfo.Side == SideSell && orderInfo.Type == "LIMIT" {
						status.Reset(settings.Step)

						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, lastOrder.Price, settings, &status).SetSide(SideBuy)
						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))

							continue
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.Enabled); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}
					}
				case mongoStructs.LiquidationSELL.ToString():
					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if orderInfo.Status == OrderStatusFilled && orderInfo.Side == SideBuy && orderInfo.Type == "LIMIT" {
						pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, lastOrder.Price, settings, &status).SetSide(SideSell)

						u.logRus.
							WithField("pricePlan", pricePlan).
							Debug("LiquidationSELL")

						if err := u.CreateLimitOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))

							continue
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.New); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}
					}
				case mongoStructs.LiquidationBUY.ToString():
					if lastOrder.Status == OrderStatusCanceled && lastOrder.Side == SideSell && lastOrder.Type == "OCO" {
						status.
							SetQuantityByStep(settings.Step).
							SetOrderTry(1)

						pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, lastOrder.StopPrice, settings, &status).SetSide(SideBuy)

						u.logRus.
							WithField("pricePlan", pricePlan).
							WithField("status", pricePlan.Status).
							Debug("LiquidationBUY")

						if err := u.CreateLimitOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							continue
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.LiquidationSELL); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}
					}
				case mongoStructs.Disabled.ToString():
					continue
				}

				status.
					SetOrderTry(lastOrder.Try).
					SetQuantityByStep(lastOrder.Quantity)

				lastOrderStatus := lastOrder.Status

				switch lastOrder.Type {
				case "LIMIT":
					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					if orderInfo.Status == OrderStatusFilled {
						lastOrderStatus = OrderStatusFilled
					}
				case "OCO":
					orderList, err := u.getOrderList(lastOrder.OrderID)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					for _, o := range orderList.Orders {
						orderInfo, err := u.getOrderInfo(o.OrderID, symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						switch true {
						case orderInfo.Type == "STOP_LOSS_LIMIT" && orderInfo.Status == OrderStatusFilled:
							lastOrderStatus = OrderStatusCanceled

							status.
								AddOrderTry(1)

							if orderInfo.Side == SideSell {
								status.SetQuantityByStep(settings.Step)
							}

						case orderInfo.Type == "LIMIT_MAKER" && orderInfo.Status == OrderStatusFilled:
							lastOrderStatus = OrderStatusFilled

							if orderInfo.Side == SideSell {
								status.Reset(settings.Step)

							}
						}
					}
				}

				u.logRus.
					WithField("lastOrderStatus", lastOrder.Status).
					WithField("newOrderStatus", lastOrderStatus).
					WithField("orderTry", status.OrderTry).
					WithField("quantity", status.Quantity).
					WithField("sessionID", status.SessionID).
					Debug("update status")

				if lastOrder.Status != lastOrderStatus {
					if err := u.orderRepo.SetStatus(lastOrder.ID, lastOrderStatus); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}
					//
					//if lastOrder, err = u.orderRepo.GetByID(lastOrder.ID); err != nil {
					//	u.logRus.
					//		WithError(err).
					//		Error(string(debug.Stack()))
					//}
				}

				if status.Quantity > settings.Limit {
					if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.LiquidationBUY); err != nil {
						u.logRus.
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

				openOrders, err := u.getOpenOrders(symbol)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))

					continue
				}

				if len(openOrders) == 0 {
					switch lastOrder.Side {
					case SideBuy:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideSell)
						u.logRus.Debug(pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))

							continue
						}

					case SideSell:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						u.logRus.Debug(pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
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

func (u *orderUseCase) fillPricePlan(orderType string, symbol string, actualPrice float64, settings *mongoStructs.Settings, status *structs.Status) *structs.PricePlan {
	var out structs.PricePlan

	out.Symbol = symbol
	out.ActualPrice = actualPrice
	out.SafeDelta = float64(settings.Delta) / 100 * 10

	switch orderType {
	case OrderTypeLimit:
		out.ActualPricePercent = out.ActualPrice / 100 * (float64(settings.Delta) * 1.2)
		out.ActualStopPricePercent = out.ActualPrice / 100 * (float64(settings.Delta) * 1.2)
	case OrderTypeOCO:
		out.ActualPricePercent = out.ActualPrice / 100 * float64(settings.Delta)
		out.ActualStopPricePercent = out.ActualPrice / 100 * float64(settings.Delta)
	case OrderTypeBatch:
		out.ActualPricePercent = float64(settings.Delta)
		out.ActualStopPricePercent = float64(settings.Delta)
	}

	//out.ActualPricePercent = out.ActualPrice / 100 * (settings.Delta + (settings.DeltaStep * float64(orderTry)))

	out.StopPriceBUY = out.ActualPrice + out.ActualStopPricePercent
	out.StopPriceSELL = out.ActualPrice - out.ActualStopPricePercent

	out.PriceBUY = out.ActualPrice - out.ActualPricePercent
	out.PriceSELL = out.ActualPrice + out.ActualPricePercent

	out.Status = status

	return &out
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
func (u *orderUseCase) CreateLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	actualPrice, err := u.priceUseCase.GetPrice(pricePlan.Symbol)
	if err != nil {
		return err
	}

	switch pricePlan.Side {
	case SideBuy:
		if actualPrice < pricePlan.PriceBUY {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideBuy)
			pricePlan = newPricePlan

		}
	case SideSell:
		if actualPrice > pricePlan.PriceSELL {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideSell)
			pricePlan = newPricePlan
		}
	}

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}
	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("type", "LIMIT")
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("quantity", fmt.Sprintf("%.5f", pricePlan.Status.Quantity))
	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceBUY))
	case SideSell:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceSELL))
	}
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

		return errors.New(errMsg.Msg)
	}

	orderModel := models.Order{
		OrderID:     o.OrderID,
		SessionID:   pricePlan.Status.SessionID,
		Symbol:      o.Symbol,
		ActualPrice: pricePlan.ActualPrice,
		Side:        pricePlan.Side,
		Quantity:    pricePlan.Status.Quantity,
		Type:        "LIMIT",
		Status:      "NEW",
		Try:         pricePlan.Status.OrderTry,
	}

	switch pricePlan.Side {
	case SideBuy:
		orderModel.Price = pricePlan.PriceBUY
	case SideSell:
		orderModel.Price = pricePlan.PriceSELL
	}

	if err := u.orderRepo.Store(&orderModel); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) createOCOOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	actualPrice, err := u.priceUseCase.GetPrice(pricePlan.Symbol)
	if err != nil {
		return err
	}

	switch pricePlan.Side {
	case SideBuy:
		if actualPrice < pricePlan.PriceBUY {
			newPricePlan := u.fillPricePlan(OrderTypeOCO, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideBuy)
			pricePlan = newPricePlan

		}
	case SideSell:
		if actualPrice > pricePlan.PriceSELL {
			newPricePlan := u.fillPricePlan(OrderTypeOCO, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideSell)
			pricePlan = newPricePlan
		}
	}

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderOCO)

	q := baseURL.Query()
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("quantity", fmt.Sprintf("%.5f", pricePlan.Status.Quantity))

	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceBUY))
		q.Set("stopPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceBUY))
		q.Set("stopLimitPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceBUY))
	case SideSell:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceSELL))
		q.Set("stopPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceSELL))
		q.Set("stopLimitPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceSELL))
	}

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

	var oList structs.OrderList
	var errMsg structs.Err

	if err := json.Unmarshal(req, &oList); err != nil {
		return err
	}

	if oList.OrderListID == 0 {
		if err := json.Unmarshal(req, &errMsg); err != nil {
			return err
		}

		switch errMsg.Code {
		case -2010:
			return structs.ErrTheRelationshipOfThePrices
		}

		return errors.New(errMsg.Msg)
	}

	o := models.Order{
		OrderID:     oList.OrderListID,
		SessionID:   pricePlan.Status.SessionID,
		Symbol:      oList.Symbol,
		ActualPrice: pricePlan.ActualPrice,
		Side:        pricePlan.Side,
		Quantity:    pricePlan.Status.Quantity,
		Type:        "OCO",
		Status:      "NEW",
		Try:         pricePlan.Status.OrderTry,
	}

	switch pricePlan.Side {
	case SideBuy:
		o.Price = pricePlan.PriceBUY
		o.StopPrice = pricePlan.StopPriceBUY
	case SideSell:
		o.Price = pricePlan.PriceSELL
		o.StopPrice = pricePlan.StopPriceSELL
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
