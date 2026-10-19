package mongomanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func (mongoConnection *MongoConnection) ensureStripeUserUIDIndex(ctx context.Context) error {
	collection, err := mongoConnection.getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		return err
	}
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "uid", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("UniqueStripeUserUID"),
	})
	if err != nil {
		mongoConnection.Log.Error("Error creating unique Stripe user UID index", zap.Error(err))
		return err
	}
	mongoConnection.stripeUserUIDIndexReady.Store(true)
	return nil
}

func (mongoConnection *MongoConnection) CreateUser(ctx context.Context, user User) error {
	return mongoConnection.createDocument(ctx, databaseStripe, collectionUsers, user)
}

func (mongoConnection *MongoConnection) EnsureUser(ctx context.Context, uid string) error {
	if uid == "" {
		return errors.New("user.UID is empty")
	}
	if !mongoConnection.stripeUserUIDIndexReady.Load() {
		if err := mongoConnection.ensureStripeUserUIDIndex(ctx); err != nil {
			return fmt.Errorf("ensure unique Stripe user UID index: %w", err)
		}
	}
	if err := mongoConnection.CreateUser(ctx, User{UID: uid}); err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("create Stripe user placeholder: %w", err)
	}
	return nil
}

func (mongoConnection *MongoConnection) ClaimUserProvisioning(ctx context.Context, uid, token string, now, leaseExpiresAt time.Time) (bool, error) {
	if uid == "" || token == "" {
		return false, errors.New("user UID and provisioning token are required")
	}
	if !leaseExpiresAt.After(now) {
		return false, errors.New("provisioning lease expiration must be in the future")
	}
	collection, err := mongoConnection.getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		return false, err
	}
	filter := bson.M{
		"uid": uid,
		"$and": bson.A{
			bson.M{"$or": bson.A{
				bson.M{"customerID": bson.M{"$exists": false}},
				bson.M{"customerID": ""},
			}},
			bson.M{"$or": bson.A{
				bson.M{"provisioningLeaseExpiresAt": bson.M{"$exists": false}},
				bson.M{"provisioningLeaseExpiresAt": bson.M{"$lte": now}},
			}},
		},
	}
	update := bson.M{"$set": bson.M{
		"provisioningToken":          token,
		"provisioningLeaseExpiresAt": leaseExpiresAt,
	}}
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("claim Stripe customer provisioning lease: %w", err)
	}
	return result.MatchedCount == 1, nil
}

func (mongoConnection *MongoConnection) SetUserCustomerID(ctx context.Context, uid, token, customerID string) error {
	if uid == "" || token == "" || customerID == "" {
		return errors.New("user UID, provisioning token, and customer ID are required")
	}
	collection, err := mongoConnection.getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		return err
	}
	filter := bson.M{
		"uid":               uid,
		"provisioningToken": token,
		"$or": bson.A{
			bson.M{"customerID": bson.M{"$exists": false}},
			bson.M{"customerID": ""},
		},
	}
	update := bson.M{
		"$set":   bson.M{"customerID": customerID},
		"$unset": bson.M{"provisioningToken": "", "provisioningLeaseExpiresAt": ""},
	}
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("save Stripe customer mapping: %w", err)
	}
	if result.MatchedCount != 1 {
		return errors.New("customer provisioning lease is no longer owned")
	}
	return nil
}

func (mongoConnection *MongoConnection) ReleaseUserProvisioning(ctx context.Context, uid, token string) error {
	collection, err := mongoConnection.getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		return err
	}
	_, err = collection.UpdateOne(ctx, bson.M{"uid": uid, "provisioningToken": token}, bson.M{
		"$unset": bson.M{"provisioningToken": "", "provisioningLeaseExpiresAt": ""},
	})
	if err != nil {
		return fmt.Errorf("release Stripe customer provisioning lease: %w", err)
	}
	return nil
}

func (mongoConnection *MongoConnection) UpdateUser(ctx context.Context, user User) error {
	if user.UID == "" {
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	update := bson.M{"$set": user}
	return mongoConnection.updateDocument(ctx, databaseStripe, collectionUsers, filter, update)
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
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, mongo.ErrNoDocuments
		}
		mongoConnection.Log.Error("Error finding user", zap.Error(err))
		return User{}, fmt.Errorf("error finding user: %w", err)
	}
	var user User
	decodeErr := singleResult.Decode(&user)
	if decodeErr != nil {
		return User{}, decodeErr
	}
	return user, nil
}
