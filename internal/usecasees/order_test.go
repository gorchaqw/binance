package usecasees_test

import (
	ctrlMocks "binance/internal/controllers/mocks"
	mongoMocks "binance/internal/repository/mongo/mocks"
	"binance/internal/repository/mongo/structs"
	pgMocks "binance/internal/repository/postgres/mocks"
	"binance/internal/usecasees"
	orderStructs "binance/internal/usecasees/structs"
	"binance/models"
	"os"

	"encoding/json"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const testCaseSELLOrder = "SELLOrder"

type mockGen struct {
	clientCtrl *ctrlMocks.ClientCtrl
}

type mockGenSELLOrder struct {
	clientCtrl   *ctrlMocks.ClientCtrl
	cryptoCtrl   *ctrlMocks.CryptoCtrl
	tgmCtrl      *ctrlMocks.TgmCtrl
	orderRepo    *pgMocks.OrderRepo
	settingsRepo *mongoMocks.SettingsRepo
	priceRepo    *pgMocks.PriceRepo
}

func Test_OrderUseCase_SELLOrder(t *testing.T) {
	var wg sync.WaitGroup
	var mockGen mockGenSELLOrder

	wg.Add(1)

	logger := &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.DebugLevel,
	}

	mockGen.clientMocks()
	mockGen.tgmMocks()
	mockGen.cryptoMocks()
	mockGen.orderMocks()
	mockGen.settingsMocks()

	priceRepo := &pgMocks.PriceRepo{}

	priceUseCase := usecasees.NewPriceUseCase(
		mockGen.clientCtrl,
		mockGen.tgmCtrl,
		priceRepo,
		"https://api.binance.com",
		logger,
	)

	orderUseCase := usecasees.NewOrderUseCase(
		mockGen.clientCtrl,
		mockGen.cryptoCtrl,
		mockGen.tgmCtrl,
		mockGen.settingsRepo,
		mockGen.orderRepo,
		priceUseCase,
		"https://api.binance.com",
		logger,
	)

	assert.NoError(t, orderUseCase.Monitoring("BTCBUSD"))

	wg.Wait()
}

func (m *mockGenSELLOrder) clientMocks() {
	m.clientCtrl = &ctrlMocks.ClientCtrl{}

	orderListStruct := orderStructs.OrderList{
		OrderListID: 1,
	}
	orderListJson, _ := json.Marshal(&orderListStruct)

	orderStruct := orderStructs.Order{
		Symbol:  "BTCBUSD",
		OrderId: 1,
	}
	orderJson, _ := json.Marshal(&orderStruct)

	openOrdersStruct := []orderStructs.Order{}
	openOrdersJson, _ := json.Marshal(&openOrdersStruct)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/orderList"
	}), []byte(nil), true).Return(orderListJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(orderJson, nil)

	m.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/openOrders"
	}), []byte(nil), true).Return(openOrdersJson, nil)

	m.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order/oco"
	}), []byte(nil), true).Return(orderListJson, nil)

}

func (m *mockGenSELLOrder) tgmMocks() {
	m.tgmCtrl = &ctrlMocks.TgmCtrl{}

	m.tgmCtrl.On("Send", mock.AnythingOfType("string")).Return(nil)
}

func (m *mockGenSELLOrder) cryptoMocks() {
	m.cryptoCtrl = &ctrlMocks.CryptoCtrl{}
	m.cryptoCtrl.On("GetSignature", mock.AnythingOfType("string")).Return("630e26f39d6728d0e7feffb9", nil)

}
func (m *mockGenSELLOrder) orderMocks() {
	m.orderRepo = &pgMocks.OrderRepo{}

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
}

func (m *mockGenSELLOrder) settingsMocks() {
	m.settingsRepo = &mongoMocks.SettingsRepo{}

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

func (m *mockGenSELLOrder) priceMocks() {
	m.priceRepo = &pgMocks.PriceRepo{}
}
