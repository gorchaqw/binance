package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const priceUrlPath = "/api/v3/ticker/price"

type priceUseCase struct {
	clientController *controllers.ClientController
	tgmController    *controllers.TgmController

	priceRepo *sqlite.PriceRepository

	url string

	logger *logrus.Logger
}

func NewPriceUseCase(
	client *controllers.ClientController,
	tgm *controllers.TgmController,
	priceRepo *sqlite.PriceRepository,
	url string,
	logger *logrus.Logger,
) *priceUseCase {
	return &priceUseCase{
		clientController: client,
		tgmController:    tgm,
		priceRepo:        priceRepo,
		url:              url,
		logger:           logger,
	}
}

func (u *priceUseCase) GetPrice() error {

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(priceUrlPath)

	tNow := time.Now()
	tNow.AddDate(0, 0, 1)

	q := baseURL.Query()
	q.Set("symbol", "BTCBUSD")

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
	if err != nil {
		return err
	}

	type reqJson struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return err
	}

	if err := u.tgmController.Send(fmt.Sprintf("%s\n%s", out.Symbol, out.Price)); err != nil {
		return err
	}

	price, err := strconv.ParseFloat(out.Price, 64)
	if err != nil {
		return err
	}

	if err := u.priceRepo.Store(context.Background(), &models.Price{
		Symbol: out.Symbol,
		Price:  price,
	}); err != nil {
		return err
	}

	return nil
}
