package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const databaseStripe = "stripe"
const collectionUsers = "users"

func initMongoStripe(log *zap.Logger) error {
	ctx := context.Background()
	log.Info("Check for database: " + databaseStripe + " and collection: " + collectionUsers)
	client, users, err := getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		log.Error("Error getting collection", zap.Error(err))
		return errors.New("Error getting collection")
	} else {
		defer disconnect(ctx, client)
	}
	return checkConnectability(ctx, users, log)
}

func checkConnectability(ctx context.Context, users *mongo.Collection, log *zap.Logger) error {
	usersCount, err := users.CountDocuments(ctx, bson.D{})
	if err != nil {
		log.Error("Error counting documents", zap.Error(err))
		return errors.New("Error counting documents")
	} else if usersCount == 0 {
		log.Warn("Users collection is empty.")
	} else {
		log.Info("Users count: ", zap.Any("count", usersCount))
	}
	return nil
}

func CreateUser(ctx context.Context, user User) error {
	return createDocument(ctx, databaseStripe, collectionUsers, user)
}

func UpdateUser(ctx context.Context, user User) error {
	if user.UID == "" {
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	return updateDocument(ctx, databaseStripe, collectionUsers, user, filter)
}

func DeleteUser(ctx context.Context, user User) error {
	if user.UID == "" {
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	return deleteDocument(ctx, databaseStripe, collectionUsers, filter)
}

func GetUser(ctx context.Context, userFilter User) (User, error) {
	if userFilter.UID == "" {
		return User{}, errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": userFilter.UID}
	singleResult, err := findDocument(ctx, databaseStripe, collectionUsers, filter)
	if err != nil {
		return User{}, err
	}
	var user User
	decodeErr := singleResult.Decode(&user)
	if decodeErr != nil {
		return User{}, decodeErr
	}
	return user, nil
}
