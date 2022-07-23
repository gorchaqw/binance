package models

import "time"

type Order struct {
	ID        int       `db:"id"`
	OrderId   int64     `db:"order_id"`
	Symbol    string    `db:"symbol"`
	Side      string    `db:"side"`
	Quantity  string    `db:"quantity"`
	Price     float64   `db:"price"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}
