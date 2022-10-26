package postgres_test

import (
	"binance/internal/repository/postgres"
	"binance/models"
	"fmt"
	"math/rand"
	"time"

	"log"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	_ "github.com/lib/pq"
)

type PGTest struct {
	conn *sqlx.DB
}

func initPGTest() *PGTest {
	var out PGTest
	db, err := sqlx.Connect("postgres", "host=localhost user=binance password=binance dbname=binance sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}

	out.conn = db

	return &out
}

func Test_GetBySessionID(t *testing.T) {
	c := initPGTest()
	pgStore := postgres.NewOrderRepository(c.conn, postgres.Features)

	oList, err := pgStore.GetBySessionID("cc3336da-432f-4e9e-9152-d976732f9b8d")
	assert.NoError(t, err)

	fmt.Printf("%+v", oList)
}

func Test_OrderStore(t *testing.T) {
	c := initPGTest()
	pgStore := postgres.NewOrderRepository(c.conn, postgres.Features)

	rand.Seed(time.Now().UnixNano())

	var lastID, firstID string
	symbol := "BTCBUSD"

	t.Run("Store", func(t *testing.T) {
		err := pgStore.Store(&models.Order{
			ID:          uuid.NewString(),
			OrderID:     rand.Int63(),
			SessionID:   uuid.New().String(),
			Symbol:      symbol,
			Side:        "BUY",
			Quantity:    rand.Float64(),
			Price:       rand.Float64(),
			ActualPrice: rand.Float64(),
			StopPrice:   rand.Float64(),
			Status:      "NEW",
			Try:         1,
			Type:        "OCO",
			CreatedAt:   time.Now(),
		})

		assert.NoError(t, err)
	})

	t.Run("GetLast", func(t *testing.T) {
		o, err := pgStore.GetLast(symbol)
		assert.NoError(t, err)

		assert.Equal(t, o.Side, "BUY")
		lastID = o.ID

		t.Logf("%+v", o)
	})

	t.Run("GetFirst", func(t *testing.T) {
		o, err := pgStore.GetFirst(symbol)
		assert.NoError(t, err)

		assert.Equal(t, o.Side, "BUY")
		firstID = o.ID

		t.Logf("%+v", o)
	})

	t.Run("GetByID", func(t *testing.T) {
		f, err := pgStore.GetByID(firstID)
		assert.NoError(t, err)

		t.Logf("%+v", f)

		l, err := pgStore.GetByID(lastID)
		assert.NoError(t, err)

		t.Logf("%+v", l)
	})

}
