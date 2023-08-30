package controllers_test

import (
	"binance/internal/controllers"
	"binance/internal/usecasees"
	"binance/internal/usecasees/structs"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
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

var (
	//apiKey     = "8332396867b2d1fe57dfa2735a8b10727da8e2d382942481ec3461d07d7cdc32"
	//secretKey  = "d4c11ab5e1f27eb14d735b2c1ac2bb3e62ea3f9da6f8accfecbd3e19a534b717"
	//binanceUrl = "https://testnet.binancefuture.com"

	apiKey     = "sBGPvKBA9qQH5OJscrXunjDd4b89SsC64K2kdtbrGiazEXnuszyqrT7dsmUSFrcu"
	secretKey  = "r4k6e234hHoE3Z88S8YYcMKrRMKdbC1OULDmIavQuoVhc2zzmVfp3JthgyC3QsiI"
	binanceUrl = "https://fapi.binance.com"
)

func Test_TradesList(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	//cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	baseURL, err := url.Parse(binanceUrl)
	assert.NoError(t, err)

	baseURL.Path = path.Join("/fapi/v1/trades")

	q := baseURL.Query()
	q.Set("symbol", usecasees.BTCUSDT)
	q.Set("limit", "500")

	baseURL.RawQuery = q.Encode()

	fmt.Println(baseURL)

	resp, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	var out []usecasees.Trade
	if err := json.Unmarshal(resp, &out); err != nil {
		assert.NoError(t, err)
	}

	for _, trade := range out {
		price, err := strconv.ParseFloat(trade.Price, 64)
		if err != nil {
			assert.NoError(t, err)
		}

		qty, err := strconv.ParseFloat(trade.Price, 64)
		if err != nil {
			assert.NoError(t, err)
		}

		fmt.Printf("%f : %f\n", qty, price)
	}

	//fmt.Printf("%+v", out)

}

func Test_Depth(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	//cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	baseURL, err := url.Parse(binanceUrl)
	assert.NoError(t, err)

	baseURL.Path = path.Join("/fapi/v1/depth")

	q := baseURL.Query()
	q.Set("symbol", usecasees.BTCUSDT)
	q.Set("limit", "500")

	baseURL.RawQuery = q.Encode()

	fmt.Println(baseURL)

	resp, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	var out usecasees.Depth
	if err := json.Unmarshal(resp, &out); err != nil {
		assert.Nil(t, err)
	}

	sum1 := float64(0)
	max1 := float64(0)

	for _, g := range out.Asks {
		//fmt.Printf("%+v %d\n", g, k)

		if q, err := strconv.ParseFloat(g[1], 64); err == nil {
			sum1 += q
		}

		if s, err := strconv.ParseFloat(g[0], 64); err == nil {
			if s > max1 {
				max1 = s
			}
		}

	}

	fmt.Printf("%f %f \n", sum1, max1)

	sum1 = float64(0)
	max2 := float64(30000)

	for _, g := range out.Bids {
		//fmt.Printf("%+v %d\n", g, k)
		if q, err := strconv.ParseFloat(g[1], 64); err == nil {
			sum1 += q
		}

		if s, err := strconv.ParseFloat(g[0], 64); err == nil {
			if s < max1 {
				max2 = s
			}
		}
	}
	fmt.Printf("%f, %f \n", sum1, max2)

	fmt.Printf("%f \n", (max1-max2)/4)

}

func Test_Ticker24(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	//cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	baseURL, err := url.Parse(binanceUrl)
	assert.NoError(t, err)

	baseURL.Path = path.Join("fapi/v1/ticker/24hr")

	q := baseURL.Query()
	q.Set("symbol", usecasees.BTCUSDT)

	baseURL.RawQuery = q.Encode()

	fmt.Println(baseURL)

	resp, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	var out usecasees.PriceChangeStatistics
	if err := json.Unmarshal(resp, &out); err != nil {
		assert.Nil(t, err)
	}

	fmt.Println("HighPrice", out.HighPrice)
	fmt.Println("LowPrice", out.LowPrice)

	highPrice, err := strconv.ParseFloat(out.HighPrice, 64)
	assert.NoError(t, err)

	lowPrice, err := strconv.ParseFloat(out.LowPrice, 64)
	assert.NoError(t, err)

	avgPrice := (highPrice + lowPrice) / 2
	fmt.Println("AvgPrice", avgPrice)

	deltaPrice := avgPrice / 100 * 0.2
	fmt.Println("DeltaPrice", deltaPrice)

	fmt.Println("SHORT", avgPrice+deltaPrice)
	fmt.Println("LONG", avgPrice-deltaPrice)
	fmt.Println("LIMIT", deltaPrice/10)

}
func Test_DailyAccountSnapshot(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	baseURL, err := url.Parse(binanceUrl)
	assert.NoError(t, err)

	baseURL.Path = path.Join("/sapi/v1/capital/withdraw/apply")

	q := baseURL.Query()
	q.Set("coin", "BUSD")
	q.Set("address", "0xec70bf48617269754fee71c3a8e3e63645972f30")
	q.Set("amount", "10")
	q.Set("walletType", "0")

	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	assert.NoError(t, os.WriteFile("./example.json", req, 0644))

	fmt.Printf("%s", req)
}

func Test_CalcCommission(t *testing.T) {
	priceBUY := 21251.00
	priceSELL := priceBUY + 45.00
	quantityStep := 0.002
	commissionTaker := 0.004
	lim := 5.5

	for i := 1; ; i++ {
		quantity := (quantityStep + 0.001) * math.Pow(2, float64(i-1))

		if lim < quantity {
			return
		}

		pBUY := priceBUY * quantity
		pSELL := priceSELL * quantity

		deltaP := pSELL - pBUY
		deltaPrice := priceSELL - priceBUY

		commission := deltaP / 100 * (2 * commissionTaker)

		profit := deltaP - commission
		lose := (-1 * deltaP) - commission

		fmt.Printf("%d]\ndeltaPrice:\t %.4f\n"+
			"deltaP:\t %.4f\n"+
			"comission:\t%.6f\n"+
			"profit:\t%.4f\n"+
			"lose:\t%.4f\n"+
			"quantity:\t%.4f\n\n",
			i,
			deltaPrice,
			deltaP,
			commission,
			profit,
			lose,
			quantity,
		)
	}

	// pSELL-pBUY = 0.000012

	//for i := 1; ; i++ {
	//	quantity = step * math.Pow(2.7, float64(i-1))
	//	profit := quantity * stepPrice
	//	commissionTaker := profit * taker
	//	totalCommission := 2 * commissionTaker
	//	commissionPrice := totalCommission * profit
	//	total := profit - totalCommission
	//	loss := (-1 * profit) - totalCommission
	//
	//	if lim < quantity {
	//		return
	//	}
	//
	//	total *= 20
	//	loss *= 20
	//
	//	fmt.Printf("%d]\n"+
	//		"Quanity:\t\t%.5f\n"+
	//		"Profit:\t\t\t%.5f\n"+
	//		"Commission:\t\t%.5f\n"+
	//		"CommissionPrice:\t%.5f\n"+
	//		"ComissionTaker:\t%.5f\n"+
	//		"Total:\t\t\t%.5f\n"+
	//		"Total x2:\t\t%.5f\n"+
	//		"Loss:\t\t\t%.5f\n\n",
	//		i,
	//		quantity,
	//		profit*20,
	//		commission,
	//		commissionPrice,
	//		commissionTaker*20,
	//		total,
	//		total*2,
	//		loss,
	//	)
	//}
}

func Test_BatchOrders(t *testing.T) {
	client := &http.Client{}

	logger := logrus.New()

	symbol := "BTCUSDT"

	//price := 18966.00

	actualPrice := 21650.00
	quantity := 0.001

	takeProfitPrice := actualPrice + 100
	//stopLossPrice := actualPrice - 100

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	baseURL, err := url.Parse("https://testnet.binancefuture.com")
	assert.NoError(t, err)

	baseURL.Path = path.Join("/fapi/v1/batchOrders")

	//limitOrder := structs.FeatureOrderReq{
	//	Symbol:       symbol,
	//	Type:         "LIMIT",
	//	Side:         "BUY",
	//	PositionSide: "LONG",
	//	Price:        fmt.Sprintf("%.1f", actualPrice),
	//	Quantity:     fmt.Sprintf("%.3f", quantity),
	//	TimeInForce:  "GTC",
	//}

	limitOrderSELL := structs.FeatureOrderReq{
		Symbol:       symbol,
		Type:         "LIMIT",
		Side:         "SELL",
		PositionSide: "SHORT",
		Price:        fmt.Sprintf("%.1f", actualPrice),
		Quantity:     fmt.Sprintf("%.3f", quantity),
		TimeInForce:  "GTC",
	}

	takeProfitOrder := structs.FeatureOrderReq{
		Symbol:        symbol,
		Type:          "TAKE_PROFIT",
		Side:          "SELL",
		Price:         fmt.Sprintf("%f", takeProfitPrice),
		StopPrice:     fmt.Sprintf("%f", takeProfitPrice),
		PositionSide:  "LONG",
		Quantity:      "0.001",
		PriceProtect:  "false",
		ClosePosition: "false",
	}

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
		NewClientOrderId: uuid.NewString(),
		Symbol:           symbol,
		Type:             usecasees.OrderTypeCurrentStopLoss,
		PriceProtect:     "true",
		Quantity:         "0.001",
		ClosePosition:    "true",

		Side:         "SELL",
		PositionSide: "LONG",
		StopPrice:    "19000",
	}

	orders := []structs.FeatureOrderReq{
		limitOrderSELL,
		takeProfitOrder,
		stopLossOrder,
	}

	batchOrders, err := json.Marshal(orders)
	assert.NoError(t, err)

	q := baseURL.Query()
	q.Set("batchOrders", fmt.Sprintf("%s", batchOrders))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	respBody, err := clientController.Send(http.MethodPost, baseURL, nil, true)
	assert.NoError(t, err)

	var resp []structs.FeatureOrderResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		logrus.Debug(err)
	}

	fmt.Printf("%+v", resp)
}

func Test_ChangePositionMode(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()

	baseURL, err := url.Parse("https://testnet.binancefuture.com/fapi/v1/positionSide/dual")
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

func Test_GetFeatureOrderInfo(t *testing.T) {
	client := &http.Client{}

	logger := logrus.New()

	baseURL, err := url.Parse("https://testnet.binancefuture.com/fapi/v1/order")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", "BTCUSDT")
	q.Set("origClientOrderId", "049f1ac4-640b-4e37-b4b7-08cf7cf4b570")
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	fmt.Printf("%s", req)
}

func Test_CreateFuturesMarketOrder(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	symbol := "BTCUSDT"
	baseURL, err := url.Parse("https://testnet.binancefuture.com/fapi/v1/order")
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
	q.Set("newClientOrderId", uuid.NewString())
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
	logger := logrus.New()
	symbol := "BTCUSDT"
	baseURL, err := url.Parse(fmt.Sprintf("%s/fapi/v1/order", binanceUrl))
	assert.NoError(t, err)
	quantity := 0.001
	price := float64(26700)

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
func Test_CreateFuturesTakeProfitOrder(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	symbol := "BTCUSDT"
	baseURL, err := url.Parse(fmt.Sprintf("%s/fapi/v1/order", binanceUrl))
	assert.NoError(t, err)
	quantity := 0.02
	price := float64(28800)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("side", "SELL")
	q.Set("type", "LIMIT")
	q.Set("positionSide", "SHORT")
	q.Set("quantity", fmt.Sprintf("%.3f", quantity))
	q.Set("price", fmt.Sprintf("%.1f", price))
	//q.Set("stopPrice", fmt.Sprintf("%.1f", price))
	q.Set("recvWindow", "60000")
	//q.Set("closePosition", "TRUE")

	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	// take profit dont touch
	//q := baseURL.Query()
	//q.Set("symbol", symbol)
	//q.Set("side", "BUY")
	//q.Set("type", "LIMIT")
	//q.Set("positionSide", "LONG")
	//q.Set("quantity", fmt.Sprintf("%.3f", quantity))
	//q.Set("price", fmt.Sprintf("%.1f", price))
	////q.Set("stopPrice", fmt.Sprintf("%.1f", price))
	//q.Set("recvWindow", "60000")
	////q.Set("closePosition", "TRUE")
	//
	//q.Set("timeInForce", "GTC")
	//q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

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

func Test_CreateFuturesStopLossOrder(t *testing.T) {
	client := &http.Client{}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	//cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)
	cryptoController := controllers.NewCryptoController(secretKey)

	baseURL, err := url.Parse(binanceUrl)
	assert.NoError(t, err)

	baseURL.Path = path.Join("/fapi/v1/order")

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
	q.Set("side", "BUY")
	q.Set("type", "STOP")
	q.Set("positionSide", "SHORT")
	q.Set("quantity", fmt.Sprintf("%.3f", 0.001))
	q.Set("price", fmt.Sprintf("%.1f", 29160.5))
	q.Set("stopPrice", fmt.Sprintf("%.1f", 29170.5))
	//q.Set("stopPrice", fmt.Sprintf("%.1f", price))
	q.Set("recvWindow", "60000")
	//q.Set("closePosition", "TRUE")

	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	// take profit dont touch
	//q := baseURL.Query()
	//q.Set("symbol", symbol)
	//q.Set("side", "BUY")
	//q.Set("type", "LIMIT")
	//q.Set("positionSide", "LONG")
	//q.Set("quantity", fmt.Sprintf("%.3f", quantity))
	//q.Set("price", fmt.Sprintf("%.1f", price))
	////q.Set("stopPrice", fmt.Sprintf("%.1f", price))
	//q.Set("recvWindow", "60000")
	////q.Set("closePosition", "TRUE")
	//
	//q.Set("timeInForce", "GTC")
	//q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

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
	logger := logrus.New()
	symbol := "BTCBUSD"
	baseURL, err := url.Parse("https://fapi.binance.com/fapi/v1/order")
	assert.NoError(t, err)

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
	logger := logrus.New()
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
	logger := logrus.New()
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
	logger := logrus.New()
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
	logger := logrus.New()
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
	logger := logrus.New()
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
	logger := logrus.New()
	baseURL, err := url.Parse("https://fapi.binance.com/fapi/v1/order")
	assert.NoError(t, err)

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(
		client,
		apiKey,
		logger,
	)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
	q.Set("orderId", fmt.Sprintf("%d", 54221521995))
	q.Set("recvWindow", "60000")
	//q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

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
	logger := logrus.New()
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
