package mongo

import (
	"context"
	"errors"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

const databaseStripe = "stripe"
const collectionUsers = "users"

func initMongoStripe() {
	ctx := context.Background()
	logger.Info("Check for database: " + databaseStripe + " and collection: " + collectionUsers)
	client, users, err := getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		logger.Error("Error getting collection", zap.Error(err))
		os.Exit(1)
	} else {
		defer client.Disconnect(ctx)
	}
	usersCount, err := users.CountDocuments(ctx, bson.D{})
	if err != nil {
		logger.Error("Error counting documents", zap.Error(err))
	} else if usersCount == 0 {
		logger.Warn("Users collection is empty.")
	} else {
		logger.Info("Users count: ", zap.Any("count", usersCount))
	}
}

func createUser(ctx context.Context, user User) error {
	return createDocument(ctx, databaseStripe, collectionUsers, user)
}

func updateUser(ctx context.Context, user User) error {
	if user.UID == "" {
		logger.Error("user.UID is empty")
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	return updateDocument(ctx, databaseStripe, collectionUsers, user, filter)
}

func deleteUser(ctx context.Context, user User) error {
	if user.UID == "" {
		logger.Error("user.UID is empty")
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	return deleteDocument(ctx, databaseStripe, collectionUsers, filter)
}

func getUser(ctx context.Context, user User) (User, error) {
	if user.UID == "" {
		logger.Error("user.UID is empty")
		return User{}, errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	document, err := findDocument(ctx, databaseStripe, collectionUsers, filter)
	if err != nil {
		logger.Error("Error finding document", zap.Error(err))
		return User{}, err
	}
	return document.(User), nil
}
