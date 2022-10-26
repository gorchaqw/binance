package postgres

import (
	"time"

	"binance/models"

	"github.com/jmoiron/sqlx"
)

const (
	Spot     = "SPOT"
	Features = "FEATURES"
)

type OrderRepository struct {
	conn  *sqlx.DB
	table string
}

func NewOrderRepository(conn *sqlx.DB, table string) OrderRepo {
	return &OrderRepository{
		conn:  conn,
		table: table,
	}
}

func (r *OrderRepository) Store(m *models.Order) error {
	switch r.table {
	case Spot:
		if _, err := r.conn.NamedExec("INSERT INTO orders (id,order_id,session_id,symbol,side,quantity,actual_price,price,stop_price,status,type,try) VALUES (:id,:order_id,:session_id,:symbol,:side,:quantity,:actual_price,:price,:stop_price,:status,:type,:try)", m); err != nil {
			return err
		}
	case Features:
		if _, err := r.conn.NamedExec("INSERT INTO features_orders (id,order_id,session_id,symbol,side,quantity,actual_price,price,stop_price,status,type,try) VALUES (:id,:order_id,:session_id,:symbol,:side,:quantity,:actual_price,:price,:stop_price,:status,:type,:try)", m); err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderRepository) GetLast(symbol string) (*models.Order, error) {
	var order models.Order

	switch r.table {
	case Spot:
		if err := r.conn.QueryRowx("SELECT * FROM orders WHERE symbol = $1 AND type = 'MARKET' ORDER BY created_at DESC LIMIT 1", symbol).StructScan(&order); err != nil {
			return nil, err
		}
	case Features:
		if err := r.conn.QueryRowx("SELECT * FROM features_orders WHERE symbol = $1 AND type = 'MARKET'  ORDER BY created_at DESC LIMIT 1", symbol).StructScan(&order); err != nil {
			return nil, err
		}
	}

	return &order, nil
}

func (r *OrderRepository) GetFirst(symbol string) (*models.Order, error) {
	var order models.Order

	switch r.table {
	case Spot:
		if err := r.conn.QueryRowx("SELECT * FROM orders WHERE symbol = $1 AND side = $2 AND try = $3 ORDER BY id DESC LIMIT 1", symbol, "BUY", 1).StructScan(&order); err != nil {
			return nil, err
		}
	case Features:
		if err := r.conn.QueryRowx("SELECT * FROM features_orders WHERE symbol = $1 AND side = $2 AND try = $3 ORDER BY id DESC LIMIT 1", symbol, "BUY", 1).StructScan(&order); err != nil {
			return nil, err
		}
	}

	return &order, nil
}

func (r *OrderRepository) GetByID(id string) (*models.Order, error) {
	var order models.Order

	switch r.table {
	case Spot:
		if err := r.conn.QueryRowx("SELECT * FROM orders WHERE id = $1 LIMIT 1", id).StructScan(&order); err != nil {
			return nil, err
		}
	case Features:
		if err := r.conn.QueryRowx("SELECT * FROM features_orders WHERE id = $1 LIMIT 1", id).StructScan(&order); err != nil {
			return nil, err
		}
	}

	return &order, nil
}

func (r *OrderRepository) GetBySessionID(sessionID string) ([]models.Order, error) {
	var orders []models.Order

	switch r.table {
	case Spot:
		if err := r.conn.Select(&orders, "SELECT * FROM orders where session_id = $1 ORDER BY id DESC;", sessionID); err != nil {
			return nil, err
		}
	case Features:
		if err := r.conn.Select(&orders, "SELECT * FROM features_orders where session_id = $1 ORDER BY id DESC;", sessionID); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *OrderRepository) GetBySessionIDWithSide(sessionID, side string) ([]models.Order, error) {
	var orders []models.Order

	switch r.table {
	case Spot:
		if err := r.conn.Select(&orders, "SELECT * FROM orders where session_id = $1 AND side = $2 ORDER BY id DESC;", sessionID, side); err != nil {
			return nil, err
		}
	case Features:
		if err := r.conn.Select(&orders, "SELECT * FROM features_orders where session_id = $1 AND side = $2 ORDER BY id DESC;", sessionID, side); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *OrderRepository) GetLastWithInterval(symbol string, sTime, eTime time.Time) ([]models.Order, error) {
	var orders []models.Order

	switch r.table {
	case Spot:
		if err := r.conn.Select(&orders, "SELECT * FROM orders where created_at > $1 AND created_at < $2 AND symbol = $3;", sTime.UTC(), eTime.UTC(), symbol); err != nil {
			return nil, err
		}
	case Features:
		if err := r.conn.Select(&orders, "SELECT * FROM features_orders where created_at > $1 AND created_at < $2 AND symbol = $3;", sTime.UTC(), eTime.UTC(), symbol); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *OrderRepository) SetActualPrice(id int, price float64) error {
	switch r.table {
	case Spot:
		if _, err := r.conn.Exec("UPDATE orders SET price = $1 where id = $2;", price, id); err != nil {
			return err
		}
	case Features:
		if _, err := r.conn.Exec("UPDATE features_orders SET price = $1 where id = $2;", price, id); err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderRepository) SetTry(id, try int) error {
	switch r.table {
	case Spot:
		if _, err := r.conn.Exec("UPDATE orders SET try = $1 where id = $2;", try, id); err != nil {
			return err
		}
	case Features:
		if _, err := r.conn.Exec("UPDATE features_orders SET try = $1 where id = $2;", try, id); err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderRepository) SetStatus(id string, status string) error {
	switch r.table {
	case Spot:
		if _, err := r.conn.Exec("UPDATE orders SET status = $1 where id = $2;", status, id); err != nil {
			return err
		}

	case Features:
		if _, err := r.conn.Exec("UPDATE features_orders SET status = $1 where id = $2;", status, id); err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderRepository) SetOrderID(id string, orderID int64) error {
	switch r.table {
	case Spot:
		if _, err := r.conn.Exec("UPDATE orders SET order_id = $1 where id = $2;", orderID, id); err != nil {
			return err
		}

	case Features:
		if _, err := r.conn.Exec("UPDATE features_orders SET order_id = $1 where id = $2;", orderID, id); err != nil {
			return err
		}
	}

	return nil
}
