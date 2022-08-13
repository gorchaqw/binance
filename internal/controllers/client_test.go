package controllers_test

import (
	"binance/internal/controllers"
	"binance/internal/usecasees/structs"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"testing"
	"time"
)

func TestTimeFrame(t *testing.T) {
	TIME_FRAME := 15 * time.Minute
	tNow := time.Now()

	tNow.Truncate(4 * time.Hour)

	fmt.Printf("%d\n", tNow.Hour())
	fmt.Printf("%d\n", tNow.Minute())

	v := (float64(tNow.Minute()) - float64(tNow.Minute()%int(TIME_FRAME.Minutes()))) / TIME_FRAME.Minutes()

	fmt.Printf("%.12f\n", v)

}

func TestGetOrders(t *testing.T) {
	apiUrl := "https://api1.binance.com"
	orderOpenUrlPath := "/api/v3/openOrders"
	//orderAllUrlPath := "/api/v3/allOrders"
	//queryOrder := "/api/v3/order"
	symbol := "BTCBUSD"
	secretKey := "H6kbAHyGNNUdpp1aFEQpqwcQgDLTEWCe45W46vDcWGRtcZuKLJ2g52MdqC6QjuI5"
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	client := &http.Client{}
	logger := logrus.New()

	cryptoController := controllers.NewCryptoController(secretKey)
	clientController := controllers.NewClientController(client, apiKey, logger)

	baseURL, err := url.Parse(apiUrl)
	assert.NoError(t, err)

	baseURL.Path = path.Join(orderOpenUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	//q.Set("limit", "1")
	//q.Set("orderId", "5661989138")
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Add(time.Second*60).Unix()))

	sig := cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := clientController.Send(http.MethodGet, baseURL, nil, true)
	assert.NoError(t, err)

	var out []structs.Order

	fmt.Printf("%s", req)

	assert.NoError(t, json.Unmarshal(req, &out))

	fmt.Printf("%+v", out)

}

func TestQ(t *testing.T) {
	s := 0.001
	l := 0.014
	q := float64(0)

	for i := 1; i < 100; i++ {
		q = s * float64(i) * 2
		if q > l {
			return
		}

		fmt.Printf("i = %d, q = %f \n", i, q)
	}
}

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
