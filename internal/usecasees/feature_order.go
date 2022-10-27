package usecasees

import (
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/usecasees/structs"
	"binance/models"
	"database/sql"
	"encoding/json"
	"errors"
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
	actualPrice  float64
	lastOrder    *models.Order
	settings     *mongoStructs.Settings
	status       *structs.Status
	ordersList   [3]*models.Order
	ordersStatus *structs.FeatureOrdersStatus
}

const chkTime = 250 * time.Millisecond

func newMonitor() *Monitor {
	return &Monitor{
		ordersStatus: &structs.FeatureOrdersStatus{
			MarketOrder: structs.OrderStatus{
				Status: OrderStatusNotFound,
			},
			TakeProfitOrder: structs.OrderStatus{
				Status: OrderStatusNotFound,
			},
			StopLossOrder: structs.OrderStatus{
				Status: OrderStatusNotFound,
			},
		},
		status: &structs.Status{
			OrderTry:  1,
			SessionID: uuid.New().String(),
			Mode:      "middle",
		},
	}
}

func (m *Monitor) UpdateOrderStatus(u *orderUseCase) {
	for {
		if m.ordersList[0] == nil {
			u.logRus.Debug("ordersList is nil")
			continue
		}

		ordersStatus, err := u.fillFeatureOrdersStatus(m.ordersList)
		if err != nil {
			u.logRus.
				WithError(err).
				Error(string(debug.Stack()))
		}

		m.ordersStatus = ordersStatus

		time.Sleep(chkTime)
	}
}

func (m *Monitor) UpdateOrdersList(u *orderUseCase) {
	for {
		var out [3]*models.Order
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
			case OrderTypeMarket:
				out[0] = &o
			case OrderTypeTakeProfit:
				out[1] = &o
			case OrderTypeStopLoss:
				out[2] = &o
			}
		}

		m.ordersList = out

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
	m.status.Quantity = settings.Step
}

func (m *Monitor) UpdateLastOrder(u *orderUseCase, symbol string) {
	for {
		order, err := u.orderRepo.GetLast(symbol)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				if m.actualPrice != 0 && m.settings != nil && m.status != nil {

					m.ordersStatus.MarketOrder.Status = OrderStatusInProgress

					pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, m.actualPrice, m.settings, m.status).SetSide(SideBuy)

					if err := u.storeFeaturesMarketOrder(pricePlan); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
					}

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
			SetSessionID(order.SessionID).
			SetOrderTry(order.Try).
			SetQuantityByStep(m.settings.Step).
			SetMode(structs.Middle)

		m.lastOrder = order

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
		}

		m.actualPrice = price

		time.Sleep(chkTime)
	}
}

func (u *orderUseCase) FeaturesMonitoring(symbol string) error {
	u.logRus.Debug("Start FeaturesMonitoring")

	chkCreateOrdersFunc := func(f *structs.FeatureOrdersStatus) bool {
		if f.MarketOrder.Status == OrderStatusInProgress ||
			f.StopLossOrder.Status == OrderStatusInProgress ||
			f.TakeProfitOrder.Status == OrderStatusInProgress {

			return false
		}

		switch true {
		case f.MarketOrder.Status == OrderStatusFilled &&
			f.TakeProfitOrder.Status == OrderStatusNotFound &&
			f.StopLossOrder.Status == OrderStatusNotFound:

			return true
		}
		return false
	}

	chkTakeProfitCancelFunc := func(f *structs.FeatureOrdersStatus) bool {
		if f.MarketOrder.Status == OrderStatusInProgress ||
			f.StopLossOrder.Status == OrderStatusInProgress ||
			f.TakeProfitOrder.Status == OrderStatusInProgress {

			return false
		}

		switch true {
		case f.MarketOrder.Status == OrderStatusFilled &&
			f.StopLossOrder.Status == OrderStatusFilled &&
			f.TakeProfitOrder.Status == OrderStatusNew:
			return true
		}
		return false
	}

	chkStopLossCancelFunc := func(f *structs.FeatureOrdersStatus) bool {
		if f.MarketOrder.Status == OrderStatusInProgress ||
			f.StopLossOrder.Status == OrderStatusInProgress ||
			f.TakeProfitOrder.Status == OrderStatusInProgress {

			return false
		}

		switch true {
		case f.MarketOrder.Status == OrderStatusFilled &&
			f.TakeProfitOrder.Status == OrderStatusFilled &&
			f.StopLossOrder.Status == OrderStatusNew:
			return true
		}
		return false
	}

	chkCreateLimitOrderTakeProfitFunc := func(f *structs.FeatureOrdersStatus) bool {
		if f.MarketOrder.Status == OrderStatusInProgress ||
			f.StopLossOrder.Status == OrderStatusInProgress ||
			f.TakeProfitOrder.Status == OrderStatusInProgress {

			return false
		}

		switch true {
		case f.MarketOrder.Status == OrderStatusFilled &&
			(f.StopLossOrder.Status == OrderStatusCanceled || f.StopLossOrder.Status == OrderStatusExpired) &&
			f.TakeProfitOrder.Status == OrderStatusFilled:

			return true
		}
		return false
	}

	chkCreateLimitOrderStopLossFunc := func(f *structs.FeatureOrdersStatus) bool {
		if f.MarketOrder.Status == OrderStatusInProgress ||
			f.StopLossOrder.Status == OrderStatusInProgress ||
			f.TakeProfitOrder.Status == OrderStatusInProgress {

			return false
		}

		switch true {
		case f.MarketOrder.Status == OrderStatusFilled &&
			f.StopLossOrder.Status == OrderStatusFilled &&
			(f.TakeProfitOrder.Status == OrderStatusCanceled || f.TakeProfitOrder.Status == OrderStatusExpired):

			return true
		}
		return false
	}

	m := newMonitor()
	m.UpdateSettings(u, symbol)

	go m.UpdateLastOrder(u, symbol)
	go m.UpdateActualPrice(u, symbol)
	go m.UpdateOrdersList(u)
	go m.UpdateOrderStatus(u)

	for {
		//u.logRus.Debug("actualPrice ", m.actualPrice)
		//u.logRus.Debug("lastOrder ", m.lastOrder)
		//u.logRus.Debug("settings ", m.settings)
		//u.logRus.Debug("status ", m.status)
		//u.logRus.Debug("ordersList ", m.ordersList)
		//u.logRus.Debug("ordersStatus ", m.ordersStatus)

		if m.settings == nil || m.status == nil || m.ordersStatus == nil {
			continue
		}

		if chkCreateOrdersFunc(m.ordersStatus) {
			if m.ordersList[1] != nil && m.ordersList[2] != nil {
				continue
			}

			pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, m.ordersStatus.MarketOrder.ActualPrice, m.settings, m.status)

			switch m.ordersStatus.MarketOrder.Side {
			case SideBuy:
				pricePlan.SetSide(SideBuy)
			case SideSell:
				pricePlan.SetSide(SideSell)
			}

			m.ordersStatus.StopLossOrder.Status = OrderStatusInProgress
			m.ordersStatus.TakeProfitOrder.Status = OrderStatusInProgress

			if err := u.storeFeatureOrder(pricePlan, u.constructTakeProfitOrder(pricePlan, m.settings)); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}

			if err := u.storeFeatureOrder(pricePlan, u.constructStopLossOrder(pricePlan, m.settings)); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
		}

		if chkTakeProfitCancelFunc(m.ordersStatus) {
			u.logRus.Debugf("chkTakeProfitCancel: %+v", m.ordersStatus)

			m.ordersStatus.TakeProfitOrder.Status = OrderStatusInProgress

			if _, err := u.cancelFeatureOrder(m.ordersStatus.TakeProfitOrder.OrderId, symbol); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
		}

		if chkStopLossCancelFunc(m.ordersStatus) {
			u.logRus.Debugf("chkStopLossCancel: %+v", m.ordersStatus)

			m.ordersStatus.StopLossOrder.Status = OrderStatusInProgress

			if _, err := u.cancelFeatureOrder(m.ordersStatus.StopLossOrder.OrderId, symbol); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}
		}

		if chkCreateLimitOrderTakeProfitFunc(m.ordersStatus) {
			m.status.Reset(m.settings.Step)

			pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, m.ordersStatus.TakeProfitOrder.ActualPrice, m.settings, m.status)

			switch m.ordersStatus.MarketOrder.Side {
			case SideBuy:
				pricePlan.SetSide(SideSell)
			case SideSell:
				pricePlan.SetSide(SideBuy)
			}

			m.ordersStatus.MarketOrder.Status = OrderStatusInProgress

			if err := u.storeFeaturesMarketOrder(pricePlan); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}

			u.logRus.Debugf("chkLimit pricePlan %+v", pricePlan)
		}

		if chkCreateLimitOrderStopLossFunc(m.ordersStatus) {
			m.status.
				NewSessionID().
				AddOrderTry(1).
				SetQuantityByStep(m.settings.Step)

			pricePlan := u.fillPricePlan(OrderTypeBatch, symbol, m.ordersStatus.StopLossOrder.ActualPrice, m.settings, m.status)

			switch m.ordersStatus.MarketOrder.Side {
			case SideBuy:
				pricePlan.SetSide(SideSell)
			case SideSell:
				pricePlan.SetSide(SideBuy)
			}

			m.ordersStatus.MarketOrder.Status = OrderStatusInProgress

			if err := u.storeFeaturesMarketOrder(pricePlan); err != nil {
				u.logRus.
					WithError(err).
					Error(string(debug.Stack()))
			}

			u.logRus.Debugf("chkLimit pricePlan %+v", pricePlan)
		}

		time.Sleep(chkTime)
	}
}

func (u *orderUseCase) fillFeatureOrdersStatus(oList [3]*models.Order) (*structs.FeatureOrdersStatus, error) {
	var out structs.FeatureOrdersStatus

	out.MarketOrder.Status = OrderStatusNotFound
	out.TakeProfitOrder.Status = OrderStatusNotFound
	out.StopLossOrder.Status = OrderStatusNotFound

	for _, o := range oList {
		if o == nil {
			continue
		}

		order, err := u.getFeatureOrderInfo(o.ID, o.Symbol)
		if err != nil {
			return nil, err
		}

		if o.OrderID != order.OrderId {
			if err := u.orderRepo.SetOrderID(order.ClientOrderId, order.OrderId); err != nil {
				return nil, err
			}

			o.OrderID = order.OrderId
		}

		if o.Status != order.Status {
			if err := u.orderRepo.SetStatus(order.ClientOrderId, order.Status); err != nil {
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

func (u *orderUseCase) constructTakeProfitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrderReq {
	out := structs.FeatureOrderReq{
		NewClientOrderId: uuid.NewString(),
		Symbol:           settings.Symbol,
		Type:             OrderTypeTakeProfit,
		PriceProtect:     "true",
		Quantity:         fmt.Sprintf("%.3f", pricePlan.Status.Quantity),
		ClosePosition:    "true",
	}

	switch pricePlan.Side {
	case SideBuy:
		out.Side = SideSell
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.PriceSELL)
		out.PositionSide = "LONG"
	case SideSell:
		out.Side = SideBuy
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.PriceBUY)
		out.PositionSide = "SHORT"
	}

	return &out
}
func (u *orderUseCase) constructStopLossOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) *structs.FeatureOrderReq {
	out := structs.FeatureOrderReq{
		NewClientOrderId: uuid.NewString(),
		Symbol:           settings.Symbol,
		Type:             OrderTypeStopLoss,
		PriceProtect:     "true",
		Quantity:         fmt.Sprintf("%.3f", pricePlan.Status.Quantity),
		ClosePosition:    "true",
	}

	switch pricePlan.Side {
	case SideBuy:
		out.Side = SideSell
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.StopPriceSELL)
		out.PositionSide = "LONG"
	case SideSell:
		out.Side = SideBuy
		out.StopPrice = fmt.Sprintf("%.1f", pricePlan.StopPriceBUY)
		out.PositionSide = "SHORT"
	}
	return &out
}

func (u *orderUseCase) storeFeatureOrder(pricePlan *structs.PricePlan, order *structs.FeatureOrderReq) error {
	quantity, err := strconv.ParseFloat(order.Quantity, 64)
	if err != nil {
		return err
	}

	stopPrice, err := strconv.ParseFloat(order.StopPrice, 64)
	if err != nil {
		return err
	}

	o := models.Order{
		ID:          order.NewClientOrderId,
		SessionID:   pricePlan.Status.SessionID,
		Try:         pricePlan.Status.OrderTry,
		ActualPrice: pricePlan.ActualPrice,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Type:        order.Type,
		Quantity:    quantity,
		StopPrice:   stopPrice,
		Status:      OrderStatusInProgress,
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return err
	}

	if err := u.createFeatureOrder(&o); err != nil {
		u.logRus.Debug(err)
	}

	return nil
}

func (u *orderUseCase) createFeatureOrder(order *models.Order) error {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", order.Symbol)
	q.Set("type", order.Type)
	q.Set("priceProtect", "true")
	q.Set("quantity", fmt.Sprintf("%.3f", order.Quantity))
	q.Set("closePosition", "true")
	q.Set("newClientOrderId", order.ID)

	q.Set("side", order.Side)
	q.Set("stopPrice", fmt.Sprintf("%.2f", order.StopPrice))

	switch order.Side {
	case SideBuy:
		q.Set("positionSide", "SHORT")
	case SideSell:
		q.Set("positionSide", "LONG")
	}

	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	resp, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		u.logRus.Debug(err)

		return u.createFeatureOrder(order)
	}

	var respOrder structs.FeatureOrderResp
	if err := json.Unmarshal(resp, &respOrder); err != nil {
		return err
	}

	if respOrder.OrderId == 0 {
		return fmt.Errorf("err OrderId == 0 : %s", resp)
	}

	return nil
}

func (u *orderUseCase) createBatchOrders(pricePlan *structs.PricePlan, orders []structs.FeatureOrderReq) error {
	for _, order := range orders {
		o := models.Order{
			ID:          order.NewClientOrderId,
			SessionID:   pricePlan.Status.SessionID,
			Try:         pricePlan.Status.OrderTry,
			ActualPrice: pricePlan.ActualPrice,
			Symbol:      order.Symbol,
			Side:        order.Side,
			Type:        order.Type,
		}

		if err := u.orderRepo.Store(&o); err != nil {
			return err
		}
	}

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

	u.logRus.Debugf("createBatchOrders Req: %s", req)
	u.logRus.Debugf("createBatchOrders PricePlan: %+v", pricePlan)
	u.logRus.Debugf("createBatchOrders Orders: %+v", orders)

	var orderList []structs.FeatureOrderResp
	if err := json.Unmarshal(req, &orderList); err != nil {
		return errors.New(fmt.Sprintf("%+v %s", err, req))
	}

	for _, o := range orderList {
		if o.OrderId == 0 {
			if err := u.orderRepo.SetStatus(o.ClientOrderId, OrderStatusError); err != nil {
				return err
			}
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
func (u *orderUseCase) getFeatureOrderInfo(orderID string, symbol string) (*structs.FeatureOrderResp, error) {
	baseURL, err := url.Parse(featureURL)
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

		return u.getFeatureOrderInfo(orderID, symbol)
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
		u.logRus.Debug(err)

		return u.cancelFeatureOrder(orderID, symbol)
	}

	var out structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *orderUseCase) storeFeaturesMarketOrder(pricePlan *structs.PricePlan) error {
	o := models.Order{
		ID:          uuid.NewString(),
		SessionID:   pricePlan.Status.SessionID,
		Try:         pricePlan.Status.OrderTry,
		ActualPrice: pricePlan.ActualPrice,
		Symbol:      pricePlan.Symbol,
		Side:        pricePlan.Side,
		Type:        "MARKET",
		Quantity:    pricePlan.Status.Quantity,
		Status:      OrderStatusInProgress,
		StopPrice:   0,
		Price:       0,
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return err
	}

	if err := u.createFeaturesMarketOrder(&o); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) createFeaturesMarketOrder(order *models.Order) error {
	baseURL, err := url.Parse(featureURL)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(featureOrder)

	q := baseURL.Query()
	q.Set("symbol", order.Symbol)
	switch order.Side {
	case SideBuy:
		q.Set("side", SideBuy)
		q.Set("positionSide", "LONG")
	case SideSell:
		q.Set("side", SideSell)
		q.Set("positionSide", "SHORT")
	}
	q.Set("newClientOrderId", order.ID)
	q.Set("type", "MARKET")
	q.Set("quantity", fmt.Sprintf("%.3f", order.Quantity))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	u.logRus.Debugf("baseURL: %+v", baseURL.String())

	resp, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		u.logRus.Debug(err)

		return u.createFeaturesMarketOrder(order)
	}
	u.logRus.Debugf("createFeaturesMarketOrder Req: %s", resp)

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
		u.logger.Debug(err)

		return u.featuresGetPrice(symbol)
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
