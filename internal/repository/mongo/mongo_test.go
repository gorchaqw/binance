package mongo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSetDefault(t *testing.T) {
	credential := options.Credential{
		Username: "binance",
		Password: "binance",
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017").SetAuth(credential))
	assert.NoError(t, err)

	repo := NewSettingsRepository(client)

	assert.NoError(t, repo.SetDefault())

	s, err := repo.Load("BTCBUSD")
	assert.NoError(t, err)

	fmt.Println("Wait")

	time.Sleep(20 * time.Second)

	fmt.Println("Go")

	assert.NoError(t, repo.ReLoad(s))

	fmt.Println(s)
}
