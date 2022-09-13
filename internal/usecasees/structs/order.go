package structs

import (
	"binance/internal/controllers"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
)

type Mode string
type MetricConst string

const (
	MetricOrderComplete            MetricConst = "order_complete"
	MetricOrderStopLossLimitFilled MetricConst = "order_stop_loss_limit_filled"
	MetricOrderLimitMaker          MetricConst = "order_limit_maker_filled"
	MetricOrderOSONewPricePlan     MetricConst = "order_oso_new_price_plan"
	MetricOrderLimitNewPricePlan   MetricConst = "order_limit_new_price_plan"

	Middle Mode = "MIDDLE"
	UpDown Mode = "UP_DOWN"
)

func (s MetricConst) ToString() string {
	return fmt.Sprintf("%s", s)
}

var ErrTheRelationshipOfThePrices = errors.New("the relationship of the prices for the orders is not correct")

type Status struct {
	OrderTry  int
	Quantity  float64
	SessionID string
	Mode      Mode
}

func (s *Status) Reset(v float64) {
	s.OrderTry = 1
	s.Quantity = v
	s.SessionID = uuid.New().String()
}

func (s *Status) NewSessionID() *Status {
	s.SessionID = uuid.New().String()
	return s
}

func (s *Status) SetOrderTry(v int) *Status {
	s.OrderTry = v

	return s
}

func (s *Status) AddOrderTry(v int) *Status {
	s.OrderTry += v

	return s
}

func (s *Status) SetQuantityByStep(v float64) *Status {
	//s.Quantity = v * math.Pow(2, float64(s.OrderTry-1))
	s.Quantity = v * math.Pow(2.5, float64(s.OrderTry-1))

	return s
}

func (s *Status) SetSessionID(v string) *Status {
	s.SessionID = v

	return s
}

func (s *Status) SetMode(v Mode) *Status {
	s.Mode = v

	return s
}

type Order struct {
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

type FeatureOrderResp struct {
	OrderId       int64  `json:"orderId,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
	Status        string `json:"status,omitempty"`
	ClientOrderId string `json:"clientOrderId,omitempty"`
	Price         string `json:"price,omitempty"`
	Quantity      string `json:"quantity,omitempty"`
	AvgPrice      string `json:"avgPrice,omitempty"`
	OrigQty       string `json:"origQty,omitempty"`
	ExecutedQty   string `json:"executedQty,omitempty"`
	CumQuote      string `json:"cumQuote,omitempty"`
	TimeInForce   string `json:"timeInForce,omitempty"`
	Type          string `json:"type,omitempty"`
	ReduceOnly    bool   `json:"reduceOnly,omitempty"`
	ClosePosition bool   `json:"closePosition,omitempty"`
	Side          string `json:"side,omitempty"`
	PositionSide  string `json:"positionSide,omitempty"`
	StopPrice     string `json:"stopPrice,omitempty"`
	WorkingType   string `json:"workingType,omitempty"`
	PriceProtect  bool   `json:"priceProtect,omitempty"`
	OrigType      string `json:"origType,omitempty"`
	Time          int64  `json:"time,omitempty"`
	UpdateTime    int64  `json:"updateTime,omitempty"`
}
type FeatureOrderReq struct {
	OrderId       int64  `json:"orderId,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
	Status        string `json:"status,omitempty"`
	ClientOrderId string `json:"clientOrderId,omitempty"`
	Price         string `json:"price,omitempty"`
	Quantity      string `json:"quantity,omitempty"`
	AvgPrice      string `json:"avgPrice,omitempty"`
	OrigQty       string `json:"origQty,omitempty"`
	ExecutedQty   string `json:"executedQty,omitempty"`
	CumQuote      string `json:"cumQuote,omitempty"`
	TimeInForce   string `json:"timeInForce,omitempty"`
	Type          string `json:"type,omitempty"`
	ReduceOnly    string `json:"reduceOnly,omitempty"`
	ClosePosition string `json:"closePosition,omitempty"`
	Side          string `json:"side,omitempty"`
	PositionSide  string `json:"positionSide,omitempty"`
	StopPrice     string `json:"stopPrice,omitempty"`
	WorkingType   string `json:"workingType,omitempty"`
	PriceProtect  string `json:"priceProtect,omitempty"`
	OrigType      string `json:"origType,omitempty"`
	Time          int64  `json:"time,omitempty"`
	UpdateTime    int64  `json:"updateTime,omitempty"`
}

type LimitOrder struct {
	Symbol              string        `json:"symbol"`
	OrderID             int64         `json:"orderId"`
	OrderListID         int           `json:"orderListId"`
	ClientOrderID       string        `json:"clientOrderId"`
	TransactTime        int64         `json:"transactTime"`
	Price               string        `json:"price"`
	OrigQty             string        `json:"origQty"`
	ExecutedQty         string        `json:"executedQty"`
	CummulativeQuoteQty string        `json:"cummulativeQuoteQty"`
	Status              string        `json:"status"`
	TimeInForce         string        `json:"timeInForce"`
	Type                string        `json:"type"`
	Side                string        `json:"side"`
	Fills               []interface{} `json:"fills"`
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

type Err struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Err) Send(tgm controllers.TgmCtrl) error {
	if err := tgm.Send(fmt.Sprintf("[ Err createOCOOrder ]\n"+
		"Code:\t%d\n"+
		"Msg:\t%s",
		e.Code,
		e.Msg,
	)); err != nil {
		return err
	}

	return nil
}

type PricePlan struct {
	Symbol                 string
	Side                   string
	ActualPrice            float64
	ActualPricePercent     float64
	ActualStopPricePercent float64
	StopPriceBUY           float64
	StopPriceSELL          float64
	PriceBUY               float64
	PriceSELL              float64
	SafeDelta              float64
	Status                 *Status
}

func (p *PricePlan) SetSide(s string) *PricePlan {
	p.Side = s
	return p
}

type FeatureOrdersStatus struct {
	MarketOrder     OrderStatus
	TakeProfitOrder OrderStatus
	StopLossOrder   OrderStatus
}

type OrderStatus struct {
	ActualPrice float64
	OrderId     int64
	Status      string
	Side        string
}
