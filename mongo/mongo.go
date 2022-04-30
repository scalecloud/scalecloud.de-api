package mongo

import (
	"context"
	"time"

	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var logger, _ = zap.NewProduction()

func InitMongo() {
	logger.Info("Init Mongo")
}

func startDBCon() {
	connectionString := "mongodb://localhost:27017"
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))
	if err != nil {
		logger.Error("Error creating client", zap.Error(err))
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		logger.Error("Error connecting to client", zap.Error(err))
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		logger.Error("Error pinging client", zap.Error(err))
	}
	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		logger.Error("Error listing databases", zap.Error(err))
	}
	logger.Info("Databases", zap.Any("databases", databases))
}
