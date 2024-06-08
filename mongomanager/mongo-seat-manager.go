package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (mongoConnection *MongoConnection) ensureSeatIndex() error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "subscriptionID", Value: 1},
			{Key: "email", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("UniqueSubscriptionEmail"),
	}
	collection, err := mongoConnection.getCollection(context.Background(), databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	name, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		mongoConnection.Log.Error("Error creating index for seats", zap.String("error", err.Error()))
		return err
	}

	mongoConnection.Log.Info("Required index for collection " + collection.Name() + " is present. Index: " + name)
	return nil
}

func (mongoConnection *MongoConnection) CreateSeat(ctx context.Context, seat Seat) error {
	err := ValidateStruct(seat)
	if err != nil {
		return err
	}
	return mongoConnection.createDocument(ctx, databaseSubscription, collectionSeats, seat)
}

func (mongoConnection *MongoConnection) GetSeats(ctx context.Context, subscriptionID string) ([]Seat, error) {
	if subscriptionID == "" {
		return []Seat{}, errors.New("subscription ID is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
	}
	var seats []Seat
	err := mongoConnection.findDocuments(ctx, databaseSubscription, collectionSeats, filter, &seats)
	if err != nil {
		return []Seat{}, err
	}
	return seats, nil
}

func (mongoConnection *MongoConnection) GetSeat(ctx context.Context, subscriptionID, email string) (Seat, error) {
	if subscriptionID == "" {
		return Seat{}, errors.New("subscription ID is empty")
	}
	if email == "" {
		return Seat{}, errors.New("email is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
		"email":          email,
	}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, filter)
	if err != nil {
		return Seat{}, err
	}
	var seat Seat
	decodeErr := singleResult.Decode(&seat)
	if decodeErr != nil {
		return Seat{}, decodeErr
	}
	return seat, nil
}

func (mongoConnection *MongoConnection) DeleteSeat(ctx context.Context, seat Seat) error {
	if seat.SubscriptionID == "" {
		return errors.New("subscription ID is empty")
	}
	if seat.EMail == "" {
		return errors.New("email is empty")
	}
	filter := bson.M{"subscriptionID": seat.SubscriptionID}
	return mongoConnection.deleteDocument(ctx, databaseStripe, collectionUsers, filter)
}
