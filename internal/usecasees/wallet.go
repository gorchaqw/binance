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

const walletUrlPath = "/api/v3/account"

type walletUseCase struct {
	clientController *controllers.ClientController
	cryptoController *controllers.CryptoController
	tgmController    *controllers.TgmController

	url string

	logger *logrus.Logger
}

func NewWalletUseCase(
	client *controllers.ClientController,
	crypto *controllers.CryptoController,
	tgmController *controllers.TgmController,
	url string,
	logger *logrus.Logger,
) *walletUseCase {
	return &walletUseCase{
		clientController: client,
		cryptoController: crypto,
		tgmController:    tgmController,
		url:              url,
		logger:           logger,
	}
}

func (u *walletUseCase) GetAllCoins() error {

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(walletUrlPath)

	q := baseURL.Query()
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	ticker := time.NewTicker(10 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
				if err != nil {
					u.logger.WithField("method", "GetAllCoins").Debug(err)
				}

				if err := u.tgmController.Send(fmt.Sprintf("%s\n%s", req, t.Format(time.RFC822))); err != nil {
					u.logger.WithField("method", "GetAllCoins").Debug(err)
				}
			}
		}
	}()

	return nil
}
