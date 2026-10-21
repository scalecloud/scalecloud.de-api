package mongodbmanager

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

func (mongoDBConnection *MongoDBConnection) ensureStripeUserUIDIndex(ctx context.Context) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseStripe, collectionUsers)
	if err != nil {
		return err
	}
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "uid", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("UniqueStripeUserUID"),
	})
	if err != nil {
		mongoDBConnection.Log.Error("Error creating unique Stripe user UID index", zap.Error(err))
		return err
	}
	mongoDBConnection.stripeUserUIDIndexReady.Store(true)
	return nil
}

func (mongoDBConnection *MongoDBConnection) CreateUser(ctx context.Context, user User) error {
	return mongoDBConnection.createDocument(ctx, databaseStripe, collectionUsers, user)
}

func (mongoDBConnection *MongoDBConnection) EnsureUser(ctx context.Context, uid string) error {
	if uid == "" {
		return errors.New("user.UID is empty")
	}
	if !mongoDBConnection.stripeUserUIDIndexReady.Load() {
		if err := mongoDBConnection.ensureStripeUserUIDIndex(ctx); err != nil {
			return fmt.Errorf("ensure unique Stripe user UID index: %w", err)
		}
	}
	if err := mongoDBConnection.CreateUser(ctx, User{UID: uid}); err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("create Stripe user placeholder: %w", err)
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) ClaimUserProvisioning(ctx context.Context, uid, token string, now, leaseExpiresAt time.Time) (bool, error) {
	if uid == "" || token == "" {
		return false, errors.New("user UID and provisioning token are required")
	}
	if !leaseExpiresAt.After(now) {
		return false, errors.New("provisioning lease expiration must be in the future")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseStripe, collectionUsers)
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

func (mongoDBConnection *MongoDBConnection) SetUserCustomerID(ctx context.Context, uid, token, customerID string) error {
	if uid == "" || token == "" || customerID == "" {
		return errors.New("user UID, provisioning token, and customer ID are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseStripe, collectionUsers)
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

func (mongoDBConnection *MongoDBConnection) ReleaseUserProvisioning(ctx context.Context, uid, token string) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseStripe, collectionUsers)
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

func (mongoDBConnection *MongoDBConnection) UpdateUser(ctx context.Context, user User) error {
	if user.UID == "" {
		return errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": user.UID}
	update := bson.M{"$set": user}
	return mongoDBConnection.updateDocument(ctx, databaseStripe, collectionUsers, filter, update)
}

func (mongoDBConnection *MongoDBConnection) DeleteUser(ctx context.Context, customerID string) error {
	filter := bson.M{"customerID": customerID}
	return mongoDBConnection.deleteDocument(ctx, databaseStripe, collectionUsers, filter)
}

func (mongoDBConnection *MongoDBConnection) GetUser(ctx context.Context, userFilter User) (User, error) {
	if userFilter.UID == "" {
		return User{}, errors.New("user.UID is empty")
	}
	filter := bson.M{"uid": userFilter.UID}
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseStripe, collectionUsers, filter)
	if err != nil {
		mongoDBConnection.Log.Error("Error finding user", zap.Error(err))
		return User{}, fmt.Errorf("error finding user: %w", err)
	}
	if singleResult == nil {
		return User{}, nil
	}
	var user User
	decodeErr := singleResult.Decode(&user)
	if decodeErr != nil {
		return User{}, decodeErr
	}
	return user, nil
}
