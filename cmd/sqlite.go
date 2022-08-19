package main

import (
	"github.com/jmoiron/sqlx"
	"log"

	_ "github.com/lib/pq"
)

func (a *App) InitDB() error {
	db, err := sqlx.Connect("postgres", "host=binance-postgres user=binance password=binance dbname=binance sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	a.DB = db

	return nil
}
