package mongomanager

import (
	"bufio"
	"context"
	"errors"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getConnectionString() (string, error) {
	file, err := os.Open(connectionString)
	if err != nil {
		return "", errors.New("connectionString does not exist")
	}
	defer file.Close()
	var result string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		result = scanner.Text()
	}
	return result, nil
}

func getClient(ctx context.Context) (*mongo.Client, error) {
	uri, err := getConnectionString()
	if err != nil {
		return nil, err
	}
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(uri + x509).
		SetServerAPIOptions(serverAPIOptions)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getCollection(context context.Context, databaseName, collectionName string) (*mongo.Client, *mongo.Collection, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, nil, err
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
		return err
	}
	defer disconnect(ctx, client)
	_, err = collection.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}

func updateDocument(ctx context.Context, databaseName, collectionName string, filter, update interface{}) error {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return err
	}
	defer disconnect(ctx, client)
	if filter == nil {
		return errors.New("filter is nil")
	}
	if update == nil {
		return errors.New("update is nil")
	}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func deleteDocument(ctx context.Context, databaseName, collectionName string, filter interface{}) error {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return err
	}
	defer disconnect(ctx, client)
	if filter == nil {
		return errors.New("filter is nil")
	}
	_, err = collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func findDocument(ctx context.Context, databaseName, collectionName string, filter interface{}) (*mongo.SingleResult, error) {
	client, collection, err := getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return nil, err
	}
	defer disconnect(ctx, client)
	if filter == nil {
		return nil, errors.New("filter is nil")
	}
	singleResult := collection.FindOne(ctx, filter)
	if singleResult.Err() != nil {
		return nil, singleResult.Err()
	}
	return singleResult, nil
}

func disconnect(ctx context.Context, client *mongo.Client) error {
	return client.Disconnect(ctx)
}
