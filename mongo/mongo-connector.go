package mongo

import (
	"bufio"
	"context"
	"errors"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getConnectionString() string {
	file, err := os.Open(connectionString)
	if err != nil {
		logger.Error("Error opening file", zap.String("file", connectionString), zap.Error(err))
		os.Exit(1)
	}
	defer file.Close()
	var result string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		result = scanner.Text()
	}
	return result
}

func getClient(ctx context.Context) *mongo.Client {
	uri := getConnectionString() + x509
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(serverAPIOptions)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Error connecting to MongoDB", zap.Error(err))
	}
	return client
}

func getCollection(context context.Context, databaseName, collectionName string) (*mongo.Client, *mongo.Collection, error) {
	client := getClient(context)
	if client == nil {
		return nil, nil, errors.New("client is nil")
	}
	collection := client.Database(databaseName).Collection(collectionName)
	if collection == nil {
		return nil, nil, errors.New("collection is nil")
	}
	return client, collection, nil
}

func createDocument(ctx context.Context, databaseName, collectionName string, document interface{}) error {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		logger.Error("Error getting collection", zap.Error(err))
		return err
	}
	defer client.Disconnect(ctx)
	result, err := collection.InsertOne(ctx, document)
	if err != nil {
		logger.Error("Error creating document", zap.Error(err))
		return err
	} else {
		logger.Info("Created document", zap.Any("result", result))
		return nil
	}
}

func updateDocument(ctx context.Context, databaseName, collectionName string, filter, update interface{}) error {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		logger.Error("Error getting collection", zap.Error(err))
		return err
	}
	defer client.Disconnect(ctx)
	if filter == nil {
		logger.Error("filter is nil")
		return errors.New("filter is nil")
	}
	if update == nil {
		logger.Error("update is nil")
		return errors.New("update is nil")
	}
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error("Error updating document", zap.Error(err))
		return err
	} else {
		logger.Info("Updated document", zap.Any("result", result))
		return nil
	}
}

func deleteDocument(ctx context.Context, databaseName, collectionName string, filter interface{}) error {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		logger.Error("Error getting collection", zap.Error(err))
		return err
	}
	defer client.Disconnect(ctx)
	if filter == nil {
		logger.Error("filter is nil")
		return errors.New("filter is nil")
	}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		logger.Error("Error deleting document", zap.Error(err))
		return err
	} else {
		logger.Info("Deleted document", zap.Any("result", result))
		return nil
	}
}

func findDocument(ctx context.Context, databaseName, collectionName string, filter interface{}) (interface{}, error) {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		logger.Error("Error getting collection", zap.Error(err))
		return nil, err
	}
	defer client.Disconnect(ctx)
	if filter == nil {
		logger.Error("filter is nil")
		return nil, errors.New("filter is nil")
	}
	result := collection.FindOne(ctx, filter)
	if result.Err() != nil {
		logger.Error("Error finding document", zap.Error(result.Err()))
		return nil, result.Err()
	} else {
		logger.Info("Found document")
		return result.Decode(filter), nil
	}
}
