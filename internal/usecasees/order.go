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

const orderUrlPath = "/api/v3/order"

type orderUseCase struct {
	clientController *controllers.ClientController
	cryptoController *controllers.CryptoController
	tgmController    *controllers.TgmController

	url string

	logger *logrus.Logger
}

func NewOrderUseCase(
	client *controllers.ClientController,
	crypto *controllers.CryptoController,
	tgmController *controllers.TgmController,
	url string,
	logger *logrus.Logger,
) *orderUseCase {
	return &orderUseCase{
		clientController: client,
		cryptoController: crypto,
		tgmController:    tgmController,
		url:              url,
		logger:           logger,
	}
}

func (u *orderUseCase) GetOrder() error {

	return nil

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderUrlPath)

	tNow := time.Now()
	tNow.AddDate(0, 0, 1)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")
	q.Set("side", "SELL")
	q.Set("type", "LIMIT")
	q.Set("timeInForce", "GTC")
	q.Set("quantity", "0.00118")
	q.Set("price", "21200")
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d", tNow.Unix()*1000))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	if err := u.tgmController.Send(fmt.Sprintf("%s", req)); err != nil {
		return err
	}

	return nil
}
