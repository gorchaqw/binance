package controllers_test

import (
	"binance/internal/controllers"
	"binance/internal/usecasees"
	"binance/internal/usecasees/structs"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_GetPositionInfo(t *testing.T) {
	client := &http.Client{}
	apiKey := "GjQaJQSciytAuD6Td6ZSk1ZXtfEQAdhdDb1dqcE67csSXzBJtDOPmU5IxYAvFZvk"
	logger := logrus.New()
	secretKey := "HeIwNhAQRjWsJTcfVUlXc3yS04Vag9cTPRb2Ls88dBG5x6YtybE579uJhIwz95MC"
	symbol := "BTCBUSD"
	//
	//price := 18966.00

	actualPrice := 18860.00
	//quantity := 0.001

	takeProfitPrice := actualPrice + 100
	//stopLossPrice := actualPrice - 100

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	baseURL, err := url.Parse("https://fapi.binance.com")
	assert.NoError(t, err)

	baseURL.Path = path.Join("/fapi/v1/batchOrders")

	//	"symbol": "BTCBUSD",
	//	"timeInForce": "GTE_GTC",
	//	"type": "STOP_MARKET",
	//	"reduceOnly": true,
	//	"closePosition": true,
	//	"side": "BUY",
	//	"stopPrice": "18800",
	//	"workingType": "MARK_PRICE",
	//	"priceProtect": false,
	//	"origType": "STOP_MARKET",
	//	"time": 1662492808077,
	//	"updateTime": 1662492808077
	//}

	//limitOrder := structs.FeatureOrderReq{
	//	Symbol:       symbol,
	//	Type:         "LIMIT",
	//	Side:         "BUY",
	//	PositionSide: "LONG",
	//	Price:        fmt.Sprintf("%.1f", actualPrice),
	//	Quantity:     fmt.Sprintf("%.3f", quantity),
	//	TimeInForce:  "GTC",
	//}

	//limitOrderSELL := structs.FeatureOrderReq{
	//	Symbol:       symbol,
	//	Type:         "LIMIT",
	//	Side:         "SELL",
	//	PositionSide: "SHORT",
	//	Price:        fmt.Sprintf("%.1f", actualPrice),
	//	Quantity:     fmt.Sprintf("%.3f", quantity),
	//	TimeInForce:  "GTC",
	//}

	//takeProfitOrder := structs.FeatureOrderReq{
	//	Symbol:        symbol,
	//	Type:          "TAKE_PROFIT_MARKET",
	//	Side:          "SELL",
	//	StopPrice:     fmt.Sprintf("%f", takeProfitPrice),
	//	WorkingType:   "MARK_PRICE",
	//	PositionSide:  "LONG",
	//	PriceProtect:  "false",
	//	ClosePosition: "true",
	//}

	//takeProfitOrder := structs.FeatureOrderReq{
	//	Symbol:        symbol,
	//	Type:          "TAKE_PROFIT_MARKET",
	//	Side:          "BUY",
	//	StopPrice:     fmt.Sprintf("%f", stopLossPrice),
	//	WorkingType:   "MARK_PRICE",
	//	PositionSide:  "SHORT",
	//	PriceProtect:  "false",
	//	ClosePosition: "true",
	//}
	//
	stopLossOrder := structs.FeatureOrderReq{
		Symbol:        symbol,
		Type:          "STOP_MARKET",
		Side:          "BUY",
		StopPrice:     fmt.Sprintf("%.1f", takeProfitPrice),
		WorkingType:   "MARK_PRICE",
		PositionSide:  "SHORT",
		PriceProtect:  "false",
		ClosePosition: "true",
	}

	orders := []structs.FeatureOrderReq{
		//limitOrderSELL,
		//takeProfitOrder,
		stopLossOrder,
	}

	batchOrders, err := json.Marshal(orders)
	assert.NoError(t, err)

	fmt.Printf("%s", batchOrders)

	q := baseURL.Query()
	q.Set("batchOrders", fmt.Sprintf("%s", batchOrders))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)
}

func Test_ChangePositionMode(t *testing.T) {
	client := &http.Client{}
	apiKey := "GjQaJQSciytAuD6Td6ZSk1ZXtfEQAdhdDb1dqcE67csSXzBJtDOPmU5IxYAvFZvk"
	logger := logrus.New()
	secretKey := "HeIwNhAQRjWsJTcfVUlXc3yS04Vag9cTPRb2Ls88dBG5x6YtybE579uJhIwz95MC"

	baseURL, err := url.Parse("https://fapi.binance.com/fapi/v1/positionSide/dual")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("dualSidePosition", "true")
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)
}
func Test_CreateFuturesMarketOrder(t *testing.T) {
	client := &http.Client{}
	apiKey := "GjQaJQSciytAuD6Td6ZSk1ZXtfEQAdhdDb1dqcE67csSXzBJtDOPmU5IxYAvFZvk"
	logger := logrus.New()
	secretKey := "HeIwNhAQRjWsJTcfVUlXc3yS04Vag9cTPRb2Ls88dBG5x6YtybE579uJhIwz95MC"
	symbol := "BTCBUSD"
	baseURL, err := url.Parse("https://fapi.binance.com/fapi/v1/order")
	assert.NoError(t, err)
	quantity := 0.001

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("side", "BUY")
	q.Set("positionSide", "LONG")
	q.Set("type", "MARKET")
	q.Set("quantity", fmt.Sprintf("%.3f", quantity))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)

	var o structs.FeatureOrderResp
	assert.NoError(t, json.Unmarshal(req, &o))

	fmt.Printf("%+v", o)
}
func Test_CreateFuturesLimitOrder(t *testing.T) {
	client := &http.Client{}
	apiKey := "GjQaJQSciytAuD6Td6ZSk1ZXtfEQAdhdDb1dqcE67csSXzBJtDOPmU5IxYAvFZvk"
	logger := logrus.New()
	secretKey := "HeIwNhAQRjWsJTcfVUlXc3yS04Vag9cTPRb2Ls88dBG5x6YtybE579uJhIwz95MC"
	symbol := "BTCBUSD"
	baseURL, err := url.Parse("https://fapi.binance.com/fapi/v1/order")
	assert.NoError(t, err)
	quantity := 0.001
	price := float64(19000)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("side", "BUY")
	q.Set("type", "LIMIT")
	q.Set("positionSide", "LONG")
	q.Set("quantity", fmt.Sprintf("%.3f", quantity))
	q.Set("price", fmt.Sprintf("%.1f", price))
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)

	var o structs.LimitOrder
	assert.NoError(t, json.Unmarshal(req, &o))

	fmt.Printf("%+v", o)
}

func Test_allOpenOrders(t *testing.T) {
	client := &http.Client{}
	apiKey := "GjQaJQSciytAuD6Td6ZSk1ZXtfEQAdhdDb1dqcE67csSXzBJtDOPmU5IxYAvFZvk"
	logger := logrus.New()
	secretKey := "HeIwNhAQRjWsJTcfVUlXc3yS04Vag9cTPRb2Ls88dBG5x6YtybE579uJhIwz95MC"
	symbol := "BTCBUSD"
	baseURL, err := url.Parse("https://fapi.binance.com/fapi/v1/order")
	assert.NoError(t, err)
	//quantity := 0.001
	//price := float64(19900)
	//stopPrice := float64(19600)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", "12045007771")
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)
}

func Test_CreateLimitOrder(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	symbol := "BTCUSDT"
	baseURL, err := url.Parse("https://api.binance.com/api/v3/order")
	assert.NoError(t, err)
	quantity := 0.00055
	price := float64(20000)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("side", "BUY")
	q.Set("type", "LIMIT")
	q.Set("quantity", fmt.Sprintf("%.5f", quantity))
	q.Set("price", fmt.Sprintf("%.2f", price))
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(60*time.Second).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	var o structs.LimitOrder
	assert.NoError(t, json.Unmarshal(req, &o))

	fmt.Printf("%+v", o)

}
func Test_OSO(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	symbol := "BTCBUSD"
	baseURL, err := url.Parse("https://api.binance.com/api/v3/order/oco")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("side", usecasees.SideBuy)
	q.Set("quantity", "0.0005")
	q.Set("recvWindow", "60000")
	q.Set("price", "23800")
	q.Set("stopPrice", "23900")
	q.Set("stopLimitPrice", "23900")
	q.Set("stopLimitTimeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	var oList structs.OrderList

	assert.NoError(t, json.Unmarshal(req, &oList))

	fmt.Printf("%+v", oList)
}
func Test_GetOrderList(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	baseURL, err := url.Parse("https://api.binance.com/api/v3/orderList")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("orderListId", fmt.Sprintf("%d", 72305328))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)

	var out structs.OrderList

	assert.NoError(t, json.Unmarshal(req, &out))
}
func Test_WalletGetAllCoins(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	baseURL, err := url.Parse("https://api.binance.com/api/v3/account")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)
}
func Test_WalletSnapshot(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	baseURL, err := url.Parse("https://api.binance.com/sapi/v1/accountSnapshot")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	eTime := time.Now()
	sTime := eTime.Add(-24 * time.Hour)

	q := baseURL.Query()
	q.Set("type", fmt.Sprintf("%s", "SPOT"))
	q.Set("startTime", fmt.Sprintf("%d000", sTime.Unix()))
	q.Set("endTime", fmt.Sprintf("%d000", eTime.Unix()))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)
}
func Test_GetOrderInfo(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	baseURL, err := url.Parse("https://api.binance.com/api/v3/order")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
	q.Set("orderId", fmt.Sprintf("%d", 5715391573))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)

	var out structs.Order

	assert.NoError(t, json.Unmarshal(req, &out))

	fmt.Printf("%+v", out)
}
func Test_GetOpenOrders(t *testing.T) {
	client := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := logrus.New()
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	baseURL, err := url.Parse("https://api.binance.com/api/v3/openOrders")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("req:'%s'", req)

	var out []structs.Order

	assert.NoError(t, json.Unmarshal(req, &out))

	fmt.Printf("%+v", out)
	fmt.Printf("%+v", len(out))
}

func Test_POW(t *testing.T) {
	deltaP := float64(100)

	priceBUY := float64(21640)
	priceSELL := priceBUY + deltaP
	priceSELLTest := priceSELL + deltaP

	fmt.Println(0.05500 - 0.02750)
	fmt.Println(0.00055 * 5 * priceBUY)

	step := 0.00055 * 5

	Q := priceBUY * step
	s1 := priceSELL * step
	s2 := priceSELLTest * step

	fmt.Printf("Q = %.5f\ns1 = %.5f\ns2 = %.5f\ndeltaS = %.5f\ndeltaP = %.5f\n", Q, s1, s2, s2-s1, priceSELLTest-priceSELL)
}

func Test_Limit(t *testing.T) {
	price := 21704.00
	pStep := 0.0006

	delta := 0.25
	deltaStep := 0.065
	lim := 0.02

	for i := 1; i <= 7; i++ {
		p := price / 100 * (delta + (deltaStep * float64(i)))
		fmt.Printf("%d]\t%.2f\n", i, p)
	}

	fmt.Println()

	n := 0
	s := lim
	for s >= pStep {
		fmt.Printf("%d]\t%.5f\n", n, s)

		n++
		s = s / 2
	}
}

func Test_Ticker(t *testing.T) {
	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)
	wait := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				fmt.Println("1")

				continue
			}
		}
	}()

	<-wait
}

func TestStep(t *testing.T) {
	step := 0.008
	lim := 0.35
	quantity := 0.00

	for i := 1; ; i++ {
		quantity = (step * math.Pow(2, float64(i-1))) + (0.0015 * math.Pow(2, float64(i-1)))

		if lim < quantity {
			return
		}

		fmt.Printf("%d]\t%.5f\n", i, quantity)
	}
}

func Test_Calc(t *testing.T) {
	priceBUY := float64(21640)
	priceSELL := float64(21704)

	money := float64(328)

	quantity := 0.0006
	limit := money / priceBUY

	fmt.Printf("limit:\t%.5f\n\n", limit)

	try := 1
	nQuantity := float64(0)

	for {
		if try == 1 {
			nQuantity = quantity
		} else {
			n := try - 1
			nQuantity = quantity * math.Pow(2, float64(n))

			if nQuantity > limit {
				return
			}
		}

		fmt.Printf("[%d] %.5f\n", try, nQuantity)

		buy := priceBUY * nQuantity
		sell := priceSELL * nQuantity

		profit := sell - buy

		fmt.Printf("buy price:\t%.4f\n"+
			"sell price:\t%.4f\n"+
			"buy:\t%.4f\n"+
			"sell:\t%.4f\n"+
			"profit:\t%.4f\n\n",
			priceBUY,
			priceSELL,
			buy,
			sell,
			profit,
		)

		try++
	}
}

func TestPrice(t *testing.T) {
	d := (20500 - 19800) / 5
	priceSELL := 20310 + d
	fmt.Println(priceSELL, d/2)
}

func TestStat(t *testing.T) {
	file, err := os.Open("binance.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 5
	reader.Comment = '#'
	var totalProfit, totalComission float64

	for {
		record, e := reader.Read()
		if e != nil {
			fmt.Println(e)
			break
		}

		switch record[1] {
		case "COMMISSION":
			f, err := strconv.ParseFloat(record[2], 64)
			if err != nil {
				panic(err)
			}
			totalComission += f
		case "REALIZED_PNL":
			f, err := strconv.ParseFloat(record[2], 64)
			if err != nil {
				panic(err)
			}
			totalProfit += f
		}
	}

	fmt.Println(totalProfit)
	fmt.Println(totalComission)
}
