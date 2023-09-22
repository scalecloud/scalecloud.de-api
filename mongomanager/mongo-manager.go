package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const x509 = "keys/mongodb-atlas.pem"
const connectionString = "keys/mongodb-atlas-connection-string.txt"

type MongoConnection struct {
	Client *mongo.Client
	Log    *zap.Logger
}

func InitMongoConnection(ctx context.Context, log *zap.Logger) (*MongoConnection, error) {
	log.Info("Init MongoManager")
	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}
	mongoManager := &MongoConnection{
		Log:    log.Named("firebasemanager"),
		Client: client,
	}
	return mongoManager, nil
}

func InitMongo(log *zap.Logger) error {
	log.Info("Init Mongo")
	if fileExists(connectionString) {
		log.Debug("connectionString exists. ", zap.String("file", connectionString))
	} else {
		return errors.New("connectionString does not exist")
	}
	if fileExists(x509) {
		log.Debug("x509 exists. ", zap.String("file", x509))
	} else {
		return errors.New("x509 does not exist")
	}
	err := initMongoStripe(log)
	if err != nil {
		return err
	}
	return nil
}
