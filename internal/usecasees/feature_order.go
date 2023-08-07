package usecasees

import (
	"binance/internal/controllers"
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/usecasees/structs"
	"binance/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"strconv"
	"time"
)

type Monitor struct {
	actualPrice     float64
	actualPriceChan chan float64

	lastOrder     *models.Order
	lastOrderChan chan *models.Order

	settings *mongoStructs.Settings
	status   *structs.Status

	ordersList     ordersList
	ordersListChan chan [3]*models.Order
}

type ordersList [3]*models.Order

func (o *ordersList) SetLimit(order *models.Order) {
	o[OrderTypeLimitID] = order
}
func (o *ordersList) SetTakeProfit(order *models.Order) {
	o[OrderTypeTakeProfitID] = order
}
func (o *ordersList) SetStopLoss(order *models.Order) {
	o[OrderTypeStopLossID] = order
}

func (o *ordersList) IsNil() bool {
	return o[OrderTypeLimitID] == nil && o[OrderTypeTakeProfitID] == nil && o[OrderTypeStopLossID] == nil
}

func (o *ordersList) Has(orderType string) bool {
	switch orderType {
	case OrderTypeLimit:
		return o[OrderTypeLimitID] != nil
	case OrderTypeTakeProfit:
		return o[OrderTypeTakeProfitID] != nil
	case OrderTypeStopLoss:
		return o[OrderTypeStopLossID] != nil
	default:
		panic("error type")
	}
}

func (o *ordersList) Get(orderType string) *models.Order {
	switch orderType {
	case OrderTypeLimit:
		return o[OrderTypeLimitID]
	case OrderTypeTakeProfit:
		return o[OrderTypeTakeProfitID]
	case OrderTypeStopLoss:
		return o[OrderTypeStopLossID]
	default:
		panic("error type")
	}
}

const chkTime = 100 * time.Millisecond

func newMonitor() *Monitor {
	return &Monitor{
		actualPriceChan: make(chan float64),
		lastOrderChan:   make(chan *models.Order),
		ordersListChan:  make(chan [3]*models.Order),
		status: &structs.Status{
			OrderTry:  1,
			SessionID: uuid.New().String(),
			Mode:      "middle",
		},
	}
}

func (m *Monitor) Update() {
	for {
		select {
		case newActualPrice := <-m.actualPriceChan:
			m.actualPrice = newActualPrice
		case newLastOrder := <-m.lastOrderChan:
			m.lastOrder = newLastOrder
		case newOrdersList := <-m.ordersListChan:
			m.ordersList = newOrdersList
		}
	}
}

func (m *Monitor) UpdateCreateOrder(u *orderUseCase) {
	for {
		if m.ordersList.Has(OrderTypeLimit) {
			if m.ordersList.Get(OrderTypeLimit).Status == OrderStatusInProgress {

				if m.ordersList.Get(OrderTypeLimit).Type != OrderTypeLimit {
					u.logRus.Panicf("error order type\n id: %d\norder: %+v", 0, m.ordersList.Get(OrderTypeLimit))
				}

				if err := u.createFeaturesLimitOrder(m.ordersList.Get(OrderTypeLimit)); err != nil {
					switch err {
					case controllers.ErrUnknownOrderSent:
						if err := u.orderRepo.Delete(m.ordersList.Get(OrderTypeLimit).ID); err != nil {
							u.logRus.
								WithField("func", "Delete").
								WithField("type", OrderTypeLimit).
								WithField("status", m.ordersList.Get(OrderTypeLimit).Status).
								WithField("orderID", m.ordersList.Get(OrderTypeLimit).ID).
								Debug(err)

							continue
						}
					}

					u.logRus.
						WithField("func", "createFeaturesMarketOrder").
						WithField("type", OrderTypeLimit).
						WithField("status", m.ordersList.Get(OrderTypeLimit).Status).
						WithField("orderID", m.ordersList.Get(OrderTypeLimit).ID).
						Debug(err)

					continue
				}

				m.ordersList.Get(OrderTypeLimit).Status = OrderStatusNew

				if err := u.orderRepo.SetStatus(m.ordersList.Get(OrderTypeLimit).ID, OrderStatusNew); err != nil {
					u.logRus.
						WithField("func", "SetStatus").
						WithField("type", OrderTypeLimit).
						WithField("status", m.ordersList.Get(OrderTypeLimit).Status).
						WithField("orderID", m.ordersList.Get(OrderTypeLimit).ID).
						Debug(err)
				}
			}
		}

		if m.ordersList.Has(OrderTypeTakeProfit) {
			if m.ordersList.Get(OrderTypeTakeProfit).Status == OrderStatusInProgress {

				if m.ordersList.Get(OrderTypeTakeProfit).Type != OrderTypeTakeProfit {
					u.logRus.Panicf("error order type\n id: %d\norder: %+v", 1, m.ordersList.Get(OrderTypeTakeProfit))
				}

				if err := u.createFeaturesLimitOrder(m.ordersList.Get(OrderTypeTakeProfit)); err != nil {
					u.logRus.
						WithField("func", "createFeatureOrder").
						WithField("type", OrderTypeTakeProfit).
						WithField("status", m.ordersList.Get(OrderTypeTakeProfit).Status).
						WithField("orderID", m.ordersList.Get(OrderTypeTakeProfit).ID).
						Debug(err)

					continue
				}

				m.ordersList.Get(OrderTypeTakeProfit).Status = OrderStatusNew

				if err := u.orderRepo.SetStatus(m.ordersList.Get(OrderTypeTakeProfit).ID, OrderStatusNew); err != nil {
					u.logRus.
						WithField("func", "SetStatus").
						WithField("type", OrderTypeTakeProfit).
						WithField("status", m.ordersList.Get(OrderTypeTakeProfit).Status).
						WithField("orderID", m.ordersList.Get(OrderTypeTakeProfit).ID).
						Debug(err)

				}
			}
		}

		if m.ordersList.Has(OrderTypeStopLoss) {
			if m.ordersList.Get(OrderTypeStopLoss).Status == OrderStatusInProgress {

				if m.ordersList.Get(OrderTypeStopLoss).Type != OrderTypeStopLoss {
					u.logRus.Panicf("error order type\n id: %d\norder: %+v", 2, m.ordersList.Get(OrderTypeStopLoss))
				}

				if err := u.createFeaturesLimitOrder(m.ordersList.Get(OrderTypeStopLoss)); err != nil {
					u.logRus.
						WithField("func", "createFeatureOrder").
						WithField("type", OrderTypeStopLoss).
						WithField("status", m.ordersList.Get(OrderTypeStopLoss).Status).
						WithField("orderID", m.ordersList.Get(OrderTypeStopLoss).ID).
						Debug(err)

					continue
				}

				m.ordersList.Get(OrderTypeStopLoss).Status = OrderStatusNew

				if err := u.orderRepo.SetStatus(m.ordersList.Get(OrderTypeStopLoss).ID, OrderStatusNew); err != nil {
					u.logRus.
						WithField("func", "SetStatus").
						WithField("type", OrderTypeStopLoss).
						WithField("status", m.ordersList.Get(OrderTypeStopLoss).Status).
						WithField("orderID", m.ordersList.Get(OrderTypeStopLoss).ID).
						Debug(err)
				}
			}
		}

		time.Sleep(chkTime)
	}
}

func (m *Monitor) UpdateOrderStatus(u *orderUseCase) {
	for {
		for _, o := range m.ordersList {
			if o == nil {
				continue
			}

			order, err := u.getFeatureOrderInfo(o.ID, o.Symbol)
			if err != nil {
				u.logRus.
					WithField("orderId", o.ID).
					WithField("func", "getFeatureOrderInfo").Debug(err)

				continue
			}

			if o.OrderID != order.OrderId {
				if err := u.orderRepo.SetOrderID(order.ClientOrderId, order.OrderId); err != nil {
					u.logRus.WithField("func", "SetOrderID").Debug(err)

					continue
				}
			}

			if o.Status != order.Status {
				if err := u.orderRepo.SetStatus(order.ClientOrderId, order.Status); err != nil {
					u.logRus.WithField("func", "SetStatus").Debug(err)

					continue
				}
			}

			switch order.Type {
			case OrderTypeLimit:
				avgPrice, err := strconv.ParseFloat(order.AvgPrice, 64)
				if err != nil {
					u.logRus.WithField("func", "ParseFloat").Debug(err)

					continue
				}

				if err := u.orderRepo.SetActualPrice(order.ClientOrderId, avgPrice); err != nil {
					u.logRus.WithField("func", "SetActualPrice").Debug(err)

					continue
				}
			}
		}

		time.Sleep(chkTime)
	}
}

func (m *Monitor) UpdateOrdersList(u *orderUseCase) {
	for {
		var out ordersList

		if m.status.SessionID == "" {
			u.logRus.Debug("SessionID is nil")
			continue
		}

		list, err := u.orderRepo.GetBySessionID(m.status.SessionID)
		if err != nil {
			u.logRus.
				WithError(err).
				Error(string(debug.Stack()))
		}

		for _, o := range list {
			switch o.Type {
			case OrderTypeLimit:
				order := o
				out.SetLimit(&order)
			case OrderTypeTakeProfit:
				order := o
				out.SetTakeProfit(&order)
			case OrderTypeStopLoss:
				order := o
				out.SetStopLoss(&order)
			}
		}

		m.ordersListChan <- out

		time.Sleep(chkTime)
	}
}

func (m *Monitor) UpdateSettings(u *orderUseCase, symbol string) {
	settings, err := u.settingsRepo.Load(symbol)
	if err != nil {
		u.logRus.
			WithError(err).
			Error(string(debug.Stack()))
	}

	m.settings = settings
}

func (m *Monitor) UpdateLastOrder(u *orderUseCase, symbol string) {
	for {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				if m.actualPrice != 0 && m.settings != nil && m.status != nil {

					m.status.
						SetSessionID(uuid.NewString()).
						SetQuantity(m.settings.Step)
					//SetMode(structs.Middle)

					pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, m.actualPrice, m.settings, m.status)

					limitOrder, err := u.storeFeaturesLimitOrder(pricePlan)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

					limitOrder.Status = OrderStatusNotFound
					m.ordersList.SetLimit(limitOrder)
				}
				continue
			default:
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
		}

		if m.settings == nil {
			continue
		}

		m.status.
			SetSessionID(order.SessionID)
		//SetQuantityByStep(m.settings.Step)
		//SetOrderTry(order.Try).
		//SetMode(structs.Middle)

		m.lastOrderChan <- order

		time.Sleep(chkTime)
	}
}

func (m *Monitor) UpdateActualPrice(u *orderUseCase, symbol string) {
	for {
		price, err := u.priceUseCase.featuresGetPrice(symbol)
		if err != nil {
			u.logRus.
				WithError(err).
				Error(string(debug.Stack()))

			continue
		}
		m.actualPriceChan <- price

		time.Sleep(chkTime)
	}
}

func (u *orderUseCase) FeaturesMonitoring(symbol string) error {
	u.logRus.Debug("Start FeaturesMonitoring")

	chkCreateOrdersFunc := func(o ordersList) bool {
		switch true {
		case o.Has(OrderTypeLimit) && o.Get(OrderTypeLimit).Status == OrderStatusFilled &&
			o.Has(OrderTypeTakeProfit) == false &&
			o.Has(OrderTypeStopLoss) == false:

			return true
		}
		return false
	}

	chkTakeProfitCancelFunc := func(o ordersList) bool {
		if o.Has(OrderTypeLimit) && o.Get(OrderTypeLimit).Status == OrderStatusFilled &&
			o.Has(OrderTypeTakeProfit) && o.Get(OrderTypeTakeProfit).Status == OrderStatusNew &&
			o.Has(OrderTypeStopLoss) && o.Get(OrderTypeStopLoss).Status == OrderStatusFilled {

			return true
		}

		return false
	}

	chkStopLossCancelFunc := func(o ordersList) bool {
		if o.Has(OrderTypeLimit) && o.Get(OrderTypeLimit).Status == OrderStatusFilled &&
			o.Has(OrderTypeTakeProfit) && o.Get(OrderTypeTakeProfit).Status == OrderStatusFilled &&
			o.Has(OrderTypeStopLoss) && o.Get(OrderTypeStopLoss).Status == OrderStatusNew {

			return true
		}
		return false
	}

	chkCreateLimitOrderTakeProfitFunc := func(o ordersList) bool {
		if o.Has(OrderTypeLimit) && o.Get(OrderTypeLimit).Status == OrderStatusFilled &&
			o.Has(OrderTypeStopLoss) && (o.Get(OrderTypeStopLoss).Status == OrderStatusCanceled || o.Get(OrderTypeStopLoss).Status == OrderStatusExpired) &&
			o.Has(OrderTypeTakeProfit) && o.Get(OrderTypeTakeProfit).Status == OrderStatusFilled {

			return true
		}
		return false
	}

	chkCreateLimitOrderStopLossFunc := func(o ordersList) bool {
		if o.Has(OrderTypeLimit) && o.Get(OrderTypeLimit).Status == OrderStatusFilled &&
			o.Has(OrderTypeTakeProfit) && (o.Get(OrderTypeTakeProfit).Status == OrderStatusCanceled || o.Get(OrderTypeTakeProfit).Status == OrderStatusExpired) &&
			o.Has(OrderTypeStopLoss) && o.Get(OrderTypeStopLoss).Status == OrderStatusFilled {

			return true
		}
		return false
	}

	m := newMonitor()
	m.UpdateSettings(u, symbol)
	go m.Update()

	go m.UpdateActualPrice(u, symbol)
	go m.UpdateLastOrder(u, symbol)
	go m.UpdateOrdersList(u)
	go m.UpdateOrderStatus(u)
	go m.UpdateCreateOrder(u)

	for {
		if m.settings == nil || m.status == nil || m.ordersList.IsNil() {
			continue
		}

		if m.status.Quantity == 0 {
			m.status.SetQuantity(m.settings.Step)
		}

		if chkCreateOrdersFunc(m.ordersList) {
			pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, m.ordersList.Get(OrderTypeLimit).Price, m.settings, m.status)

			takeProfitOrder, err := u.storeFeatureTakeProfitOrder(pricePlan, m.ordersList.Get(OrderTypeLimit), u.constructTakeProfitOrder(pricePlan, m.settings))
			if err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
			m.ordersList.SetTakeProfit(takeProfitOrder)
			stopLossOrder, err := u.storeFeatureStopLossOrder(pricePlan, m.ordersList.Get(OrderTypeLimit), m.ordersList.Get(OrderTypeTakeProfit), u.constructStopLossOrder(pricePlan, m.settings))
			if err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
			m.ordersList.SetStopLoss(stopLossOrder)
		}

		if chkTakeProfitCancelFunc(m.ordersList) {
			if _, err := u.cancelFeatureOrder(m.ordersList.Get(OrderTypeTakeProfit).OrderID, symbol); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
		}

		if chkStopLossCancelFunc(m.ordersList) {
			if _, err := u.cancelFeatureOrder(m.ordersList.Get(OrderTypeStopLoss).OrderID, symbol); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
		}

		if chkCreateLimitOrderTakeProfitFunc(m.ordersList) {

			pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, m.ordersList.Get(OrderTypeTakeProfit).Price, m.settings, m.status)
			pricePlan.Status.NewSessionID()

			marketOrder, err := u.storeFeaturesLimitOrder(pricePlan)
			if err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}

			marketOrder.Status = OrderStatusNotFound
			m.ordersList.SetLimit(marketOrder)

			_ = u.tgmController.Send(fmt.Sprintf("%s \n"+
				"Quantity:\t%.4f\n"+
				"Price:\t%.4f\n"+
				"Order: \t %s", marketOrder.Symbol, marketOrder.Quantity, marketOrder.Price, OrderTypeTakeProfit))
		}

		if chkCreateLimitOrderStopLossFunc(m.ordersList) {
			pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, m.ordersList.Get(OrderTypeStopLoss).Price, m.settings, m.status)
			pricePlan.Status.NewSessionID()

			marketOrder, err := u.storeFeaturesLimitOrder(pricePlan)
			if err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}

			marketOrder.Status = OrderStatusNotFound
			m.ordersList.SetLimit(marketOrder)

			_ = u.tgmController.Send(fmt.Sprintf("%s \n"+
				"Quantity:\t%.4f\n"+
				"Price:\t%.4f\n"+
				"Order: \t %s", marketOrder.Symbol, marketOrder.Quantity, marketOrder.Price, OrderTypeStopLoss))
		}

		time.Sleep(500 * time.Millisecond)
	}
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

func (u *orderUseCase) constructTakeProfitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrderReq {
	out := structs.FeatureOrderReq{
		Symbol:       settings.Symbol,
		Type:         OrderTypeTakeProfit,
		PriceProtect: "true",
		Quantity:     fmt.Sprintf("%.3f", pricePlan.Status.Quantity),
		//ClosePosition: "true",
	}

	switch pricePlan.Side {
	case SideBuy:
		out.Side = SideSell
		//out.StopPrice = fmt.Sprintf("%.1f", pricePlan.LowPrice)
		//out.Price = fmt.Sprintf("%.1f", pricePlan.LowPrice)
		//out.PositionSide = "LONG"
	case SideSell:
		out.Side = SideBuy
		//out.StopPrice = fmt.Sprintf("%.1f", pricePlan.HighPrice)
		//out.Price = fmt.Sprintf("%.1f", pricePlan.HighPrice)
		//out.PositionSide = "SHORT"
	}

	return &out
}

//}

func (u *orderUseCase) constructStopLossOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrderReq {
	out := structs.FeatureOrderReq{
		Symbol:       settings.Symbol,
		Type:         OrderTypeStopLoss,
		PriceProtect: "true",
		Quantity:     fmt.Sprintf("%.3f", pricePlan.Status.Quantity),
		//ClosePosition: "true",
	}

	switch pricePlan.Side {
	case SideBuy:
		out.Side = SideBuy
		//out.StopPrice = fmt.Sprintf("%.1f", pricePlan.LowPrice)
		//out.Price = fmt.Sprintf("%.1f", pricePlan.LowPrice)
		//out.PositionSide = "LONG"
	case SideSell:
		out.Side = SideSell
		//out.StopPrice = fmt.Sprintf("%.1f", pricePlan.HighPrice)
		//out.Price = fmt.Sprintf("%.1f", pricePlan.HighPrice)
		//out.PositionSide = "SHORT"
	}

	return &out
}

func (u *orderUseCase) storeFeaturesLimitOrder(pricePlan *structs.PricePlan) (*models.Order, error) {
	o := models.Order{
		ID:           uuid.NewString(),
		SessionID:    pricePlan.Status.SessionID,
		Try:          pricePlan.Status.OrderTry,
		ActualPrice:  pricePlan.ActualPrice,
		Symbol:       pricePlan.Symbol,
		Side:         pricePlan.Side,
		Type:         "LIMIT",
		Quantity:     pricePlan.Status.Quantity,
		Status:       OrderStatusInProgress,
		StopPrice:    0,
		PositionSide: pricePlan.PositionSide,
	}

	depth, err := u.priceUseCase.GetDepthInfo(pricePlan.Symbol)
	if err != nil {
		return nil, err
	}

	//switch pricePlan.Symbol {
	//case BTCBUSD:
	//	o.Side = SideSell
	//	o.PositionSide = "SHORT"
	//	o.Price = depth.BidsMaxPrice + 0.1
	//case ETHBUSD:
	//	if depth.AsksSum < depth.BidsSum {
	//		//if pricePlan.ActualPrice > pricePlan.AvgPrice {
	//		o.Side = SideBuy
	//		o.PositionSide = "LONG"
	//		o.Price = depth.AsksMaxPrice - 0.1
	//	} else {
	//		//if pricePlan.ActualPrice <= pricePlan.AvgPrice {
	//		o.Side = SideSell
	//		o.PositionSide = "SHORT"
	//		o.Price = depth.BidsMaxPrice + 0.1
	//	}
	//case BNBBUSD:
	//	o.Side = SideBuy
	//	o.PositionSide = "LONG"
	//	o.Price = depth.AsksMaxPrice - 0.1
	//}

	if depth.BidsSum > depth.AsksSum {
		//if pricePlan.ActualPrice > pricePlan.AvgPrice {
		o.Side = SideBuy
		o.PositionSide = "LONG"
		o.Price = depth.AsksMaxPrice - 0.1
	} else {
		//if pricePlan.ActualPrice <= pricePlan.AvgPrice {
		o.Side = SideSell
		o.PositionSide = "SHORT"
		o.Price = depth.BidsMaxPrice + 0.1
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return nil, err
	}

	return &o, nil
}

func (u *orderUseCase) storeFeatureTakeProfitOrder(pricePlan *structs.PricePlan, limitOrder *models.Order, order *structs.FeatureOrderReq) (*models.Order, error) {
	o := models.Order{
		ID:          uuid.NewString(),
		SessionID:   pricePlan.Status.SessionID,
		Try:         pricePlan.Status.OrderTry,
		ActualPrice: pricePlan.ActualPrice,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Type:        order.Type,
		Quantity:    pricePlan.Status.Quantity,
		Status:      OrderStatusInProgress,
	}

	o.PositionSide = limitOrder.PositionSide

	depth, err := u.priceUseCase.GetDepthInfo(pricePlan.Symbol)
	if err != nil {
		return nil, err
	}

	switch o.PositionSide {
	case "LONG":
		o.Side = SideSell
		o.Price = depth.AsksMaxPrice - 0.1
		o.StopPrice = depth.AsksMaxPrice - 0.1

	case "SHORT":
		o.Side = SideBuy
		o.Price = depth.BidsMaxPrice + 0.1
		o.StopPrice = depth.BidsMaxPrice + 0.1
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return nil, err
	}

	return &o, nil
}

func (u *orderUseCase) storeFeatureStopLossOrder(pricePlan *structs.PricePlan, limitOrder, takeOrder *models.Order, order *structs.FeatureOrderReq) (*models.Order, error) {
	delta := limitOrder.Price - takeOrder.Price
	if delta < 0 {
		delta *= -1
	}

	o := models.Order{
		ID:          uuid.NewString(),
		SessionID:   pricePlan.Status.SessionID,
		Try:         pricePlan.Status.OrderTry,
		ActualPrice: pricePlan.ActualPrice,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Type:        order.Type,
		Quantity:    pricePlan.Status.Quantity,
		Status:      OrderStatusInProgress,
	}

	o.PositionSide = limitOrder.PositionSide

	switch o.PositionSide {
	case "LONG":
		o.Side = SideSell
		o.Price = limitOrder.Price - delta + 0.1
		o.StopPrice = limitOrder.Price - delta + 0.1

	case "SHORT":
		o.Side = SideBuy
		o.Price = limitOrder.Price + delta - 0.1
		o.StopPrice = limitOrder.Price + delta - 0.1
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return nil, err
	}

	return &o, nil
}

func (u *orderUseCase) ticker24hr(symbol string) (*PriceChangeStatistics, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featureTicker24hr)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	resp, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out PriceChangeStatistics
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

//func (u *orderUseCase) createFeatureOrder(order *models.Order) error {
//	baseURL, err := url.Parse(u.url)
//	if err != nil {
//		return err
//	}
//
//	baseURL.Path = path.Join(featureOrder)
//
//	q := baseURL.Query()
//	q.Set("symbol", order.Symbol)
//	q.Set("type", order.Type)
//	q.Set("timeInForce", "FOK")
//	q.Set("priceProtect", "true")
//	q.Set("quantity", fmt.Sprintf("%.3f", order.Quantity))
//	//q.Set("closePosition", "true")
//	q.Set("newClientOrderId", order.ID)
//
//	q.Set("side", order.Side)
//	q.Set("stopPrice", fmt.Sprintf("%.2f", order.StopPrice))
//	q.Set("price", fmt.Sprintf("%.2f", order.StopPrice))
//
//	switch order.Side {
//	case SideBuy:
//		q.Set("positionSide", "SHORT")
//	case SideSell:
//		q.Set("positionSide", "LONG")
//	}
//
//	q.Set("recvWindow", "60000")
//	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))
//
//	sig := u.cryptoController.GetSignature(q.Encode())
//	q.Set("signature", sig)
//
//	baseURL.RawQuery = q.Encode()
//
//	resp, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
//	if err != nil {
//		return err
//	}
//
//	var respOrder structs.FeatureOrderResp
//	if err := json.Unmarshal(resp, &respOrder); err != nil {
//		return err
//	}
//
//	if respOrder.OrderId == 0 {
//		return fmt.Errorf("err OrderId == 0 : %s", resp)
//	}
//
//	return nil
//}

//
//func (u *orderUseCase) createBatchOrders(pricePlan *structs.PricePlan, orders []structs.FeatureOrderReq) error {
//	for _, order := range orders {
//		o := models.Order{
//			ID:          order.NewClientOrderId,
//			SessionID:   pricePlan.Status.SessionID,
//			Try:         pricePlan.Status.OrderTry,
//			ActualPrice: pricePlan.ActualPrice,
//			Symbol:      order.Symbol,
//			Side:        order.Side,
//			Type:        order.Type,
//		}
//
//		if err := u.orderRepo.Store(&o); err != nil {
//			return err
//		}
//	}
//
//	baseURL, err := url.Parse(u.url)
//	if err != nil {
//		return err
//	}
//
//	baseURL.Path = path.Join(featureBatchOrders)
//
//	batchOrders, err := json.Marshal(orders)
//	if err != nil {
//		return err
//	}
//
//	q := baseURL.Query()
//	q.Set("batchOrders", fmt.Sprintf("%s", batchOrders))
//	q.Set("recvWindow", "60000")
//	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))
//
//	sig := u.cryptoController.GetSignature(q.Encode())
//	q.Set("signature", sig)
//
//	baseURL.RawQuery = q.Encode()
//
//	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
//	if err != nil {
//		return err
//	}
//
//	u.logRus.Debugf("createBatchOrders Req: %s", req)
//	u.logRus.Debugf("createBatchOrders PricePlan: %+v", pricePlan)
//	u.logRus.Debugf("createBatchOrders Orders: %+v", orders)
//
//	var orderList []structs.FeatureOrderResp
//	if err := json.Unmarshal(req, &orderList); err != nil {
//		return errors.New(fmt.Sprintf("%+v %s", err, req))
//	}
//
//	for _, o := range orderList {
//		if o.OrderId == 0 {
//			if err := u.orderRepo.SetStatus(o.ClientOrderId, OrderStatusError); err != nil {
//				return err
//			}
//		}
//	}
//
//	return nil
//}

func (u *orderUseCase) getFeaturePositionInfo(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse(u.url)
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
func (u *orderUseCase) getFeatureOrderInfo(orderID string, symbol string) (*structs.FeatureOrderResp, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("origClientOrderId", orderID)
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		u.logRus.Debug(err)

		return nil, err
	}

	var respOrderInfo structs.FeatureOrderResp

	if err := json.Unmarshal(req, &respOrderInfo); err != nil {
		u.logRus.Debug(err)

		return nil, err
	}

	if respOrderInfo.OrderId == 0 {
		if err := u.orderRepo.SetStatus(orderID, OrderStatusError); err != nil {
			u.logRus.Debug(err)

			return nil, err
		}
	}

	return &respOrderInfo, nil

}
func (u *orderUseCase) cancelFeatureOrder(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse(u.url)
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

//func (u *orderUseCase) storeFeaturesMarketOrder(pricePlan *structs.PricePlan) (*models.Order, error) {
//	o := models.Order{
//		ID:          uuid.NewString(),
//		SessionID:   pricePlan.Status.SessionID,
//		Try:         pricePlan.Status.OrderTry,
//		ActualPrice: pricePlan.ActualPrice,
//		Symbol:      pricePlan.Symbol,
//		Side:        pricePlan.Side,
//		Type:        "MARKET",
//		Quantity:    pricePlan.Status.Quantity,
//		Status:      OrderStatusInProgress,
//		StopPrice:   0,
//		Price:       0,
//	}
//
//	if err := u.orderRepo.Store(&o); err != nil {
//		return nil, err
//	}
//
//	return &o, nil
//}

func (u *orderUseCase) createFeaturesLimitOrder(order *models.Order) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	//u.logRus.Debug("createFeaturesLimitOrder", order)

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()

	q.Set("type", order.Type)
	q.Set("symbol", order.Symbol)
	q.Set("price", fmt.Sprintf("%.1f", order.Price))
	q.Set("quantity", fmt.Sprintf("%.3f", order.Quantity))
	q.Set("side", order.Side)
	q.Set("positionSide", order.PositionSide)

	q.Set("newClientOrderId", order.ID)
	q.Set("recvWindow", "60000")

	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	switch order.Type {
	case OrderTypeTakeProfit:
		q.Set("stopPrice", fmt.Sprintf("%.1f", order.StopPrice))
	case OrderTypeStopLoss:
		q.Set("stopPrice", fmt.Sprintf("%.1f", order.StopPrice))
	case OrderTypeLimit:
		q.Set("timeInForce", "GTC")
	}

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	resp, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	var respOrder structs.FeatureOrderResp
	if err := json.Unmarshal(resp, &respOrder); err != nil {
		return err
	}

	if respOrder.OrderId == 0 {
		if err := u.orderRepo.SetStatus(order.ID, OrderStatusError); err != nil {
			return err
		}
	}

	return nil
}

//func (u *orderUseCase) createFeaturesMarketOrder(order *models.Order) error {
//	baseURL, err := url.Parse(u.url)
//	if err != nil {
//		return err
//	}
//
//	baseURL.Path = path.Join(featureOrder)
//
//	q := baseURL.Query()
//	q.Set("symbol", order.Symbol)
//	switch order.Side {
//	case SideBuy:
//		q.Set("side", SideBuy)
//		q.Set("positionSide", "LONG")
//	case SideSell:
//		q.Set("side", SideSell)
//		q.Set("positionSide", "SHORT")
//	}
//
//	q.Set("newClientOrderId", order.ID)
//	q.Set("type", "MARKET")
//	q.Set("quantity", fmt.Sprintf("%.3f", order.Quantity))
//	q.Set("recvWindow", "60000")
//	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))
//
//	sig := u.cryptoController.GetSignature(q.Encode())
//	q.Set("signature", sig)
//
//	baseURL.RawQuery = q.Encode()
//
//	resp, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
//	if err != nil {
//		return err
//	}
//
//	var respOrder structs.FeatureOrderResp
//	if err := json.Unmarshal(resp, &respOrder); err != nil {
//		return err
//	}
//
//	if respOrder.OrderId == 0 {
//		if err := u.orderRepo.SetStatus(order.ID, OrderStatusError); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}

func (u *priceUseCase) featuresGetPrice(symbol string) (float64, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return 0, err
	}

	baseURL.Path = path.Join(featureSymbolPrice)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
	if err != nil {
		u.logger.Debug(err)

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
