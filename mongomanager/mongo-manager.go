package mongomanager

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type MongoConnection struct {
	Client                  *mongo.Client
	Log                     *zap.Logger
	indexesMu               sync.Mutex
	indexesReady            bool
	stripeUserUIDIndexReady atomic.Bool
}

func InitMongoConnection(ctx context.Context, log *zap.Logger) (*MongoConnection, error) {
	log.Info("Init MongoManager")
	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}
	mongoManager := &MongoConnection{
		Log:    log.Named("mongomanager"),
		Client: client,
	}
	return mongoManager, nil
}

func CheckMongoConfig(log *zap.Logger) error {
	log.Info("Checking MongoDB configuration")
	if _, err := getConnectionString(); err != nil {
		return err
	}
	if _, err := loadClientCertificate(); err != nil {
		return err
	}
	log.Info("MongoDB configuration is present")
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

func (mongoConnection *MongoConnection) CheckHealth(ctx context.Context) error {
	if err := mongoConnection.CheckDatabaseAndCollectionExists(ctx); err != nil {
		return err
	}

	mongoConnection.indexesMu.Lock()
	defer mongoConnection.indexesMu.Unlock()
	if !mongoConnection.indexesReady {
		if err := mongoConnection.EnsureIndexes(ctx); err != nil {
			return err
		}
		mongoConnection.indexesReady = true
	}
	return nil
}

func (mongoConnection *MongoConnection) CheckDatabaseAndCollectionExists(ctx context.Context) error {
	for dbName, collections := range databases {
		err := mongoConnection.ensureCollectionsExist(ctx, dbName, collections)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mongoConnection *MongoConnection) ensureCollectionsExist(ctx context.Context, dbName string, collectionNames []string) error {
	database := mongoConnection.Client.Database(dbName)
	existingCollections, err := database.ListCollectionNames(ctx, bson.M{})
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
