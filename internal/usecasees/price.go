package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/sqlite"
	"binance/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	priceUrlPath          = "/api/v3/ticker/price"
	priceChangeStatistics = "/api/v3/ticker/24hr"
)

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

func (u *priceUseCase) GetAverage(symbol string) error {
	ticker := time.NewTicker(10 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				eTime := time.Now()
				sTime := eTime.Add(-6 * time.Hour)

				pList, err := u.priceRepo.GetByCreatedByInterval(symbol, sTime, eTime)
				if err != nil {
					u.logger.Debug(err)
				}

				sum := float64(0)
				for _, p := range pList {
					sum += p.Price
				}
				avr := sum / float64(len(pList))

				if err := u.tgmController.Send(fmt.Sprintf("[ Average ]\n%f\n%s", avr, t.Format(time.RFC822))); err != nil {
					u.logger.Debug(err)
				}
			}
		}
	}()

	return nil
}

type PriceChangeStatistics struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	BidPrice           string `json:"bidPrice"`
	BidQty             string `json:"bidQty"`
	AskPrice           string `json:"askPrice"`
	AskQty             string `json:"askQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstID            int    `json:"firstId"`
	LastID             int    `json:"lastId"`
	Count              int    `json:"count"`
}

func (u *priceUseCase) GetPriceChangeStatistics(symbol string) (*PriceChangeStatistics, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(priceChangeStatistics)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
	if err != nil {
		return nil, err
	}

	var out PriceChangeStatistics

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *priceUseCase) GetPrice(symbol string) (float64, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return 0, err
	}

	baseURL.Path = path.Join(priceUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
	if err != nil {
		return 0, err
	}

	type reqJson struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(out.Price, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (u *priceUseCase) Monitoring(symbol string) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(priceUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	ticker := time.NewTicker(15 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
				if err != nil {
					u.logger.Debug(err)
				}

				type reqJson struct {
					Symbol string `json:"symbol"`
					Price  string `json:"price"`
				}
				var out reqJson

				if err := json.Unmarshal(req, &out); err != nil {
					u.logger.Debug(err)
				}

				price, err := strconv.ParseFloat(out.Price, 64)
				if err != nil {
					u.logger.Debug(err)
				}

				if err := u.priceRepo.Store(&models.Price{
					Symbol: out.Symbol,
					Price:  price,
				}); err != nil {
					u.logger.Debug(err)
				}
			}
		}
	}()

	return nil
}
