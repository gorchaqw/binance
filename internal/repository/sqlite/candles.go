package sqlite

import (
	"binance/models"
	"github.com/jmoiron/sqlx"
)

type CandleRepository struct {
	conn *sqlx.DB
}

func NewCandlesRepository(conn *sqlx.DB) *CandleRepository {
	return &CandleRepository{
		conn: conn,
	}
}

func (r *CandleRepository) Store(m *models.Candle) (err error) {

	if _, err := r.conn.NamedExec("INSERT INTO candles (symbol,open_price,close_price,max_price,min_price,time_frame,open_time,close_time) VALUES (:symbol,:open_price,:close_price,:max_price,:min_price,:time_frame,:open_time,:close_time)", m); err != nil {
		return err
	}

	return nil
}

func (r *CandleRepository) GetLast(symbol string) (*models.Candle, error) {
	var candle models.Candle
	if err := r.conn.QueryRowx("SELECT * FROM candles WHERE symbol = $1 ORDER BY id DESC LIMIT 1", symbol).StructScan(&candle); err != nil {
		return nil, err
	}

	return &candle, nil
}
