package sqlite

import (
	"binance/models"
	"github.com/jmoiron/sqlx"
	"time"
)

type PriceRepository struct {
	conn *sqlx.DB
}

func NewPriceRepository(conn *sqlx.DB) *PriceRepository {
	return &PriceRepository{
		conn: conn,
	}
}

func (r *PriceRepository) GetByCreatedByInterval(sTime, eTime time.Time) ([]models.Price, error) {
	var out []models.Price

	if err := r.conn.Select(&out, "SELECT * FROM prices where created_at > $1 AND created_at < $2;", sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05")); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PriceRepository) Store(m *models.Price) (err error) {

	if _, err := r.conn.NamedExec("INSERT INTO prices (symbol,price) VALUES (:symbol,:price)", m); err != nil {
		return err
	}

	return nil
}
