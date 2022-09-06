package usecasees

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ic2hrmk/promtail"
	"github.com/sirupsen/logrus"

	"binance/internal/controllers"
	"binance/internal/repository/mongo"
	mongoStructs "binance/internal/repository/mongo/structs"
	"binance/internal/repository/postgres"
	"binance/internal/usecasees/structs"
	"binance/models"
)

const (
	orderUrlPath     = "/api/v3/order"
	orderList        = "/api/v3/orderList"
	orderAllUrlPath  = "/api/v3/allOrders"
	orderOpenUrlPath = "/api/v3/openOrders"
	orderOCO         = "/api/v3/order/oco"

	BTC  = "BTC"
	ETH  = "ETH"
	RUB  = "RUB"
	BUSD = "BUSD"
	USDT = "USDT"

	ETHRUB  = ETH + RUB
	ETHBUSD = ETH + BUSD
	BTCBUSD = BTC + BUSD
	BTCUSDT = BTC + USDT

	SideSell = "SELL"
	SideBuy  = "BUY"

	OrderStatusNew      = "NEW"
	OrderStatusCanceled = "CANCELED"
	OrderStatusFilled   = "FILLED"

	OrderTypeLimit = "LIMIT"
	OrderTypeOCO   = "OCO"
)

var (
	SymbolList = []string{
		BTCBUSD,
	}
)

type orderUseCase struct {
	clientController controllers.ClientCtrl
	cryptoController controllers.CryptoCtrl

	settingsRepo mongo.SettingsRepo
	orderRepo    postgres.OrderRepo

	priceUseCase *priceUseCase

	url string

	logRus   *logrus.Logger
	promTail promtail.Client
	metrics  map[structs.MetricConst]prometheus.Counter
}

func NewOrderUseCase(
	client controllers.ClientCtrl,
	crypto controllers.CryptoCtrl,
	settingsRepo mongo.SettingsRepo,
	orderRepo postgres.OrderRepo,
	priceUseCase *priceUseCase,
	url string,
	logger *logrus.Logger,
	promTail promtail.Client,
	metrics map[structs.MetricConst]prometheus.Counter,
) *orderUseCase {
	return &orderUseCase{
		clientController: client,
		cryptoController: crypto,
		settingsRepo:     settingsRepo,
		orderRepo:        orderRepo,
		priceUseCase:     priceUseCase,
		url:              url,
		logRus:           logger,
		promTail:         promTail,
		metrics:          metrics,
	}
}

func (u *orderUseCase) FeaturesMonitoring(symbol string) error {
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)

	settings, err := u.settingsRepo.Load(symbol)
	if err != nil {
		return err
	}

	var status structs.Status
	status.Reset(settings.Step)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				settings, err = u.settingsRepo.Load(symbol)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("Settings: %+v", settings)

				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					switch err {
					case sql.ErrNoRows:
						status.Reset(settings.Step)

						actualPrice, err := u.priceUseCase.GetPrice(symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status)

						if err := u.createFeaturesLimitOrder(pricePlan.SetSide(SideBuy), settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

						if err := u.createFeaturesLimitOrder(pricePlan.SetSide(SideSell), settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}
					default:
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}
				u.promTail.Debugf("LastOrder: %+v", lastOrder)

				status.
					SetOrderTry(lastOrder.Try).
					SetQuantity(lastOrder.Quantity).
					SetSessionID(lastOrder.SessionID)

				oList, err := u.orderRepo.GetBySessionID(lastOrder.SessionID)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}

				for _, o := range oList {
					orderInfo, err := u.getFeatureOrderInfo(o.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					if err := u.orderRepo.SetStatus(lastOrder.ID, orderInfo.Status); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}

				oListSELL, err := u.orderRepo.GetBySessionIDWithSide(lastOrder.SessionID, SideSell)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("oListSELL: %+v", oListSELL)

				oListBUY, err := u.orderRepo.GetBySessionIDWithSide(lastOrder.SessionID, SideBuy)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("oListBUY: %+v", oListBUY)

				sellOrders := func(orders []models.Order) (int, int) {
					var sFilled, sNew int
					for _, o := range orders {
						switch o.Side {
						case SideSell:
							if o.Status == OrderStatusFilled {
								sFilled++
							}
							if o.Status == OrderStatusNew {
								sNew++
							}
						}
					}

					return sFilled, sNew
				}

				buyOrders := func(orders []models.Order) (int, int) {
					var bFilled, bNew int
					for _, o := range orders {
						switch o.Side {
						case SideBuy:
							if o.Status == OrderStatusFilled {
								bFilled++
							}
							if o.Status == OrderStatusNew {
								bNew++
							}
						}
					}

					return bFilled, bNew
				}

				sFilled, sNew := sellOrders(oListSELL)
				bFilled, bNew := buyOrders(oListBUY)

				switch true {
				case len(oListSELL) == 1 && len(oListBUY) == 1 && bFilled == 0 && bNew == 1 && sFilled != 0 && sNew == 0:

					status.AddOrderTry(1)

					pricePlanSELL := u.fillPricePlan(OrderTypeOCO, symbol, oListSELL[0].ActualPrice, settings, &status).SetSide(SideSell)
					pricePlanBUY := u.fillPricePlan(OrderTypeOCO, symbol, oListBUY[0].Price, settings, &status).SetSide(SideBuy)

					if err := u.createFeaturesLimitOrder(pricePlanBUY, settings); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

						continue
					}
					if err := u.createFeaturesLimitOrder(pricePlanSELL, settings); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

						continue
					}
				}

				continue

				var actualPrice float64

				switch lastOrder.Status {
				case OrderStatusFilled:
					actualPrice = lastOrder.Price
				case OrderStatusCanceled:
					actualPrice = lastOrder.StopPrice
				case OrderStatusNew:
					continue
				}

				openOrders, err := u.getOpenOrders(symbol)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

					continue
				}

				if len(openOrders) == 0 {
					switch lastOrder.Side {
					case SideBuy:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideSell)
						u.logRus.Debug(pricePlan)
						u.promTail.Debugf("SideBuy price plan: %+v", pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

					case SideSell:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						u.logRus.Debug(pricePlan)
						u.promTail.Debugf("SideSell price plan: %+v", pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

					}
				}
			}
		}
	}()

	return nil
}

func (u *orderUseCase) Monitoring(symbol string) error {
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)

	settings, err := u.settingsRepo.Load(symbol)
	if err != nil {
		return err
	}

	var status structs.Status
	status.Reset(settings.Step)

	//sendOrderInfo := func(order *models.Order) {
	//	if err := u.tgmController.Send(fmt.Sprintf("[ Last Order ]\n"+
	//		"orderId:\t%d\n"+
	//		"status:\t%s\n"+
	//		"side:\t%s\n"+
	//		"order:\t%+v\n",
	//		order.OrderID,
	//		order.Status,
	//		order.Side,
	//		order)); err != nil {
	//		u.logRus.
	//			WithError(err).
	//			Error(string(debug.Stack()))
	//		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
	//	}
	//}
	//sendLimit := func(quantity float64) {
	//	if err := u.tgmController.Send(fmt.Sprintf("[ Limit ]\n"+
	//		"quantity:\t%.5f\n",
	//		quantity)); err != nil {
	//		u.logRus.
	//			WithError(err).
	//			Error(string(debug.Stack()))
	//		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
	//	}
	//}
	//sendStat := func(stat *structs.PricePlan) {
	//	if err := u.tgmController.Send(fmt.Sprintf("[ Stat ]\n"+
	//		"actualPrice:\t%.2f\n"+
	//		"actualPricePercent:\t%.2f\n"+
	//		"stopPriceBUY:\t%.2f\n"+
	//		"stopPriceSELL:\t%.2f\ncreateFeaturesLimitOrder"+
	//		"priceBUY:\t%.2f\n"+
	//		"priceSELL:\t%.2f\n",
	//		stat.ActualPrice,
	//		stat.ActualPricePercent,
	//		stat.StopPriceBUY,
	//		stat.StopPriceSELL,
	//		stat.PriceBUY,
	//		stat.PriceSELL)); err != nil {
	//		u.logRus.
	//			WithError(err).
	//			Error(string(debug.Stack()))
	//		u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
	//	}
	//}

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				settings, err = u.settingsRepo.Load(symbol)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
				}
				u.promTail.Debugf("Settings: %+v", settings)

				lastOrder, err := u.orderRepo.GetLast(symbol)
				if err != nil {
					switch err {
					case sql.ErrNoRows:
						status.Reset(settings.Step)

						actualPrice, err := u.priceUseCase.GetPrice(symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
						}

						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}
					default:
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}
				u.promTail.Debugf("LastOrder: %+v", lastOrder)

				u.logRus.
					WithField("settings", settings.Status).
					Debug("settings")

				switch settings.Status {
				case mongoStructs.New.ToString():
					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					if orderInfo.Status == OrderStatusFilled && orderInfo.Side == SideSell && orderInfo.Type == "LIMIT" {
						status.Reset(settings.Step)

						u.promTail.Debugf("MongoStructs.New: %+v", status)

						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, lastOrder.Price, settings, &status).SetSide(SideBuy)
						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.Enabled); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
						}
					}
				case mongoStructs.LiquidationSELL.ToString():
					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					if orderInfo.Status == OrderStatusFilled && orderInfo.Side == SideBuy && orderInfo.Type == "LIMIT" {
						pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, lastOrder.Price, settings, &status).SetSide(SideSell)

						u.logRus.
							WithField("pricePlan", pricePlan).
							Debug("LiquidationSELL")

						if err := u.CreateLimitOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.New); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
						}
					}
				case mongoStructs.LiquidationBUY.ToString():
					if lastOrder.Status == OrderStatusCanceled && lastOrder.Side == SideSell && lastOrder.Type == "OCO" {
						status.
							SetQuantity(settings.Limit).
							SetOrderTry(1)

						pricePlan := u.fillPricePlan(OrderTypeLimit, symbol, lastOrder.StopPrice, settings, &status).SetSide(SideBuy)

						u.logRus.
							WithField("pricePlan", pricePlan).
							WithField("status", pricePlan.Status).
							Debug("LiquidationBUY")

						if err := u.CreateLimitOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
							continue
						}

						if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.LiquidationSELL); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
						}
					}
				case mongoStructs.Disabled.ToString():
					continue
				}

				status.
					SetOrderTry(lastOrder.Try).
					SetQuantity(lastOrder.Quantity)

				lastOrderStatus := lastOrder.Status

				switch lastOrder.Type {
				case "LIMIT":
					orderInfo, err := u.getOrderInfo(lastOrder.OrderID, symbol)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					if orderInfo.Status == OrderStatusFilled {
						lastOrderStatus = OrderStatusFilled
					}
				case "OCO":
					orderList, err := u.getOrderList(lastOrder.OrderID)
					if err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					for _, o := range orderList.Orders {
						orderInfo, err := u.getOrderInfo(o.OrderID, symbol)
						if err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
						}

						switch true {
						case orderInfo.Type == "STOP_LOSS_LIMIT" && orderInfo.Status == OrderStatusFilled:
							lastOrderStatus = OrderStatusCanceled
							u.metrics[structs.MetricOrderStopLossLimitFilled].Inc()
							u.promTail.Debugf("STOP_LOSS_LIMIT Filled: %+v", orderInfo)

							status.
								AddOrderTry(1)

							if orderInfo.Side == SideSell {
								status.AddQuantity(settings.Step)
							}

						case orderInfo.Type == "LIMIT_MAKER" && orderInfo.Status == OrderStatusFilled:
							lastOrderStatus = OrderStatusFilled
							u.metrics[structs.MetricOrderLimitMaker].Inc()
							u.promTail.Debugf("LIMIT_MAKER Filled: %+v", orderInfo)

							if orderInfo.Side == SideSell {
								status.Reset(settings.Step)

								u.metrics[structs.MetricOrderComplete].Inc()
							}
						}
					}
				}

				u.logRus.
					WithField("lastOrderStatus", lastOrder.Status).
					WithField("newOrderStatus", lastOrderStatus).
					WithField("orderTry", status.OrderTry).
					WithField("quantity", status.Quantity).
					WithField("sessionID", status.SessionID).
					Debug("update status")

				if lastOrder.Status != lastOrderStatus {
					if err := u.orderRepo.SetStatus(lastOrder.ID, lastOrderStatus); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					if lastOrder, err = u.orderRepo.GetByID(lastOrder.ID); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}
				}

				if status.Quantity > settings.Limit {
					if err := u.settingsRepo.UpdateStatus(settings.ID, mongoStructs.LiquidationBUY); err != nil {
						u.logRus.
							WithError(err).
							Error(string(debug.Stack()))
						u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())
					}

					continue
				}

				var actualPrice float64

				switch lastOrder.Status {
				case OrderStatusFilled:
					actualPrice = lastOrder.Price
				case OrderStatusCanceled:
					actualPrice = lastOrder.StopPrice
				case OrderStatusNew:
					continue
				}

				openOrders, err := u.getOpenOrders(symbol)
				if err != nil {
					u.logRus.
						WithError(err).
						Error(string(debug.Stack()))
					u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

					continue
				}

				if len(openOrders) == 0 {
					switch lastOrder.Side {
					case SideBuy:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideSell)
						u.logRus.Debug(pricePlan)
						u.promTail.Debugf("SideBuy price plan: %+v", pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

					case SideSell:
						pricePlan := u.fillPricePlan(OrderTypeOCO, symbol, actualPrice, settings, &status).SetSide(SideBuy)
						u.logRus.Debug(pricePlan)
						u.promTail.Debugf("SideSell price plan: %+v", pricePlan)

						if err := u.createOCOOrder(pricePlan, settings); err != nil {
							u.logRus.
								WithError(err).
								Error(string(debug.Stack()))
							u.promTail.Errorf("orderUseCase: %+v %s", err, debug.Stack())

							continue
						}

					}
				}
			}
		}
	}()

	return nil
}

func (u *orderUseCase) fillPricePlan(orderType string, symbol string, actualPrice float64, settings *mongoStructs.Settings, status *structs.Status) *structs.PricePlan {
	var out structs.PricePlan

	out.Symbol = symbol
	out.ActualPrice = actualPrice

	switch orderType {
	case OrderTypeLimit:
		out.ActualPricePercent = out.ActualPrice / 100 * (settings.Delta * 1.2)
	case OrderTypeOCO:
		out.ActualPricePercent = out.ActualPrice / 100 * settings.Delta
	}

	//out.ActualPricePercent = out.ActualPrice / 100 * (settings.Delta + (settings.DeltaStep * float64(orderTry)))
	out.ActualStopPricePercent = out.ActualPricePercent

	out.StopPriceBUY = out.ActualPrice + out.ActualStopPricePercent
	out.StopPriceSELL = out.ActualPrice - out.ActualStopPricePercent

	out.PriceBUY = out.ActualPrice - out.ActualPricePercent
	out.PriceSELL = out.ActualPrice + out.ActualPricePercent

	out.Status = status

	return &out
}

func (u *orderUseCase) getOpenOrders(symbol string) ([]structs.Order, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(orderOpenUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out []structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (u *orderUseCase) getOrderList(orderListID int64) (*structs.OrderList, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(orderList)

	q := baseURL.Query()
	q.Set("orderListId", fmt.Sprintf("%d", orderListID))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out structs.OrderList

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
func (u *orderUseCase) getOrderInfo(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", fmt.Sprintf("%d", orderID))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
func (u *orderUseCase) CreateLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	actualPrice, err := u.priceUseCase.GetPrice(pricePlan.Symbol)
	if err != nil {
		return err
	}

	switch pricePlan.Side {
	case SideBuy:
		if actualPrice < pricePlan.PriceBUY {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideBuy)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateLimitOrder newPricePlan: %+v", pricePlan)

			u.metrics[structs.MetricOrderLimitNewPricePlan].Inc()
		}
	case SideSell:
		if actualPrice > pricePlan.PriceSELL {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideSell)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateLimitOrder newPricePlan: %+v", pricePlan)

			u.metrics[structs.MetricOrderLimitNewPricePlan].Inc()
		}
	}

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}
	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("type", "LIMIT")
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("quantity", fmt.Sprintf("%.5f", pricePlan.Status.Quantity))
	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceBUY))
	case SideSell:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceSELL))
	}
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	var o structs.LimitOrder

	if err := json.Unmarshal(req, &o); err != nil {
		return err
	}

	if o.OrderID == 0 {
		var errMsg structs.Err
		if err := json.Unmarshal(req, &errMsg); err != nil {
			return err
		}

		return errors.New(errMsg.Msg)
	}

	orderModel := models.Order{
		OrderID:     o.OrderID,
		SessionID:   pricePlan.Status.SessionID,
		Symbol:      o.Symbol,
		ActualPrice: pricePlan.ActualPrice,
		Side:        pricePlan.Side,
		Quantity:    pricePlan.Status.Quantity,
		Type:        "LIMIT",
		Status:      "NEW",
		Try:         pricePlan.Status.OrderTry,
	}

	switch pricePlan.Side {
	case SideBuy:
		orderModel.Price = pricePlan.PriceBUY
	case SideSell:
		orderModel.Price = pricePlan.PriceSELL
	}

	if err := u.orderRepo.Store(&orderModel); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) getFeatureOrderInfo(orderID int64, symbol string) (*structs.Order, error) {
	baseURL, err := url.Parse("https://fapi.binance.com")
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join("/fapi/v1/order")

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", fmt.Sprintf("%d", orderID))
	q.Set("recvWindow", "60000")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodGet, baseURL, nil, true)
	if err != nil {
		return nil, err
	}

	var out structs.Order

	if err := json.Unmarshal(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (u *orderUseCase) createFeaturesLimitOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	actualPrice, err := u.priceUseCase.GetPrice(pricePlan.Symbol)
	if err != nil {
		return err
	}

	switch pricePlan.Side {
	case SideBuy:
		if actualPrice < pricePlan.PriceBUY {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideBuy)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateLimitOrder newPricePlan: %+v", pricePlan)

			u.metrics[structs.MetricOrderLimitNewPricePlan].Inc()
		}
	case SideSell:
		if actualPrice > pricePlan.PriceSELL {
			newPricePlan := u.fillPricePlan(OrderTypeLimit, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideSell)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateLimitOrder newPricePlan: %+v", pricePlan)

			u.metrics[structs.MetricOrderLimitNewPricePlan].Inc()
		}
	}

	baseURL, err := url.Parse("https://fapi.binance.com")
	if err != nil {
		return err
	}

	baseURL.Path = path.Join("/fapi/v1/order")

	q := baseURL.Query()
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("type", "LIMIT")
	q.Set("quantity", fmt.Sprintf("%.3f", pricePlan.Status.Quantity))
	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.1f", pricePlan.PriceBUY))
	case SideSell:
		q.Set("price", fmt.Sprintf("%.1f", pricePlan.PriceSELL))
	}
	//q.Set("stopPrice", fmt.Sprintf("%.2f", stopPrice))
	q.Set("recvWindow", "60000")
	q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()
	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	u.promTail.Debugf("createFeaturesLimitOrder Req: %s", req)

	var order structs.LimitOrder
	var errMsg structs.Err

	if err := json.Unmarshal(req, &order); err != nil {
		return err
	}

	if order.OrderID == 0 {
		if err := json.Unmarshal(req, &errMsg); err != nil {
			return err
		}

		switch errMsg.Code {
		case -2010:
			return structs.ErrTheRelationshipOfThePrices
		}

		return errors.New(errMsg.Msg)
	}

	o := models.Order{
		OrderID:     order.OrderID,
		SessionID:   pricePlan.Status.SessionID,
		Symbol:      order.Symbol,
		ActualPrice: pricePlan.ActualPrice,
		Side:        pricePlan.Side,
		Quantity:    pricePlan.Status.Quantity,
		Type:        "LIMIT",
		Status:      "NEW",
		Try:         pricePlan.Status.OrderTry,
	}

	switch pricePlan.Side {
	case SideBuy:
		o.Price = pricePlan.PriceBUY
		o.StopPrice = pricePlan.StopPriceBUY
	case SideSell:
		o.Price = pricePlan.PriceSELL
		o.StopPrice = pricePlan.StopPriceSELL
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) createOCOOrder(pricePlan *structs.PricePlan, settings *mongoStructs.Settings) error {
	u.promTail.Debugf("CreateOCOOrder pricePlan: %+v", pricePlan)
	u.promTail.Debugf("CreateOCOOrder pricePlan.Status: %+v", pricePlan.Status)
	u.promTail.Debugf("CreateOCOOrder pricePlan.Settings: %+v", settings)

	actualPrice, err := u.priceUseCase.GetPrice(pricePlan.Symbol)
	if err != nil {
		return err
	}

	switch pricePlan.Side {
	case SideBuy:
		if actualPrice < pricePlan.PriceBUY {
			newPricePlan := u.fillPricePlan(OrderTypeOCO, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideBuy)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateOCOOrder newPricePlan: %+v", pricePlan)
			u.promTail.Debugf("CreateOCOOrder newPricePlan.Settings: %+v", settings)
			u.promTail.Debugf("CreateOCOOrder newPricePlan.Status: %+v", pricePlan.Status)

			u.metrics[structs.MetricOrderOSONewPricePlan].Inc()
		}
	case SideSell:
		if actualPrice > pricePlan.PriceSELL {
			newPricePlan := u.fillPricePlan(OrderTypeOCO, pricePlan.Symbol, actualPrice, settings, pricePlan.Status).SetSide(SideSell)
			pricePlan = newPricePlan
			u.promTail.Debugf("CreateOCOOrder newPricePlan: %+v", pricePlan)
			u.promTail.Debugf("CreateOCOOrder newPricePlan.Settings: %+v", settings)
			u.promTail.Debugf("CreateOCOOrder newPricePlan.Status: %+v", pricePlan.Status)

			u.metrics[structs.MetricOrderOSONewPricePlan].Inc()
		}
	}

	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderOCO)

	q := baseURL.Query()
	q.Set("symbol", pricePlan.Symbol)
	q.Set("side", pricePlan.Side)
	q.Set("quantity", fmt.Sprintf("%.5f", pricePlan.Status.Quantity))

	switch pricePlan.Side {
	case SideBuy:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceBUY))
		q.Set("stopPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceBUY))
		q.Set("stopLimitPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceBUY))
	case SideSell:
		q.Set("price", fmt.Sprintf("%.2f", pricePlan.PriceSELL))
		q.Set("stopPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceSELL))
		q.Set("stopLimitPrice", fmt.Sprintf("%.2f", pricePlan.StopPriceSELL))
	}

	q.Set("recvWindow", "60000")
	q.Set("stopLimitTimeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d000", time.Now().Unix()))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodPost, baseURL, nil, true)
	if err != nil {
		return err
	}

	var oList structs.OrderList
	var errMsg structs.Err

	if err := json.Unmarshal(req, &oList); err != nil {
		return err
	}

	if oList.OrderListID == 0 {
		if err := json.Unmarshal(req, &errMsg); err != nil {
			return err
		}

		switch errMsg.Code {
		case -2010:
			return structs.ErrTheRelationshipOfThePrices
		}

		return errors.New(errMsg.Msg)
	}

	o := models.Order{
		OrderID:     oList.OrderListID,
		SessionID:   pricePlan.Status.SessionID,
		Symbol:      oList.Symbol,
		ActualPrice: pricePlan.ActualPrice,
		Side:        pricePlan.Side,
		Quantity:    pricePlan.Status.Quantity,
		Type:        "OCO",
		Status:      "NEW",
		Try:         pricePlan.Status.OrderTry,
	}

	switch pricePlan.Side {
	case SideBuy:
		o.Price = pricePlan.PriceBUY
		o.StopPrice = pricePlan.StopPriceBUY
	case SideSell:
		o.Price = pricePlan.PriceSELL
		o.StopPrice = pricePlan.StopPriceSELL
	}

	if err := u.orderRepo.Store(&o); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) cancelOrder(symbol, orderId string) error {
	baseURL, err := url.Parse(u.url)
	if err != nil {
		return err
	}

	baseURL.Path = path.Join(orderUrlPath)

	q := baseURL.Query()
	q.Set("symbol", symbol)
	q.Set("orderId", orderId)
	q.Set("recvWindow", "60000")
	//q.Set("timeInForce", "GTC")
	q.Set("timestamp", fmt.Sprintf("%d", time.Now().Unix()*1000))

	sig := u.cryptoController.GetSignature(q.Encode())
	q.Set("signature", sig)

	baseURL.RawQuery = q.Encode()

	req, err := u.clientController.Send(http.MethodDelete, baseURL, nil, true)
	if err != nil {
		return err
	}

	type reqJson struct {
		Symbol              string `json:"symbol"`
		OrigClientOrderId   string `json:"origClientOrderId"`
		OrderId             int    `json:"orderId"`
		OrderListId         int    `json:"orderListId"`
		ClientOrderId       string `json:"clientOrderId"`
		Price               string `json:"price"`
		OrigQty             string `json:"origQty"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
		Status              string `json:"status"`
		TimeInForce         string `json:"timeInForce"`
		Type                string `json:"type"`
		Side                string `json:"side"`
	}

	var out reqJson

	if err := json.Unmarshal(req, &out); err != nil {
		return err
	}

	if out.OrderId != 0 {
		return nil
	}

	var errStruct struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(req, &errStruct); err != nil {
		return err
	}

	return nil
}
