package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func (a *App) InitDB(dbConfig *DB) error {
	db, err := sqlx.Connect("postgres", dbConfig.DSN())
	if err != nil {
		log.Fatalln(err)
	}
	a.DB = db

	return nil
}
