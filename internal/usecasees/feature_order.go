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
	ticker := time.NewTicker(2 * time.Second)
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
						status.Reset(settings.Step)

						actualPrice, err := u.priceUseCase.GetPrice(symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						if err := u.createBatchOrders(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

					default:
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}
				u.promTail.Debugf("LastOrder: %+v", lastOrder)

				status.
					SetOrderTry(lastOrder.Try).
					SetQuantity(lastOrder.Quantity).
					SetSessionID(lastOrder.SessionID)

				oList, err := u.orderRepo.GetBySessionID(lastOrder.SessionID)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}

				continue

				for _, o := range oList {
					orderInfo, err := u.getFeatureOrderInfo(o.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					if err := u.orderRepo.SetStatus(lastOrder.ID, orderInfo.Status); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}

				oListSELL, err := u.orderRepo.GetBySessionIDWithSide(lastOrder.SessionID, SideSell)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("oListSELL: %+v", oListSELL)

				oListBUY, err := u.orderRepo.GetBySessionIDWithSide(lastOrder.SessionID, SideBuy)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("oListBUY: %+v", oListBUY)

				sellOrders := func(orders []models.Order) (int, int) {
					var sFilled, sNew int
					for _, o := range orders {
						switch o.Side {
						case SideSell:
							if o.Status == OrderStatusFilled {
								sFilled++
							}
							if o.Status == OrderStatusNew {
								sNew++
							}
						}
					}

					return sFilled, sNew
				}

				buyOrders := func(orders []models.Order) (int, int) {
					var bFilled, bNew int
					for _, o := range orders {
						switch o.Side {
						case SideBuy:
							if o.Status == OrderStatusFilled {
								bFilled++
							}
							if o.Status == OrderStatusNew {
								bNew++
							}
						}
					}

					return bFilled, bNew
				}

				sFilled, sNew := sellOrders(oListSELL)
				bFilled, bNew := buyOrders(oListBUY)

				switch true {
				case len(oListSELL) == 1 && len(oListBUY) == 1 && bFilled == 0 && bNew == 1 && sFilled != 0 && sNew == 0:

					status.AddOrderTry(1)

					pricePlanSELL := u.fillPricePlan(OrderTypeOCO, symbol, oListSELL[0].ActualPrice, settings, &status).SetSide(SideSell)
					pricePlanBUY := u.fillPricePlan(OrderTypeOCO, symbol, oListBUY[0].Price, settings, &status).SetSide(SideBuy)

					if err := u.createFeaturesLimitOrder(pricePlanBUY, settings); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

						continue
					}
					if err := u.createFeaturesLimitOrder(pricePlanSELL, settings); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

						continue
					}
				}

				continue

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
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

					continue
				}

				if len(openOrders) == 0 {
					switch lastOrder.Side {
					case SideBuy:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideSell)
						u.logRus.Debug(pricePlan)
						u.promTail.Debugf("SideBuy price plan: %+v", pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

					case SideSell:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						u.logRus.Debug(pricePlan)
						u.promTail.Debugf("SideSell price plan: %+v", pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

					}
				}
			}
		}
	}()

	return nil
}

func (u *orderUseCase) initOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	if err := u.createFeaturesLimitOrder(pricePlan.SetSide(SideBuy), settings); err != nil {
		u.logRus.
			WithError(err).
			Error(string(debug.Stack()))
		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

		return err
	}

	if err := u.createFeaturesLimitOrder(pricePlan.SetSide(SideSell), settings); err != nil {
		u.logRus.
			WithError(err).
			Error(string(debug.Stack()))
		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

		return err
	}

	return nil
}

func (u *orderUseCase) constructLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrder {
	return &structs.FeatureOrder{
		Symbol:      settings.Symbol,
		Type:        "LIMIT",
		Side:        "BUY",
		Price:       fmt.Sprintf("%f", pricePlan.ActualPrice),
		Quantity:    fmt.Sprintf("%f", pricePlan.Status.Quantity),
		TimeInForce: "timeInForce",
	}
}

func (u *orderUseCase) constructTakeProfitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrder {
	return &structs.FeatureOrder{
		Symbol:        settings.Symbol,
		Type:          "TAKE_PROFIT_MARKET",
		Side:          "SELL",
		StopPrice:     fmt.Sprintf("%f", pricePlan.StopPriceSELL),
		WorkingType:   "MARK_PRICE",
		PositionSide:  "LONG",
		PriceProtect:  "false",
		ClosePosition: "true",
	}
}

func (u *orderUseCase) constructStopLossOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrder {
	return &structs.FeatureOrder{
		Symbol:        settings.Symbol,
		Type:          "STOP_MARKET",
		Side:          "SELL",
		StopPrice:     fmt.Sprintf("%f", pricePlan.StopPriceSELL),
		WorkingType:   "MARK_PRICE",
		PositionSide:  "LONG",
		PriceProtect:  "false",
		ClosePosition: "true",
	}
}

func (u *orderUseCase) createBatchOrders(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureBatchOrders)

	orders := []*structs.FeatureOrder{
		u.constructLimitOrder(pricePlan, settings),
		u.constructTakeProfitOrder(pricePlan, settings),
		u.constructStopLossOrder(pricePlan, settings),
	}

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
	var orderList []structs.FeatureOrder

	if err := json.Unmarshal(req, &orderList); err != nil {
		return err
	}

	for _, order := range orderList {
		quantity, err := strconv.ParseFloat(order.Quantity, 64)
		if err != nil {
			return err
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

	u.promTail.Debugf("BatchOrders Request: %s", req)

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
func (u *orderUseCase) getFeatureOrderInfo(orderID int64, symbol string) (*structs.Order, error) {
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

	var out structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *orderUseCase) createFeaturesLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	actualPrice, err := u.priceUseCase.GetPrice(pricePlan.Symbol)
	if err != nil {
		return err
	}

	switch pricePlan.Side {
	case SideBuy:
		if actualPrice < pricePlan.PriceBUY {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideBuy)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateLimitOrder newPricePlan: %+v", pricePlan)

			u.metrics[structs.MetricOrderLimitNewPricePlan].Inc()
		}
	case SideSell:
		if actualPrice > pricePlan.PriceSELL {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideSell)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateLimitOrder newPricePlan: %+v", pricePlan)

			u.metrics[structs.MetricOrderLimitNewPricePlan].Inc()
		}
	}

	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("type", "LIMIT")
	q.Set("quantity", fmt.Sprintf("%.3f", pricePlan.Status.Quantity))
	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.1f", pricePlan.PriceBUY))
	case SideSell:
		q.Set("price", fmt.Sprintf("%.1f", pricePlan.PriceSELL))
	}
	//q.Set("stopPrice", fmt.Sprintf("%.2f", stopPrice))
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
