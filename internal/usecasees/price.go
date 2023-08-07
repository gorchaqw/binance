package usecasees

import (
	"binance/internal/controllers"
	"binance/internal/repository/postgres"
	"binance/models"
	"encoding/json"
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
	clientController controllers.ClientCtrl
	tgmController    controllers.TgmCtrl

	priceRepo postgres.PriceRepo

	url string

	logger *logrus.Logger
}

func NewPriceUseCase(
	client controllers.ClientCtrl,
	tgm controllers.TgmCtrl,
	priceRepo postgres.PriceRepo,
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

type Depth struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	E            int64      `json:"E"`
	T            int64      `json:"T"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type DepthInfo struct {
	AsksSum float64
	BidsSum float64

	AsksMaxQuery    float64
	AsksMaxPrice    float64
	AsksMaxPosition int

	BidsMaxQuery    float64
	BidsMaxPrice    float64
	BidsMaxPosition int
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

func (u *priceUseCase) GetDepth(symbol string) (*Depth, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featureDepth)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("limit", "100")

	baseURL.RawQuery = q.Encode()

	resp, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out Depth
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *priceUseCase) GetDepthInfo(symbol string) (*DepthInfo, error) {
	var out DepthInfo

	depth, err := u.GetDepth(symbol)
	if err != nil {
		return nil, err
	}

	for k, g := range depth.Asks {
		if k < 5 {
			continue
		}

		var query, price float64

		price, err = strconv.ParseFloat(g[0], 64)
		if err != nil {
			return nil, err
		}

		query, err = strconv.ParseFloat(g[1], 64)
		if err != nil {
			return nil, err
		}

		if query > out.AsksMaxQuery {
			out.AsksMaxQuery = query
			out.AsksMaxPrice = price
			out.AsksMaxPosition = k
		}
	}

	for i := 0; i < out.AsksMaxPosition; i++ {
		query, err := strconv.ParseFloat(depth.Asks[i][1], 64)
		if err != nil {
			return nil, err
		}

		out.AsksSum += query
	}

	for k, g := range depth.Bids {
		if k < 5 {
			continue
		}

		var query, price float64

		price, err = strconv.ParseFloat(g[0], 64)
		if err != nil {
			return nil, err
		}

		query, err = strconv.ParseFloat(g[1], 64)
		if err != nil {
			return nil, err
		}

		if query > out.BidsMaxQuery {
			out.BidsMaxQuery = query
			out.BidsMaxPrice = price
			out.BidsMaxPosition = k
		}
	}

	for i := 0; i < out.BidsMaxPosition; i++ {
		query, err := strconv.ParseFloat(depth.Bids[i][1], 64)
		if err != nil {
			return nil, err
		}

		out.BidsSum += query
	}

	return &out, nil
}

func (u *priceUseCase) GetPriceChangeStatistics(symbol string) (*PriceChangeStatistics, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(featureTicker24hr)

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
	var lastPrice float64

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(priceUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)

	baseURL.RawQuery = q.Encode()

	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				req, err := u.clientController.Send(http.MethodGet, baseURL, nil, false)
				if err != nil {
					u.logger.WithField("method", "Monitoring").Debug(err)
				}

				type reqJson struct {
					Symbol string `json:"symbol"`
					Price  string `json:"price"`
				}
				var out reqJson

				if err := json.Unmarshal(req, &out); err != nil {
					u.logger.WithField("method", "Monitoring").Debug(err)
				}

				price, err := strconv.ParseFloat(out.Price, 64)
				if err != nil {
					u.logger.WithField("method", "Monitoring").Debug(err)
				}

				if price != lastPrice {
					if err := u.priceRepo.Store(&models.Price{
						Symbol: out.Symbol,
						Price:  price,
					}); err != nil {
						u.logger.WithField("method", "Monitoring").Debug(err)
					}
					lastPrice = price
				}
			}
		}
	}()

	return nil
}
