package usecasees

import (
	"binance/internal/controllers"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"path"
	"time"
)

const urlPath = "/api/v3/order"

type OrderUseCase struct {
	clientController *controllers.ClientController
	cryptoController *controllers.CryptoController

	logger *logrus.Logger
}

func NewOrderUseCase(
	client *controllers.ClientController,
	crypto *controllers.CryptoController,
	logger *logrus.Logger,
) *OrderUseCase {
	return &OrderUseCase{
		clientController: client,
		cryptoController: crypto,
		logger:           logger,
	}
}

func (u *OrderUseCase) GetOrder() error {
	baseURLs := "https://api.binance.com"

	baseURL, err := url.Parse(baseURLs)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(urlPath)

	tNow := time.Now()
	tNow.AddDate(0, 0, 1)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
	q.Set("side", "SELL")
	q.Set("type", "LIMIT")
	q.Set("timeInForce", "GTC")
	q.Set("quantity", "0.00055")
	q.Set("price", "21400")
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d", tNow.Unix()*1000))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}
	u.logger.Debug(baseURL.String())

	u.logger.Debugf("%s", req)

	return nil
}
