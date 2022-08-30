package mongo

import (
	"binance/internal/repository/mongo/structs"
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SettingsRepository struct {
	conn       *mongo.Client
	collection *mongo.Collection
}

func NewSettingsRepository(conn *mongo.Client) SettingsRepo {
	collection := conn.Database("settings").Collection("symbols")

	return &SettingsRepository{conn: conn, collection: collection}
}

func (r *SettingsRepository) SetDefault() error {
	symbols := []structs.Settings{
		{
			Symbol:    "BTCUSDT",
			Limit:     0.02,
			Step:      0.0006,
			Delta:     0.25,
			DeltaStep: 0.065,
			SpotURL:   "https://www.binance.com/ru/trade/BTC_USDT?theme=dark&type=spot",
			Status:    structs.Enabled.ToString(),
		},
	}

	for _, symbol := range symbols {
		check, err := r.Load(symbol.Symbol)
		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}

		if primitive.ObjectID.IsZero(check.ID) {
			_, err := r.collection.InsertOne(context.TODO(), symbol)
			if err != nil {
				return err
			}
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

func (r *SettingsRepository) ReLoad(settings *structs.Settings) error {

	if err := r.collection.FindOne(context.TODO(), bson.D{{"symbol", settings.Symbol}}).Decode(&settings); err != nil {
		return err
	}

	return nil
}

func (r *SettingsRepository) UpdateStatus(id primitive.ObjectID, status structs.SymbolStatus) error {
	_, err := r.collection.UpdateOne(
		context.TODO(),
		bson.D{{"_id", id}},
		bson.D{{"$set", bson.D{{"status", status}}}},
	)
	if err != nil {
		return err
	}

	return nil
}
