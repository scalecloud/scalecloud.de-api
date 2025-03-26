package mongomanager

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

func (mongoConnection *MongoConnection) getCollection(context context.Context, databaseName, collectionName string) (*mongo.Collection, error) {
	client := mongoConnection.Client
	collection := client.Database(databaseName).Collection(collectionName)
	if collection == nil {
		return nil, errors.New("collection is nil")
	}
	return collection, nil
}

func (mongoConnection *MongoConnection) createDocument(ctx context.Context, databaseName, collectionName string, document interface{}) error {
	collection, err := mongoConnection.getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return err
	}
	_, err = collection.InsertOne(ctx, document)
	if err != nil {
		mongoConnection.Log.Error("Error inserting document", zap.Error(err))
		return errors.New("error inserting document")
	}
	return nil
}

func (mongoConnection *MongoConnection) updateDocument(ctx context.Context, databaseName, collectionName string, filter, update interface{}) error {
	collection, err := mongoConnection.getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return err
	}
	if filter == nil {
		return errors.New("filter is nil")
	}
	if update == nil {
		return errors.New("update is nil")
	}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		mongoConnection.Log.Error("Error updating document", zap.Error(err))
		return errors.New("error updating document")
	}
	return nil
}

func (mongoConnection *MongoConnection) deleteDocument(ctx context.Context, databaseName, collectionName string, filter interface{}) error {
	collection, err := mongoConnection.getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return err
	}
	if filter == nil {
		return errors.New("filter is nil")
	}
	_, err = collection.DeleteOne(ctx, filter)
	if err != nil {
		mongoConnection.Log.Error("Error deleting document", zap.Error(err))
		return errors.New("error deleting document")
	}
	return nil
}

func (mongoConnection *MongoConnection) findOneDocument(ctx context.Context, databaseName, collectionName string, filter interface{}) (*mongo.SingleResult, error) {
	collection, err := mongoConnection.getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, errors.New("filter is nil")
	}
	singleResult := collection.FindOne(ctx, filter)
	if singleResult.Err() != nil {
		mongoConnection.Log.Error("Error finding document", zap.Error(singleResult.Err()))
		return nil, singleResult.Err()
	}
	return singleResult, nil
}

func (mongoConnection *MongoConnection) findDocuments(ctx context.Context, databaseName, collectionName string, filter interface{}, results interface{}, opts *options.FindOptions) error {
	collection, err := mongoConnection.getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return err
	}
	if filter == nil {
		return errors.New("filter is nil")
	}
	// Call Find with sorting and pagination
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		mongoConnection.Log.Error("Error finding documents", zap.Error(err))
		return errors.New("error finding documents")
	}
	defer cursor.Close(ctx)
	// Decode the results
	if err = cursor.All(ctx, results); err != nil {
		return err
	}
	return nil
}

func (mongoConnection *MongoConnection) countDocuments(ctx context.Context, databaseName, collectionName string, filter interface{}) (int64, error) {
	collection, err := mongoConnection.getCollection(ctx, databaseName, collectionName)
	if err != nil {
		return 0, err
	}
	if filter == nil {
		return 0, errors.New("filter is nil")
	}
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		mongoConnection.Log.Error("Error counting documents", zap.Error(err))
		return 0, errors.New("error counting documents")
	}
	return count, nil
}
