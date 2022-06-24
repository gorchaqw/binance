package models

import "time"

type Price struct {
	ID        int       `db:"id"`
	Symbol    string    `db:"symbol"`
	Price     float64   `db:"price"`
	CreatedAt time.Time `db:"created_at"`
}
