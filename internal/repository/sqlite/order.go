package sqlite

import (
	"binance/models"
	"github.com/jmoiron/sqlx"
)

type OrderRepository struct {
	conn *sqlx.DB
}

func NewOrderRepository(conn *sqlx.DB) *OrderRepository {
	return &OrderRepository{
		conn: conn,
	}
}

func (r *OrderRepository) Store(m *models.Order) (err error) {

	if _, err := r.conn.NamedExec("INSERT INTO orders (orderId,symbol,side,quantity,price) VALUES (:orderId,:symbol,:side,:quantity,:price)", m); err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) GetLast() (*models.Order, error) {
	var order models.Order
	if err := r.conn.QueryRowx("SELECT * FROM orders ORDER BY created_at DESC LIMIT 1").StructScan(&order); err != nil {
		return nil, err
	}

	return &order, nil
}
