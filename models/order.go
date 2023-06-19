package models

import "time"

type Order struct {
	ID           string    `db:"id" json:"id,omitempty"`
	OrderID      int64     `db:"order_id" json:"order_id,omitempty"`
	SessionID    string    `db:"session_id" json:"session_id,omitempty"`
	Symbol       string    `db:"symbol" json:"symbol,omitempty"`
	Side         string    `db:"side" json:"side,omitempty"`
	PositionSide string    `db:"position_side" json:"position_side,omitempty"`
	Quantity     float64   `db:"quantity" json:"quantity,omitempty"`
	Price        float64   `db:"price" json:"price,omitempty"`
	ActualPrice  float64   `db:"actual_price" json:"actual_price,omitempty"`
	StopPrice    float64   `db:"stop_price" json:"stop_price,omitempty"`
	Status       string    `db:"status" json:"status,omitempty"`
	Try          int       `db:"try" json:"try,omitempty"`
	Type         string    `db:"type" json:"type,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
