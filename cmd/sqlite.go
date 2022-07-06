package main

import (
	"github.com/jmoiron/sqlx"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func (a *App) InitDB(dbFileName string) error {
	db, err := sqlx.Connect("sqlite3", dbFileName)
	if err != nil {
		log.Fatalln(err)
	}
	a.DB = db

	return nil
}
