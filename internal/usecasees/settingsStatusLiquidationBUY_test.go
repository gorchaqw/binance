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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/sirupsen/logrus"
)

type mockGenSettingsStatusLiquidationBUY struct {
	clientCtrl   *ctrlMocks.ClientCtrl
	cryptoCtrl   *ctrlMocks.CryptoCtrl
	tgmCtrl      *ctrlMocks.TgmCtrl
	orderRepo    *pgMocks.OrderRepo
	settingsRepo *mongoMocks.SettingsRepo
	priceRepo    *pgMocks.PriceRepo

	logger *logrus.Logger
}

func newMockGenSettingsStatusLiquidationBUY() *mockGenSettingsStatusLiquidationBUY {
	return &mockGenSettingsStatusLiquidationBUY{
		clientCtrl:   &ctrlMocks.ClientCtrl{},
		cryptoCtrl:   &ctrlMocks.CryptoCtrl{},
		tgmCtrl:      &ctrlMocks.TgmCtrl{},
		orderRepo:    &pgMocks.OrderRepo{},
		settingsRepo: &mongoMocks.SettingsRepo{},
		priceRepo:    &pgMocks.PriceRepo{},
	}
}

func (mockGen *mockGenSettingsStatusLiquidationBUY) initLogger() {
	mockGen.logger = logrus.New()
	mockGen.logger.SetLevel(logrus.DebugLevel)
}

func (mockGen *mockGenSettingsStatusLiquidationBUY) run(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(1)

	mockGen.initLogger()

	mockGen.clientMocks()
	mockGen.tgmMocks()
	mockGen.cryptoMocks()
	mockGen.orderMocks()
	mockGen.settingsMocks()

	assert.NoError(t, mockGen.initOrderUseCase().Monitoring("BTCBUSD"))

	wg.Wait()
}

func (mockGen *mockGenSettingsStatusLiquidationBUY) clientMocks() {
	orderListStruct := orderStructs.OrderList{
		OrderListID: 1,
	}
	orderListJson, _ := json.Marshal(&orderListStruct)
	//
	orderStruct := orderStructs.Order{
		Symbol:  "BTCBUSD",
		OrderId: 1,
	}
	orderJson, _ := json.Marshal(&orderStruct)
	//
	openOrdersStruct := []orderStructs.Order{}
	openOrdersJson, _ := json.Marshal(&openOrdersStruct)

	limitOrderStruct := orderStructs.LimitOrder{
		OrderID: 1,
	}
	limitOrderJson, _ := json.Marshal(&limitOrderStruct)
	//

	mockGen.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/orderList"
	}), []byte(nil), true).Return(orderListJson, nil)

	mockGen.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(orderJson, nil)

	mockGen.clientCtrl.On("Send", "GET", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/openOrders"
	}), []byte(nil), true).Return(openOrdersJson, nil)

	mockGen.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order/oco"
	}), []byte(nil), true).Return(orderListJson, nil)

	mockGen.clientCtrl.On("Send", "POST", mock.MatchedBy(func(input *url.URL) bool {
		return input.Path == "/api/v3/order"
	}), []byte(nil), true).Return(limitOrderJson, nil)
}
func (mockGen *mockGenSettingsStatusLiquidationBUY) tgmMocks() {
	mockGen.tgmCtrl.On("Send", mock.AnythingOfType("string")).Return(nil)
}
func (mockGen *mockGenSettingsStatusLiquidationBUY) cryptoMocks() {
	mockGen.cryptoCtrl.On("GetSignature", mock.AnythingOfType("string")).Return("630e26f39d6728d0e7feffb9", nil)

}
func (mockGen *mockGenSettingsStatusLiquidationBUY) orderMocks() {
	mockGen.orderRepo.On("Store", mock.AnythingOfType("*models.Order")).
		Return(nil)

	mockGen.orderRepo.On("GetLast", "BTCBUSD").
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
}
func (mockGen *mockGenSettingsStatusLiquidationBUY) settingsMocks() {
	mockGen.settingsRepo.On("UpdateStatus", mock.AnythingOfType("primitive.ObjectID"), structs.LiquidationSELL).
		Return(nil)

	mockGen.settingsRepo.On("Load", "BTCBUSD").
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

func (mockGen *mockGenSettingsStatusLiquidationBUY) initOrderUseCase() *orderUseCase {
	return NewOrderUseCase(
		mockGen.clientCtrl,
		mockGen.cryptoCtrl,
		mockGen.tgmCtrl,
		mockGen.settingsRepo,
		mockGen.orderRepo,
		mockGen.initPriceUseCase(),
		"https://api.binance.com",
		mockGen.logger,
	)
}
func (mockGen *mockGenSettingsStatusLiquidationBUY) initPriceUseCase() *priceUseCase {
	return NewPriceUseCase(
		mockGen.clientCtrl,
		mockGen.tgmCtrl,
		mockGen.priceRepo,
		"https://api.binance.com",
		mockGen.logger,
	)
}
