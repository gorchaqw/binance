package mongo

import (
	"binance/internal/repository/mongo/structs"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

//go:generate mockery --case=snake --name=SettingsRepo

type SettingsRepo interface {
	SetDefault() error
	Load(symbol string) (*structs.Settings, error)
	ReLoad(settings *structs.Settings) error
	UpdateStatus(id primitive.ObjectID, status structs.SymbolStatus) error
	UpdateDepthLimit(id primitive.ObjectID, depthLimit float64) error
}
