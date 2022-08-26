package mongo

import (
	"binance/internal/repository/mongo/structs"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SettingsRepository struct {
	conn       *mongo.Client
	collection *mongo.Collection
}

func NewSettingsRepository(conn *mongo.Client) *SettingsRepository {
	collection := conn.Database("settings").Collection("symbols")

	return &SettingsRepository{conn: conn, collection: collection}
}

func (r *SettingsRepository) SetDefault() error {
	symbols := []structs.Settings{
		{
			Symbol:  "BTCBUSD",
			Limit:   0.014,
			Step:    0.0005,
			Delta:   0.5,
			SpotURL: "https://www.binance.com/ru/trade/BTC_BUSD?theme=dark&type=spot",
		},
		{
			Symbol:  "ETHRUB",
			Limit:   0.19,
			Step:    0.007,
			Delta:   0.7,
			SpotURL: "https://www.binance.com/ru/trade/ETH_RUB?theme=dark&type=spot",
		},
		{
			Symbol:  "ETHBUSD",
			Limit:   0.19,
			Step:    0.007,
			Delta:   0.7,
			SpotURL: "https://www.binance.com/ru/trade/ETH_RUB?theme=dark&type=spot",
		},
	}

	for _, symbol := range symbols {
		check, err := r.Load(symbol.Symbol)
		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}

		if primitive.ObjectID.IsZero(check.ID) {
			result, err := r.collection.InsertOne(context.TODO(), symbol)
			if err != nil {
				return err
			}

			fmt.Println(result.InsertedID)
		}
	}

	return nil
}

func (r *SettingsRepository) Load(symbol string) (*structs.Settings, error) {
	var result structs.Settings

	if err := r.collection.FindOne(context.TODO(), bson.D{{"symbol", symbol}}).Decode(&result); err != nil {
		return &result, err
	}

	return &result, nil

}
