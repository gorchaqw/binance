package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func (a *App) InitDB(dbConfig *DB) error {
	fmt.Println(dbConfig.DSN())

	db, err := sqlx.Connect("postgres", dbConfig.DSN())
	if err != nil {
		log.Fatalln(err)
	}
	a.DB = db

	return nil
}
