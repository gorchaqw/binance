package postgres

import (
	"binance/models"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type PriceRepository struct {
	conn *sqlx.DB
}

func NewPriceRepository(conn *sqlx.DB) PriceRepo {
	return &PriceRepository{
		conn: conn,
	}
}

func (r *PriceRepository) GetByCreatedByInterval(symbol string, sTime, eTime time.Time) ([]models.Price, error) {
	var out []models.Price

	if err := r.conn.Select(&out, "SELECT * FROM prices where created_at > $1 AND created_at < $2 AND symbol = $3;", sTime.UTC().Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"), symbol); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PriceRepository) GetMaxMinByCreatedByInterval(symbol string, sTime, eTime time.Time) (float64, float64, float64, float64, error) {
	var maxID, minID uint
	var maxPrice, minPrice sql.NullFloat64

	if err := r.conn.QueryRowx("SELECT max(id),min(id),max(price),min(price) FROM prices where created_at > $1 AND created_at < $2 AND symbol = $3;", sTime.UTC(), eTime.UTC(), symbol).Scan(&maxID, &minID, &maxPrice, &minPrice); err != nil {
		return 0, 0, 0, 0, err
	}

	openPrice, err := r.GetByID(symbol, minID)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	closePrice, err := r.GetByID(symbol, maxID)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return openPrice.Price, closePrice.Price, maxPrice.Float64, minPrice.Float64, nil
}

func (r *PriceRepository) Store(m *models.Price) (err error) {

	if _, err := r.conn.NamedExec("INSERT INTO prices (symbol,price) VALUES (:symbol,:price)", m); err != nil {
		return err
	}

	return nil
}

func (r *PriceRepository) GetLast(symbol string, sTime, eTime time.Time) (*models.Price, error) {
	var price models.Price
	if err := r.conn.QueryRowx("SELECT * FROM prices where created_at >= $1 AND created_at <= $2 AND symbol = $3 ORDER BY id DESC LIMIT 1", sTime.UTC(), eTime.UTC(), symbol).StructScan(&price); err != nil {
		return nil, err
	}

	return &price, nil
}

func (r *PriceRepository) GetByID(symbol string, id uint) (*models.Price, error) {
	var price models.Price
	if err := r.conn.QueryRowx("SELECT * FROM prices where id = $1 AND  symbol = $2 ORDER BY id DESC LIMIT 1", id, symbol).StructScan(&price); err != nil {
		return nil, err
	}

	return &price, nil
}
