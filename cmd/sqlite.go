package main

import (
	"github.com/jmoiron/sqlx"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func (a *App) InitDB() error {
	db, err := sqlx.Connect("sqlite3", "./store.db")
	if err != nil {
		log.Fatalln(err)
	}
	a.DB = db

	return nil
}
