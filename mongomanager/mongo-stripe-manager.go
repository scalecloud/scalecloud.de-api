package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

func (mongoConnection *MongoConnection) CreateUser(ctx context.Context, user User) error {
	return mongoConnection.createDocument(ctx, databaseStripe, collectionUsers, user)
}

func (mongoConnection *MongoConnection) UpdateUser(ctx context.Context, user User) error {
	if user.UID == "" {
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	return mongoConnection.updateDocument(ctx, databaseStripe, collectionUsers, user, filter)
}

func (mongoConnection *MongoConnection) DeleteUser(ctx context.Context, customerID string) error {
	filter := bson.M{"customerID": customerID}
	return mongoConnection.deleteDocument(ctx, databaseStripe, collectionUsers, filter)
}

func (mongoConnection *MongoConnection) GetUser(ctx context.Context, userFilter User) (User, error) {
	if userFilter.UID == "" {
		return User{}, errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": userFilter.UID}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseStripe, collectionUsers, filter)
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
