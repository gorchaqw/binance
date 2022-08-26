package structs

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SymbolStatus string

const (
	Disabled    SymbolStatus = "DISABLED"
	Enabled     SymbolStatus = "ENABLED"
	Liquidation SymbolStatus = "LIQUIDATION"
)

func (s SymbolStatus) ToString() string {
	return fmt.Sprintf("%s", s)
}

type Settings struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Symbol  string             `bson:"symbol"`
	Limit   float64            `bson:"limit"`
	Step    float64            `bson:"step"`
	Delta   float64            `bson:"delta"`
	SpotURL string             `bson:"spot_url"`
	Status  string             `bson:"status"`
}
