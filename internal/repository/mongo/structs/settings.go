package structs

import "go.mongodb.org/mongo-driver/bson/primitive"

type Settings struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Symbol string             `bson:"symbol"`
	Limit  float64            `bson:"limit"`
	Step   float64            `bson:"step"`
	Delta  float64            `bson:"delta"`
}
