package mongodbmanager

import (
	"context"
	"errors"
	"math"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (mongoDBConnection *MongoDBConnection) GetOpenCloudStorageUsedBytes(ctx context.Context, subscriptionID string) (int64, error) {
	if subscriptionID == "" {
		return 0, errors.New("subscription ID is required")
	}
	seats, err := mongoDBConnection.GetAllSeats(ctx, subscriptionID)
	if err != nil {
		return 0, err
	}
	var usedBytes int64
	for _, seat := range seats {
		if seat.OpenCloudStorageBytes < 0 || seat.OpenCloudStorageBytes > math.MaxInt64-usedBytes {
			return 0, errors.New("invalid OpenCloud seat allocation total")
		}
		usedBytes += seat.OpenCloudStorageBytes
	}

	resources, err := mongoDBConnection.GetOpenCloudResourcesBySubscription(ctx, subscriptionID)
	if err != nil {
		return 0, err
	}
	for _, resource := range resources {
		if resource.ResourceType != "space" || resource.Scope != "allocation" {
			continue
		}
		if resource.StorageQuotaBytes < 0 || resource.StorageQuotaBytes > math.MaxInt64-usedBytes {
			return 0, errors.New("invalid OpenCloud space allocation total")
		}
		usedBytes += resource.StorageQuotaBytes
	}
	return usedBytes, nil
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudUserStorageBytes(ctx context.Context, openCloudUserID string) (int64, error) {
	if openCloudUserID == "" {
		return 0, errors.New("OpenCloud user ID is required")
	}
	filter := bson.M{
		"openCloudUserId": openCloudUserID,
		"$or": []bson.M{
			{"status": SeatStatusActive},
			{"status": bson.M{"$exists": false}},
		},
	}
	var seats []Seat
	if err := mongoDBConnection.findDocuments(ctx, databaseSubscription, collectionSeats, filter, &seats, options.Find()); err != nil {
		return 0, err
	}
	var totalBytes int64
	for _, seat := range seats {
		if seat.OpenCloudStorageBytes < 0 || seat.OpenCloudStorageBytes > math.MaxInt64-totalBytes {
			return 0, errors.New("invalid OpenCloud user allocation total")
		}
		totalBytes += seat.OpenCloudStorageBytes
	}
	return totalBytes, nil
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudSeatStorageBytes(ctx context.Context, subscriptionID, uidOrEmail string, storageBytes int64) error {
	if subscriptionID == "" || uidOrEmail == "" || storageBytes < 0 {
		return errors.New("subscription ID, seat identity, and valid storage allocation are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	identityFilter := bson.M{"uid": uidOrEmail}
	if strings.Contains(uidOrEmail, "@") {
		identityFilter = bson.M{"email": uidOrEmail}
	}
	filter := bson.M{"subscriptionID": subscriptionID}
	for key, value := range identityFilter {
		filter[key] = value
	}
	result, err := collection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"openCloudStorageBytes": storageBytes}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("subscription seat not found")
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudOwnerStorageBytes(ctx context.Context, subscriptionID string, storageBytes int64) error {
	if subscriptionID == "" || storageBytes < 0 {
		return errors.New("subscription ID and valid storage allocation are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"roles":          bson.M{"$in": []string{string(RoleOwner)}},
	}, bson.M{"$set": bson.M{"openCloudStorageBytes": storageBytes}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("subscription owner seat not found")
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudSpaceAllocation(ctx context.Context, subscriptionID, driveID string) (OpenCloudResourceRegistry, error) {
	if subscriptionID == "" || driveID == "" {
		return OpenCloudResourceRegistry{}, errors.New("subscription ID and OpenCloud space ID are required")
	}
	resources, err := mongoDBConnection.GetOpenCloudResourcesBySubscription(ctx, subscriptionID)
	if err != nil {
		return OpenCloudResourceRegistry{}, err
	}
	for _, resource := range resources {
		if resource.ResourceType == "space" && resource.Scope == "allocation" && resource.ResourceID == driveID {
			return resource, nil
		}
	}
	return OpenCloudResourceRegistry{}, errors.New("OpenCloud space allocation not found")
}
