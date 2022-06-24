package sqlite

import (
	"binance/models"
	"context"

	"github.com/jmoiron/sqlx"
)

type PriceRepository struct {
	conn *sqlx.DB
}

func NewPriceRepository(conn *sqlx.DB) *PriceRepository {
	return &PriceRepository{
		conn: conn,
	}
}

func (r *PriceRepository) Store(ctx context.Context, m *models.Price) (err error) {

	_, err = r.conn.NamedExec(`INSERT prices user (symbol,price) VALUES (:symbol,:price)`, m)

	query := `INSERT prices SET symbol=? , price=?`
	stmt, err := r.conn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}

	if _, err := stmt.ExecContext(ctx, m.Symbol, m.Price); err != nil {
		return err
	}

	return nil
}
