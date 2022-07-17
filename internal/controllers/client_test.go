package controllers_test

import (
	"binance/internal/controllers"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestTicker24h(t *testing.T) {
	httpClient := &http.Client{}
	apiKey := "40A1YfOXYUm85x5slZCL6TcVdB6S8im024Uk5t7Mmj2rQJ2DB0FBSWIpaOB9Zd7J"
	logger := &logrus.Logger{}
	bURL, err := url.Parse("https://api.binance.com/api/v3/ticker/24hr")
	assert.NoError(t, err)

	q := bURL.Query()
	q.Set("symbol", "BNBBTC")
	bURL.RawQuery = q.Encode()

	clientController := controllers.NewClientController(httpClient, apiKey, logger)

	body, err := clientController.Send(http.MethodGet, bURL, nil, false)
	assert.NoError(t, err)

	fmt.Printf("%s\n", body)
}
