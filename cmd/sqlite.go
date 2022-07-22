package main

import (
	"github.com/jmoiron/sqlx"
	"log"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func (a *App) InitDB(dbFileName string) error {
	db, err := sqlx.Connect("postgres", "user=hello password=hello dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	a.DB = db

	return nil
}
