package sqlite

import (
	"binance/models"
	"database/sql"
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

func (r *PriceRepository) GetByCreatedByInterval(symbol string, sTime, eTime time.Time) ([]models.Price, error) {
	var out []models.Price

	if err := r.conn.Select(&out, "SELECT * FROM prices where created_at > $1 AND created_at < $2 AND symbol = $3;", sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"), symbol); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PriceRepository) GetMaxMinByCreatedByInterval(symbol string, sTime, eTime time.Time) (float64, float64, error) {
	var max, min sql.NullFloat64

	if err := r.conn.QueryRowx("SELECT max(price),min(price) FROM prices where created_at > $1 AND created_at < $2 AND symbol = $3;", sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"), symbol).Scan(&max, &min); err != nil {
		return 0, 0, err
	}

	return max.Float64, min.Float64, nil
}

func (r *PriceRepository) GetMaxByCreatedByInterval(symbol string, sTime, eTime time.Time) (float64, time.Time, error) {
	var max sql.NullFloat64
	var createdAt sql.NullTime

	if err := r.conn.QueryRowx("SELECT max(price),created_at FROM prices where created_at >= $1 AND created_at <= $2 AND symbol = $3;", sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"), symbol).Scan(&max, &createdAt); err != nil {
		return 0, createdAt.Time, err
	}

	return max.Float64, createdAt.Time, nil
}

func (r *PriceRepository) GetMinByCreatedByInterval(symbol string, sTime, eTime time.Time) (float64, time.Time, error) {
	var min sql.NullFloat64
	var createdAt sql.NullTime

	if err := r.conn.QueryRowx("SELECT min(price),created_at FROM prices where created_at >= $1 AND created_at <= $2 AND symbol = $3;", sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"), symbol).Scan(&min, &createdAt); err != nil {
		return 0, createdAt.Time, err
	}

	return min.Float64, createdAt.Time, nil
}

func (r *PriceRepository) Store(m *models.Price) (err error) {

	if _, err := r.conn.NamedExec("INSERT INTO prices (symbol,price) VALUES (:symbol,:price)", m); err != nil {
		return err
	}

	return nil
}
