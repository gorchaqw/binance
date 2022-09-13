package usecasees

import (
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/usecasees/structs"
	"binance/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"database/sql"
	"runtime/debug"
	"time"
)

func (u *orderUseCase) FeaturesMonitoring(symbol string) error {
	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)

	settings, err := u.settingsRepo.Load(symbol)
	if err != nil {
		return err
	}

	var status structs.Status
	status.Reset(settings.Step)

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
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("Settings: %+v", settings)

				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					switch err {
					case sql.ErrNoRows:
						actualPrice, err := u.priceUseCase.featuresGetPrice(symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, actualPrice, settings, &status).SetSide(SideBuy)

						if err := u.createFeaturesMarketOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						continue

					default:
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}
				u.promTail.Debugf("LastOrder: %+v", lastOrder)

				status.
					SetSessionID(lastOrder.SessionID).
					SetOrderTry(lastOrder.Try).
					SetQuantityByStep(settings.Step)

				status.SetMode(structs.Middle)

				u.logRus.Debugf("lastOrder: %+v", lastOrder)

				oList, err := u.orderRepo.GetBySessionID(status.SessionID)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}

				ordersStatus, err := u.fillFeatureOrdersStatus(oList)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}

				u.logRus.Debugf("ordersStatus: %+v", ordersStatus)

				chkCreateLimitOrderStopLoss := func(f *structs.FeatureOrdersStatus) bool {
					switch true {
					case f.MarketOrder.Status == OrderStatusFilled &&
						f.StopLossOrder.Status == OrderStatusFilled &&
						f.TakeProfitOrder.Status == OrderStatusCanceled:

						return true
					}
					return false
				}

				chkCreateLimitOrderTakeProfit := func(f *structs.FeatureOrdersStatus) bool {
					switch true {
					case f.MarketOrder.Status == OrderStatusFilled &&
						f.StopLossOrder.Status == OrderStatusCanceled &&
						f.TakeProfitOrder.Status == OrderStatusFilled:

						return true
					}
					return false
				}

				chkCreateOrders := func(f *structs.FeatureOrdersStatus) bool {
					switch true {
					case f.MarketOrder.Status == OrderStatusFilled &&
						f.TakeProfitOrder.Status == "" &&
						f.StopLossOrder.Status == "":

						return true
					}
					return false
				}

				chkTakeProfitCancel := func(f *structs.FeatureOrdersStatus) bool {
					switch true {
					case f.MarketOrder.Status == OrderStatusFilled &&
						f.StopLossOrder.Status == OrderStatusFilled &&
						f.TakeProfitOrder.Status == OrderStatusNew:
						return true
					}
					return false
				}

				chkStopLossCancel := func(f *structs.FeatureOrdersStatus) bool {
					switch true {
					case f.MarketOrder.Status == OrderStatusFilled &&
						f.TakeProfitOrder.Status == OrderStatusFilled &&
						f.StopLossOrder.Status == OrderStatusNew:
						return true
					}
					return false
				}

				u.logRus.Debugf("ordersStatus: %+v", ordersStatus)

				//if chkLimitOrderActual(ordersStatus) {
				//	if _, err := u.cancelFeatureOrder(ordersStatus.LimitOrder.OrderId, symbol); err != nil {
				//		u.logRus.
				//			WithError(err).
				//			Error(string(debug.Stack()))
				//	}
				//
				//	actualPrice, err := u.priceUseCase.featuresGetPrice(symbol)
				//	if err != nil {
				//		u.logRus.
				//			WithError(err).
				//			Error(string(debug.Stack()))
				//	}
				//
				//	status.
				//		NewSessionID().
				//		AddOrderTry(1).
				//		SetQuantityByStep(settings.Step)
				//
				//	pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, actualPrice, settings, &status)
				//
				//	switch ordersStatus.LimitOrder.Side {
				//	case SideBuy:
				//		pricePlan.SetSide(SideSell)
				//	case SideSell:
				//		pricePlan.SetSide(SideBuy)
				//	}
				//
				//	if err := u.createFeaturesLimitOrder(pricePlan, settings); err != nil {
				//		u.logRus.
				//			WithError(err).
				//			Error(string(debug.Stack()))
				//	}
				//
				//	u.logRus.Debugf("chkLimit pricePlan %+v", pricePlan)
				//}

				if chkCreateLimitOrderStopLoss(ordersStatus) {
					status.
						NewSessionID().
						AddOrderTry(1).
						SetQuantityByStep(settings.Step)

					u.metrics[structs.MetricOrderStopLossLimitFilled].Inc()

					pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, ordersStatus.StopLossOrder.ActualPrice, settings, &status)

					switch ordersStatus.MarketOrder.Side {
					case SideBuy:
						pricePlan.SetSide(SideSell)
					case SideSell:
						pricePlan.SetSide(SideBuy)
					}

					if err := u.createFeaturesMarketOrder(pricePlan, settings); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					u.logRus.Debugf("chkLimit pricePlan %+v", pricePlan)

				}

				if chkCreateLimitOrderTakeProfit(ordersStatus) {
					status.Reset(settings.Step)

					u.metrics[structs.MetricOrderLimitMaker].Inc()

					pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, ordersStatus.TakeProfitOrder.ActualPrice, settings, &status)

					switch ordersStatus.MarketOrder.Side {
					case SideBuy:
						pricePlan.SetSide(SideSell)
					case SideSell:
						pricePlan.SetSide(SideBuy)
					}

					if err := u.createFeaturesMarketOrder(pricePlan, settings); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					u.logRus.Debugf("chkLimit pricePlan %+v", pricePlan)

				}

				if chkCreateOrders(ordersStatus) {
					pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, ordersStatus.MarketOrder.ActualPrice, settings, &status)

					u.logRus.Debugf("chkLimit pricePlan %+v", pricePlan)

					switch ordersStatus.MarketOrder.Side {
					case SideBuy:
						pricePlan.SetSide(SideBuy)
					case SideSell:
						pricePlan.SetSide(SideSell)
					}

					orders := []structs.FeatureOrderReq{
						u.constructTakeProfitOrder(pricePlan, settings),
						u.constructStopLossOrder(pricePlan, settings),
					}

					if err := u.createBatchOrders(pricePlan, settings, orders); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

				}

				if chkStopLossCancel(ordersStatus) {
					u.logRus.Debugf("chkStopLossCancel: %+v", ordersStatus)

					if _, err := u.cancelFeatureOrder(ordersStatus.StopLossOrder.OrderId, symbol); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

				if chkTakeProfitCancel(ordersStatus) {
					u.logRus.Debugf("chkTakeProfitCancel: %+v", ordersStatus)

					if _, err := u.cancelFeatureOrder(ordersStatus.TakeProfitOrder.OrderId, symbol); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}
				}

			}
		}
	}()

	return nil
}

func (u *orderUseCase) fillFeatureOrdersStatus(oList []models.Order) (*structs.FeatureOrdersStatus, error) {
	var out structs.FeatureOrdersStatus
	for _, o := range oList {
		order, err := u.getFeatureOrderInfo(o.OrderID, o.Symbol)
		if err != nil {

		}

		if o.Status != order.Status {
			if err := u.orderRepo.SetStatus(order.OrderId, order.Status); err != nil {
				return nil, err
			}

			o.Status = order.Status
		}

		switch o.Type {
		case OrderTypeMarket:
			avgPrice, err := strconv.ParseFloat(order.AvgPrice, 64)
			if err != nil {
				return nil, err
			}

			out.MarketOrder = structs.OrderStatus{
				OrderId:     order.OrderId,
				Status:      order.Status,
				ActualPrice: avgPrice,
				Side:        order.Side,
			}
		case OrderTypeTakeProfit:
			stopPrice, err := strconv.ParseFloat(order.StopPrice, 64)
			if err != nil {
				return nil, err
			}

			out.TakeProfitOrder = structs.OrderStatus{
				OrderId:     order.OrderId,
				Status:      order.Status,
				ActualPrice: stopPrice,
				Side:        order.Side,
			}
		case OrderTypeStopLoss:
			stopPrice, err := strconv.ParseFloat(order.StopPrice, 64)
			if err != nil {
				return nil, err
			}

			out.StopLossOrder = structs.OrderStatus{
				OrderId:     order.OrderId,
				Status:      order.Status,
				ActualPrice: stopPrice,
				Side:        order.Side,
			}
		}
	}

	return &out, nil
}

//func (u *orderUseCase) constructLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) structs.FeatureOrderReq {
//	u.logRus.Debugf("constructLimitOrder: %+v", pricePlan)
//
//	out := structs.FeatureOrderReq{
//		Symbol:      settings.Symbol,
//		Type:        "LIMIT",
//		Price:       fmt.Sprintf("%.1f", pricePlan.ActualPrice),
//		Quantity:    fmt.Sprintf("%.4f", pricePlan.Status.Quantity),
//		TimeInForce: "GTC",
//	}
//
//	switch pricePlan.Side {
//	case SideBuy:
//		out.Side = SideBuy
//		out.PositionSide = "LONG"
//	case SideSell:
//		out.Side = SideSell
//		out.PositionSide = "SHORT"
//	}
//
//
//	return out
//}

func (u *orderUseCase) constructTakeProfitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) structs.FeatureOrderReq {
	u.logRus.Debugf("constructTakeProfitOrder: %+v", pricePlan)

	out := structs.FeatureOrderReq{
		Symbol:       settings.Symbol,
		Type:         OrderTypeTakeProfit,
		PriceProtect: "true",
		Quantity:     fmt.Sprintf("%.3f", pricePlan.Status.Quantity),
	}

	switch pricePlan.Side {
	case SideBuy:
		out.Side = SideSell
		out.Price = fmt.Sprintf("%.1f", pricePlan.PriceSELL)
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.PriceSELL-pricePlan.SafeDelta)
		out.PositionSide = "LONG"
	case SideSell:
		out.Side = SideBuy
		out.Price = fmt.Sprintf("%.1f", pricePlan.PriceBUY)
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.PriceBUY+pricePlan.SafeDelta)
		out.PositionSide = "SHORT"
	}

	return out
}
func (u *orderUseCase) constructStopLossOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) structs.FeatureOrderReq {
	u.logRus.Debugf("constructStopLossOrder: %+v", pricePlan)

	out := structs.FeatureOrderReq{
		Symbol:       settings.Symbol,
		Type:         OrderTypeStopLoss,
		PriceProtect: "true",
		Quantity:     fmt.Sprintf("%.3f", pricePlan.Status.Quantity),
	}

	switch pricePlan.Side {
	case SideBuy:
		out.Side = SideSell
		out.Price = fmt.Sprintf("%.1f", pricePlan.StopPriceSELL)
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.StopPriceSELL-pricePlan.SafeDelta)
		out.PositionSide = "LONG"
	case SideSell:
		out.Side = SideBuy
		out.Price = fmt.Sprintf("%.1f", pricePlan.StopPriceBUY)
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.StopPriceBUY+pricePlan.SafeDelta)
		out.PositionSide = "SHORT"
	}
	return out
}

func (u *orderUseCase) createBatchOrders(pricePlan *structs.PricePlan, settings *mongoStructs.Settings, orders []structs.FeatureOrderReq) error {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureBatchOrders)

	batchOrders, err := json.Marshal(orders)
	if err != nil {
		return err
	}

	q := baseURL.Query()
	q.Set("batchOrders", fmt.Sprintf("%s", batchOrders))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}
	var orderList []structs.FeatureOrderResp

	u.logRus.Debugf("createBatchOrders Req: %s", req)
	u.logRus.Debugf("createBatchOrders PricePlan: %+v", pricePlan)
	u.logRus.Debugf("createBatchOrders Orders: %+v", orders)

	if err := json.Unmarshal(req, &orderList); err != nil {
		return errors.New(fmt.Sprintf("%+v %s", err, req))
	}

	for _, order := range orderList {
		var quantity, stopPrice, price float64

		if order.OrigQty != "" {
			quantity, err = strconv.ParseFloat(order.OrigQty, 64)
			if err != nil {
				return err
			}
		}

		if order.StopPrice != "" {
			stopPrice, err = strconv.ParseFloat(order.StopPrice, 64)
			if err != nil {
				return err
			}
		}

		if order.Price != "" {
			price, err = strconv.ParseFloat(order.Price, 64)
			if err != nil {
				return err
			}
		}

		o := models.Order{
			OrderID:     order.OrderId,
			SessionID:   pricePlan.Status.SessionID,
			Try:         pricePlan.Status.OrderTry,
			ActualPrice: pricePlan.ActualPrice,
			Symbol:      order.Symbol,
			Side:        order.Side,
			Type:        order.Type,
			Status:      order.Status,
			Quantity:    quantity,
			StopPrice:   stopPrice,
			Price:       price,
		}

		if err := u.orderRepo.Store(&o); err != nil {
			return err
		}
	}

	return nil
}

func (u *orderUseCase) getFeaturePositionInfo(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featurePositionInfo)

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
func (u *orderUseCase) getFeatureOrderInfo(orderID int64, symbol string) (*structs.FeatureOrderResp, error) {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featureOrder)

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

	var out structs.FeatureOrderResp

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
func (u *orderUseCase) cancelFeatureOrder(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", fmt.Sprintf("%d", orderID))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodDelete, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *orderUseCase) createFeaturesMarketOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", pricePlan.Symbol)
	switch pricePlan.Side {
	case SideBuy:
		q.Set("side", SideBuy)
		q.Set("positionSide", "LONG")
	case SideSell:
		q.Set("side", SideSell)
		q.Set("positionSide", "SHORT")
	}
	q.Set("type", "MARKET")
	q.Set("quantity", fmt.Sprintf("%.3f", pricePlan.Status.Quantity))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()
	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}
	u.logRus.Debugf("createFeaturesMarketOrder Req: %s", req)

	var order structs.FeatureOrderResp
	if err := json.Unmarshal(req, &order); err != nil {
		return err
	}

	var quantity float64

	if order.OrigQty != "" {
		quantity, err = strconv.ParseFloat(order.OrigQty, 64)
		if err != nil {
			return err
		}
	}

	o := models.Order{
		OrderID:     order.OrderId,
		SessionID:   pricePlan.Status.SessionID,
		Try:         pricePlan.Status.OrderTry,
		ActualPrice: pricePlan.ActualPrice,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Type:        order.Type,
		Status:      order.Status,
		Quantity:    quantity,
		StopPrice:   0,
		Price:       0,
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) createFeaturesLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("type", "LIMIT")
	q.Set("quantity", fmt.Sprintf("%.4f", pricePlan.Status.Quantity))
	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.1f", pricePlan.ActualPrice))
		q.Set("positionSide", "LONG")
	case SideSell:
		q.Set("price", fmt.Sprintf("%.1f", pricePlan.ActualPrice))
		q.Set("positionSide", "SHORT")
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

	u.promTail.Debugf("createFeaturesLimitOrder Req: %s", req)

	var order structs.LimitOrder
	var errMsg structs.Err

	if err := json.Unmarshal(req, &order); err != nil {
		return err
	}

	if order.OrderID == 0 {
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
		OrderID:     order.OrderID,
		SessionID:   pricePlan.Status.SessionID,
		Symbol:      order.Symbol,
		ActualPrice: pricePlan.ActualPrice,
		Side:        pricePlan.Side,
		Quantity:    pricePlan.Status.Quantity,
		Type:        "LIMIT",
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

func (u *priceUseCase) featuresGetPrice(symbol string) (float64, error) {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return 0, err
	}

	baseURL.Path = path.Join(featureSymbolPrice)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
	if err != nil {
		return 0, err
	}

	type reqJson struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(out.Price, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}
