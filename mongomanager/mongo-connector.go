package mongomanager

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func sanitizeURI(rawURI string) (string, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse connection string: %w", err)
	}
	query := parsed.Query()
	query.Del("tlsCertificateKeyFile")
	query.Del("tlsCertificateFile")
	query.Del("tlsPrivateKeyFile")
	query.Del("tlsCertificateKeyFilePassword")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func getConnectionString() (string, error) {
	uri := strings.TrimSpace(os.Getenv("MONGODB_URI"))
	if uri == "" {
		return "", errors.New("MONGODB_URI is empty")
	}
	return sanitizeURI(uri)
}

func loadClientCertificate() (tls.Certificate, error) {
	encodedCertificate := strings.TrimSpace(os.Getenv("MONGODB_TLS_CERT_BASE64"))
	if encodedCertificate == "" {
		return tls.Certificate{}, errors.New("MONGODB_TLS_CERT_BASE64 is empty")
	}
	pemBytes, err := base64.StdEncoding.DecodeString(encodedCertificate)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("MONGODB_TLS_CERT_BASE64 is not valid standard Base64: %w", err)
	}
	cert, err := tls.X509KeyPair(pemBytes, pemBytes)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse MongoDB TLS certificate and private key: %w", err)
	}
	return cert, nil
}

func getClient(ctx context.Context) (*mongo.Client, error) {
	uri, err := getConnectionString()
	if err != nil {
		return nil, err
	}
	cert, err := loadClientCertificate()
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetTLSConfig(tlsConfig).
		SetServerAPIOptions(serverAPIOptions)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
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
		return fmt.Errorf("error inserting document: %w", err)
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
		resultErr := singleResult.Err()
		if errors.Is(resultErr, mongo.ErrNoDocuments) {
			return nil, nil
		}
		mongoConnection.Log.Error("Error finding document", zap.Error(resultErr))
		return nil, resultErr
	}
	return singleResult, nil
}

func (mongoConnection *MongoConnection) findDocuments(ctx context.Context, databaseName, collectionName string, filter interface{}, results interface{}, opts options.Lister[options.FindOptions]) (resultErr error) {
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
	defer func() {
		if err := cursor.Close(ctx); resultErr == nil && err != nil {
			resultErr = err
		}
	}()
	// Decode the results
	if err := cursor.All(ctx, results); err != nil {
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
