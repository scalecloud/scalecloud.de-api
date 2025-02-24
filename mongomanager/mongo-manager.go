package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
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

func CheckMongoConnectionFiles(log *zap.Logger) error {
	log.Info("Init Mongo")
	if fileExists(connectionString) {
		log.Debug("connectionString exists. ", zap.String("file", connectionString))
	} else {
		return errors.New("required file does not exist: " + connectionString)
	}
	if fileExists(x509) {
		log.Debug("x509 exists. ", zap.String("file", x509))
	} else {
		return errors.New("x509 does not exist")
	}
	log.Info("Required secrets for MongoDB are present.")
	return nil
}

func (mongoConnection *MongoConnection) CheckMongoConnectability(ctx context.Context) error {
	users, err := mongoConnection.getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		return err
	}
	usersCount, err := users.CountDocuments(ctx, bson.D{})
	if err != nil {
		return errors.New("Error counting documents: " + err.Error())
	} else if usersCount == 0 {
		mongoConnection.Log.Warn("Users collection is empty.")
	} else {
		mongoConnection.Log.Info("Users count: ", zap.Any("count", usersCount))
	}
	return nil
}

func (mongoConnection *MongoConnection) CheckDatabaseAndCollectionExists(ctx context.Context) error {
	for dbName, collections := range databases {
		err := mongoConnection.ensureCollectionsExist(dbName, collections)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mongoConnection *MongoConnection) ensureCollectionsExist(dbName string, collectionNames []string) error {
	database := mongoConnection.Client.Database(dbName)
	existingCollections, err := database.ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return errors.New("Failed to get collection names: " + err.Error())
	}

	existingCollectionsMap := make(map[string]bool)
	for _, name := range existingCollections {
		existingCollectionsMap[name] = true
	}

	for _, name := range collectionNames {
		if !existingCollectionsMap[name] {
			return errors.New("Collection " + name + " does not exist in database " + dbName)
		}
	}

	return nil
}
