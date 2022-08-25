package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/usecasees/structs"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	walletUrlPath  = "/api/v3/account"
	walletSnapshot = "/sapi/v1/accountSnapshot"
)

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

func (u *walletUseCase) Snapshot() (*structs.WalletSnapshot, error) {
	var out structs.WalletSnapshot

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(walletSnapshot)

	q := baseURL.Query()
	q.Set("type", fmt.Sprintf("%s", "SPOT"))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *walletUseCase) GetAllCoins() (*structs.WalletGetAllCoins, error) {
	var out structs.WalletGetAllCoins

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(walletUrlPath)

	q := baseURL.Query()
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		u.logger.WithField("method", "GetAllCoins").Debug(err)
	}

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
