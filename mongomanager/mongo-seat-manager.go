package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mongoConnection *MongoConnection) ensureSeatIndex() error {
	indexModel := mongo.IndexModel{
		Keys: bson.M{
			"subscriptionID": 1,
			"email":          1,
		},
		Options: options.Index().SetUnique(true),
	}

	collection, err := mongoConnection.getCollection(context.Background(), databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	// Retrieve all indexes
	cursor, err := collection.Indexes().List(context.Background())
	if err != nil {
		return err
	}

	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		return err
	}

	// Check if the index already exists
	for _, result := range results {
		if result["key"] == indexModel.Keys && result["unique"] == true {
			// The index already exists, no need to create it
			mongoConnection.Log.Info("Index for seats already exists")
			return nil
		}
	}

	// The index doesn't exist or is not the same, create it
	_, err = collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		return err
	}
	mongoConnection.Log.Info("Index for seats created")
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
