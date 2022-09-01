package structs

import (
	"binance/internal/controllers"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
)

var ErrTheRelationshipOfThePrices = errors.New("the relationship of the prices for the orders is not correct")

type Status struct {
	OrderTry  int
	Quantity  float64
	SessionID string
}

func (s *Status) Reset(v float64) {
	s.OrderTry = 1
	s.Quantity = v
	s.SessionID = uuid.New().String()
}

func (s *Status) SetOrderTry(v int) *Status {
	s.OrderTry = v

	return s
}

func (s *Status) AddOrderTry(v int) *Status {
	s.OrderTry += v

	return s
}

func (s *Status) SetQuantity(v float64) *Status {
	s.Quantity = v

	return s
}

func (s *Status) AddQuantity(v float64) *Status {
	s.Quantity = v * math.Pow(2, float64(s.OrderTry-1))

	return s
}

func (s *Status) SetSessionID(v string) *Status {
	s.SessionID = v

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

type LimitOrder struct {
	Symbol              string        `json:"symbol"`
	OrderID             int64         `json:"orderId"`
	OrderListID         int           `json:"orderListId"`
	ClientOrderID       string        `json:"clientOrderId"`
	TransactTime        int64         `json:"transactTime"`
	Price               float64       `json:"price"`
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
	Quantity               float64
	ActualPrice            float64
	ActualPricePercent     float64
	ActualStopPricePercent float64
	StopPriceBUY           float64
	StopPriceSELL          float64
	PriceBUY               float64
	PriceSELL              float64
	Status                 *Status
}

func (p *PricePlan) SetSide(s string) *PricePlan {
	p.Side = s
	return p
}
