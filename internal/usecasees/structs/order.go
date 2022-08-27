package structs

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

type PricePlan struct {
	Quantity               float64
	ActualPrice            float64
	ActualPricePercent     float64
	ActualStopPricePercent float64
	StopPriceBUY           float64
	StopPriceSELL          float64
	PriceBUY               float64
	PriceSELL              float64
}
