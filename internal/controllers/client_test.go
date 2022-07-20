package controllers_test

import (
	"binance/internal/controllers"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestTicker24h(t *testing.T) {
	httpClient := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"

	logger := &logrus.Logger{}
	clientController := controllers.NewClientController(httpClient, apiKey, logger)
	cryptoController := controllers.NewCryptoController(secretKey)

	t.Run("orders", func(t *testing.T) {

		type Order struct {
			Symbol              string `json:"symbol"`
			OrderID             int    `json:"orderId"`
			OrderListID         int    `json:"orderListId"`
			ClientOrderID       string `json:"clientOrderId"`
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

		var orders []Order
		
		symbol := "BTCBUSD"
		bURL, err := url.Parse("https://api2.binance.com/api/v3/allOrders")
		assert.NoError(t, err)

		q := bURL.Query()
		q.Set("symbol", symbol)
		q.Set("recvWindow", "60000")
		q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*30).Unix()))

		sig := cryptoController.GetSignature(q.Encode())
		q.Set("signature", sig)

		bURL.RawQuery = q.Encode()

		body, err := clientController.Send(http.MethodGet, bURL, nil, true)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &orders))

		var total float64
		for _, order := range orders {
			qty, err := strconv.ParseFloat(order.ExecutedQty, 64)
			assert.NoError(t, err)

			price, err := strconv.ParseFloat(order.Price, 64)
			assert.NoError(t, err)

			switch order.Side {
			case "BUY":
				total += qty * price
			case "SELL":
				total -= qty * price
			}
		}

		fmt.Printf("%s %.2f\n", symbol, total)
	})

	t.Run("ticker", func(t *testing.T) {
		bURL, err := url.Parse("https://api.binance.com/api/v3/ticker/24hr")
		assert.NoError(t, err)

		q := bURL.Query()
		q.Set("symbol", "BNBBTC")
		bURL.RawQuery = q.Encode()

		body, err := clientController.Send(http.MethodGet, bURL, nil, false)
		assert.NoError(t, err)

		fmt.Printf("%s\n", body)

	})
}
