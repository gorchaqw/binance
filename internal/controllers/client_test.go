package controllers_test

import (
	"binance/internal/controllers"
	"binance/internal/usecasees"
	"binance/internal/usecasees/structs"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

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
