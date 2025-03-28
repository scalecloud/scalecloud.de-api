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

func (mongoConnection *MongoConnection) CountSeats(ctx context.Context, subscriptionID string) (int64, error) {
	return mongoConnection.countDocuments(ctx, databaseSubscription, collectionSeats, bson.M{"subscriptionID": subscriptionID})
}

func (mongoConnection *MongoConnection) GetAllSeats(ctx context.Context, subscriptionID string) ([]Seat, error) {
	return mongoConnection.GetSeats(ctx, subscriptionID, 0, 0)
}

func (mongoConnection *MongoConnection) GetSeats(ctx context.Context, subscriptionID string, pageIndex int, pageSize int) ([]Seat, error) {
	if subscriptionID == "" {
		return []Seat{}, errors.New("subscription ID is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
	}
	var seats []Seat
	opts := options.Find()
	opts.SetLimit(int64(pageSize))
	opts.SetSkip(int64(pageIndex * pageSize))
	opts.SetSort(bson.D{{Key: "email", Value: 1}})
	err := mongoConnection.findDocuments(ctx, databaseSubscription, collectionSeats, filter, &seats, opts)
	if err != nil {
		return []Seat{}, err
	}
	return seats, nil
}

func (mongoConnection *MongoConnection) GetSeat(ctx context.Context, subscriptionID, uid string) (Seat, error) {
	if subscriptionID == "" {
		return Seat{}, errors.New("subscription ID is empty")
	}
	if uid == "" {
		return Seat{}, errors.New("uid is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
		"uid":            uid,
	}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, filter)
	if err != nil {
		mongoConnection.Log.Error("Error finding seat", zap.Error(err))
		return Seat{}, errors.New("error finding seat")
	}
	var seat Seat
	decodeErr := singleResult.Decode(&seat)
	if decodeErr != nil {
		return Seat{}, decodeErr
	}
	return seat, nil
}

func (mongoConnection *MongoConnection) GetOwnerSeat(ctx context.Context, subscriptionID string) (Seat, error) {
	if subscriptionID == "" {
		return Seat{}, errors.New("subscription ID is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
		"roles": bson.M{
			"$in": []string{string(RoleOwner)},
		},
	}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, filter)
	if err != nil {
		mongoConnection.Log.Error("Error finding seat", zap.Error(err))
		return Seat{}, errors.New("error finding seat")
	}
	var seat Seat
	decodeErr := singleResult.Decode(&seat)
	if decodeErr != nil {
		return Seat{}, decodeErr
	}
	return seat, nil
}

func (mongoConnection *MongoConnection) UpdateSeat(ctx context.Context, seat Seat) error {
	filter := bson.M{
		"subscriptionID": seat.SubscriptionID,
		"uid":            seat.UID,
	}
	update := bson.M{
		"$set": seat,
	}
	return mongoConnection.updateDocument(ctx, databaseSubscription, collectionSeats, filter, update)
}

func (mongoConnection *MongoConnection) DeleteSeat(ctx context.Context, seat Seat) error {
	filter := bson.M{
		"subscriptionID": seat.SubscriptionID,
		"uid":            seat.UID,
	}
	return mongoConnection.deleteDocument(ctx, databaseSubscription, collectionSeats, filter)
}
