package main

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (a *App) initMongo() error {
	credential := options.Credential{
		AuthSource: a.Config.Mongo.DBName,
		Username:   a.Config.Mongo.User,
		Password:   a.Config.Mongo.Password,
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(a.Config.Mongo.DSN()).SetAuth(credential))
	if err != nil {
		return err
	}

	a.Mongo = client

	return nil
}
