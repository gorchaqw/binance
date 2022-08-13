package models

import "time"

type Order struct {
	ID        int       `db:"id" json:"id,omitempty"`
	OrderId   int64     `db:"order_id" json:"order_id,omitempty"`
	Symbol    string    `db:"symbol" json:"symbol,omitempty"`
	Side      string    `db:"side" json:"side,omitempty"`
	Quantity  string    `db:"quantity" json:"quantity,omitempty"`
	Price     float64   `db:"price" json:"price,omitempty"`
	StopPrice float64   `db:"stop_price" json:"stop_price,omitempty"`
	Status    string    `db:"status" json:"status,omitempty"`
	Try       int       `db:"try" json:"try,omitempty"`
	Type      string    `db:"type" json:"type,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
