package models

import "time"

type Order struct {
	ID        int       `db:"id"`
	OrderId   int64     `db:"order_id"`
	Symbol    string    `db:"symbol"`
	Side      string    `db:"side"`
	Quantity  string    `db:"quantity"`
	Price     float64   `db:"price"`
	StopPrice float64   `db:"stop_price"`
	Status    string    `db:"status"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
}
