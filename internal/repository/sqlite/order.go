package sqlite

import (
	"binance/models"
	"github.com/jmoiron/sqlx"
)

const (
	ORDER_STATUS_NEW      = "NEW"
	ORDER_STATUS_CANCELED = "CANCELED"
	ORDER_STATUS_COMPLETE = "COMPLETE"
	ORDER_STATUS_FILLED   = "FILLED"
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
	if _, err := r.conn.NamedExec("INSERT INTO orders (order_id,symbol,side,quantity,price,stop_price,status,type) VALUES (:order_id,:symbol,:side,:quantity,:price,:stop_price,:status,:type)", m); err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) GetLast(symbol string) (*models.Order, error) {
	var order models.Order
	if err := r.conn.QueryRowx("SELECT * FROM orders WHERE symbol = $1 ORDER BY created_at DESC LIMIT 1", symbol).StructScan(&order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepository) SetActualPrice(id int, price float64) error {
	if _, err := r.conn.Exec("UPDATE orders SET price = $1 where id = $2;", price, id); err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) SetStatus(id int, status string) error {
	if _, err := r.conn.Exec("UPDATE orders SET status = $1 where id = $2;", status, id); err != nil {
		return err
	}

	return nil
}
