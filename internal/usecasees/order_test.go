package usecasees

import (
	ctrlMocks "binance/internal/controllers/mocks"
	mongoMocks "binance/internal/repository/mongo/mocks"
	"binance/internal/repository/mongo/structs"
	pgMocks "binance/internal/repository/postgres/mocks"
	orderStructs "binance/internal/usecasees/structs"
	"binance/models"

	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type testCaseStruct struct {
	Label string
	Mocks *testCaseMocks
}
type testCaseMocks struct {
	clientCtrl   *ctrlMocks.ClientCtrl
	cryptoCtrl   *ctrlMocks.CryptoCtrl
	tgmCtrl      *ctrlMocks.TgmCtrl
	orderRepo    *pgMocks.OrderRepo
	settingsRepo *mongoMocks.SettingsRepo
	priceRepo    *pgMocks.PriceRepo

	mockStructs *mockStructs

	logRus *logrus.Logger
}
type mockStructs struct {
	orderListJson  []byte
	orderJson      []byte
	openOrdersJson []byte
	limitOrderJson []byte
}

const (
	testCaseOrderSELL       = "order_sell"
	testCaseLiquidationBUY  = "liquidation_buy"
	testCaseLiquidationSELL = "liquidation_sell"
	testSettingsStatusNEW   = "settings_new"
)

func Test_OrderUseCase(t *testing.T) {
	t.Run("order SELL", func(t *testing.T) {
		newMonitoring(testCaseOrderSELL).run(t)
	})

	t.Run("liquidation BUY", func(t *testing.T) {
		newMonitoring(testCaseLiquidationBUY).run(t)
	})

	t.Run("liquidation SELL", func(t *testing.T) {
		newMonitoring(testCaseLiquidationSELL).run(t)
	})

	t.Run("settings status NEW", func(t *testing.T) {
		newMonitoring(testSettingsStatusNEW).run(t)
	})
}

func newMonitoring(label string) *testCaseStruct {
	return &testCaseStruct{
		Label: label,
		Mocks: &testCaseMocks{
			clientCtrl:   &ctrlMocks.ClientCtrl{},
			cryptoCtrl:   &ctrlMocks.CryptoCtrl{},
			tgmCtrl:      &ctrlMocks.TgmCtrl{},
			orderRepo:    &pgMocks.OrderRepo{},
			settingsRepo: &mongoMocks.SettingsRepo{},
			priceRepo:    &pgMocks.PriceRepo{},
			mockStructs:  &mockStructs{},
		},
	}
}

func (c *testCaseStruct) run(t *testing.T) {
	c.Mocks.initBaseMocks()
	c.initMockStructs(t)

	switch c.Label {
	case testCaseOrderSELL:
		c.Mocks.initOrderSELLMocks()
	case testCaseLiquidationBUY:
		c.Mocks.initLiquidationBUYMocks()
	case testCaseLiquidationSELL:
		c.Mocks.initLiquidationSELLMocks()
	case testSettingsStatusNEW:
		c.Mocks.initSettingsStatusNewMocks()
	}

	assert.NoError(t, c.initOrderUseCase().Monitoring("BTCBUSD"))

	time.Sleep(5 * time.Second)
}

func (m *testCaseMocks) initBaseMocks() {
	// LogRus mocks
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	m.logRus = logger

	// Tgm mocks
	m.tgmCtrl.On("Send", mock.AnythingOfType("string")).Return(nil)

	// Crypto mocks
	m.cryptoCtrl.On("GetSignature", mock.AnythingOfType("string")).Return("630e26f39d6728d0e7feffb9", nil)

}

func (m *testCaseMocks) initOrderSELLMocks() {
	// Client Mocks
	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/orderList"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(m.mockStructs.orderJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/openOrders"
	}), []byte(nil), true).Return(m.mockStructs.openOrdersJson, nil)

	m.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order/oco"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	//Order Mocks
	m.orderRepo.On("Store", mock.AnythingOfType("*models.Order")).Return(nil)
	m.orderRepo.On("GetLast", "BTCBUSD").Return(&models.Order{
		ID:          8,
		OrderID:     3145000133794585251,
		SessionID:   "cc3336da-432f-4e9e-9152-d976732f9b8d",
		Symbol:      "BTCBUSD",
		Side:        "SELL",
		Quantity:    0.0006,
		Price:       20500,
		ActualPrice: 19632,
		StopPrice:   19000,
		Status:      "FILLED",
		Try:         1,
		Type:        "OCO",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	}, nil)

	// Settings Mocks

	m.settingsRepo.On("Load", "BTCBUSD").Return(&structs.Settings{
		ID:        primitive.NewObjectID(),
		Symbol:    "BTCBUSD",
		Limit:     0.02,
		Step:      0.0006,
		Delta:     0.2,
		DeltaStep: 0.065,
		SpotURL:   "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",
		Status:    structs.Enabled.ToString(),
	}, nil)
}
func (m *testCaseMocks) initLiquidationBUYMocks() {
	// Client Mocks
	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/orderList"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(m.mockStructs.orderJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/openOrders"
	}), []byte(nil), true).Return(m.mockStructs.openOrdersJson, nil)

	m.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order/oco"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	m.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(m.mockStructs.limitOrderJson, nil)

	//Order Mocks
	m.orderRepo.On("Store", mock.AnythingOfType("*models.Order")).
		Return(nil)

	m.orderRepo.On("GetLast", "BTCBUSD").
		Return(&models.Order{
			ID:          8,
			OrderID:     3145000133794585251,
			SessionID:   "cc3336da-432f-4e9e-9152-d976732f9b8d",
			Symbol:      "BTCBUSD",
			Side:        "SELL",
			Quantity:    0.0006,
			Price:       20500,
			ActualPrice: 19632,
			StopPrice:   19000,
			Status:      "FILLED",
			Try:         1,
			Type:        "OCO",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
		}, nil)

	// Settings Mocks
	m.settingsRepo.On("UpdateStatus", mock.AnythingOfType("primitive.ObjectID"), structs.LiquidationSELL).
		Return(nil)

	m.settingsRepo.On("Load", "BTCBUSD").
		Return(&structs.Settings{
			ID:        primitive.NewObjectID(),
			Symbol:    "BTCBUSD",
			Limit:     0.02,
			Step:      0.0006,
			Delta:     0.2,
			DeltaStep: 0.065,
			SpotURL:   "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",
			Status:    structs.LiquidationBUY.ToString(),
		}, nil)
}
func (m *testCaseMocks) initLiquidationSELLMocks() {
	// Client Mocks
	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/orderList"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(m.mockStructs.orderJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/openOrders"
	}), []byte(nil), true).Return(m.mockStructs.openOrdersJson, nil)

	m.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order/oco"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	// Order Mocks
	m.orderRepo.On("Store", mock.AnythingOfType("*models.Order")).
		Return(nil)

	m.orderRepo.On("GetLast", "BTCBUSD").
		Return(&models.Order{
			ID:          8,
			OrderID:     3145000133794585251,
			SessionID:   "cc3336da-432f-4e9e-9152-d976732f9b8d",
			Symbol:      "BTCBUSD",
			Side:        "SELL",
			Quantity:    0.0006,
			Price:       20500,
			ActualPrice: 19632,
			StopPrice:   19000,
			Status:      "FILLED",
			Try:         1,
			Type:        "OCO",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
		}, nil)

	// Settings Mocks
	m.settingsRepo.On("UpdateStatus", mock.AnythingOfType("primitive.ObjectID"), structs.Enabled).
		Return(nil)

	m.settingsRepo.On("Load", "BTCBUSD").
		Return(&structs.Settings{
			ID:        primitive.NewObjectID(),
			Symbol:    "BTCBUSD",
			Limit:     0.02,
			Step:      0.0006,
			Delta:     0.2,
			DeltaStep: 0.065,
			SpotURL:   "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",
			Status:    structs.LiquidationSELL.ToString(),
		}, nil)
}
func (m *testCaseMocks) initSettingsStatusNewMocks() {
	// Client Mocks
	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/orderList"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(m.mockStructs.orderJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/openOrders"
	}), []byte(nil), true).Return(m.mockStructs.openOrdersJson, nil)

	m.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order/oco"
	}), []byte(nil), true).Return(m.mockStructs.orderListJson, nil)

	// Order Mocks
	m.orderRepo.On("Store", mock.AnythingOfType("*models.Order")).
		Return(nil)

	m.orderRepo.On("GetLast", "BTCBUSD").
		Return(&models.Order{
			ID:          8,
			OrderID:     3145000133794585251,
			SessionID:   "cc3336da-432f-4e9e-9152-d976732f9b8d",
			Symbol:      "BTCBUSD",
			Side:        "SELL",
			Quantity:    0.0006,
			Price:       20500,
			ActualPrice: 19632,
			StopPrice:   19000,
			Status:      "FILLED",
			Try:         1,
			Type:        "OCO",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
		}, nil)

	// Settings Mocks
	m.settingsRepo.On("UpdateStatus", mock.AnythingOfType("primitive.ObjectID"), structs.Enabled).
		Return(nil)

	m.settingsRepo.On("Load", "BTCBUSD").
		Return(&structs.Settings{
			ID:        primitive.NewObjectID(),
			Symbol:    "BTCBUSD",
			Limit:     0.02,
			Step:      0.0006,
			Delta:     0.2,
			DeltaStep: 0.065,
			SpotURL:   "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",
			Status:    structs.New.ToString(),
		}, nil)
}

func (c *testCaseStruct) initMockStructs(t *testing.T) {
	switch c.Label {
	default:
		// orderList
		orderListStruct := orderStructs.OrderList{
			OrderListID: 1,
		}
		orderListJson, err := json.Marshal(&orderListStruct)
		assert.NoError(t, err)

		c.Mocks.mockStructs.orderListJson = orderListJson

		// order
		orderStruct := orderStructs.Order{
			Symbol:  "BTCBUSD",
			OrderId: 1,
		}
		orderJson, err := json.Marshal(&orderStruct)
		assert.NoError(t, err)

		c.Mocks.mockStructs.orderJson = orderJson

		// openOrders
		openOrdersStruct := []orderStructs.Order{orderStruct}
		openOrdersJson, err := json.Marshal(&openOrdersStruct)
		assert.NoError(t, err)

		c.Mocks.mockStructs.openOrdersJson = openOrdersJson

		// limitOrder
		limitOrderStruct := orderStructs.LimitOrder{
			OrderID: 1,
		}
		limitOrderJson, err := json.Marshal(&limitOrderStruct)
		assert.NoError(t, err)

		c.Mocks.mockStructs.limitOrderJson = limitOrderJson
	}
}
func (c *testCaseStruct) initOrderUseCase() *orderUseCase {
	return nil
}
func (c *testCaseStruct) initPriceUseCase() *priceUseCase {
	return NewPriceUseCase(
		c.Mocks.clientCtrl,
		c.Mocks.tgmCtrl,
		c.Mocks.priceRepo,
		"https://api.binance.com",
		c.Mocks.logRus,
	)
}
