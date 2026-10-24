package mongodbmanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type OpenCloudSpaceReference struct {
	SubscriptionID string `bson:"subscriptionID"`
	DriveID        string `bson:"openCloudDriveId"`
}

type OpenCloudUsageRecord struct {
	DriveID        string    `bson:"driveId,omitempty" json:"driveId,omitempty"`
	ResourceType   string    `bson:"resourceType,omitempty" json:"resourceType,omitempty"`
	ResourceID     string    `bson:"resourceID,omitempty" json:"resourceID,omitempty"`
	SubscriptionID string    `bson:"subscriptionID,omitempty" json:"subscriptionID,omitempty"`
	Scope          string    `bson:"scope,omitempty" json:"scope,omitempty"`
	DriveType      string    `bson:"driveType,omitempty" json:"driveType,omitempty"`
	UserID         string    `bson:"userId,omitempty" json:"userId,omitempty"`
	UserEmail      string    `bson:"userEmail,omitempty" json:"userEmail,omitempty"`
	SampledAt      time.Time `bson:"sampledAt" json:"sampledAt"`
	Day            time.Time `bson:"day" json:"day"`
	UsedBytes      *int64    `bson:"usedBytes,omitempty" json:"usedBytes,omitempty"`
	QuotaBytes     *int64    `bson:"quotaBytes,omitempty" json:"quotaBytes,omitempty"`
	RemainingBytes *int64    `bson:"remainingBytes,omitempty" json:"remainingBytes,omitempty"`
	State          string    `bson:"state,omitempty" json:"state,omitempty"`
	Disabled       bool      `bson:"disabled" json:"disabled"`
}

type OpenCloudResourceRegistry struct {
	SubscriptionID string    `bson:"subscriptionID,omitempty" json:"subscriptionID,omitempty"`
	ResourceType   string    `bson:"resourceType" json:"resourceType"`
	ResourceID     string    `bson:"resourceID" json:"resourceID"`
	Scope          string    `bson:"scope,omitempty" json:"scope,omitempty"`
	DriveType      string    `bson:"driveType,omitempty" json:"driveType,omitempty"`
	UserUID        string    `bson:"userUID,omitempty" json:"userUID,omitempty"`
	UserEmail      string    `bson:"userEmail,omitempty" json:"userEmail,omitempty"`
	LastSeenAt     time.Time `bson:"lastSeenAt" json:"lastSeenAt"`
}

func (mongoDBConnection *MongoDBConnection) ensureOpenCloudUsageIndexes(ctx context.Context) error {
	history, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudHistory)
	if err != nil {
		return err
	}
	_, err = history.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "resourceType", Value: 1}, {Key: "resourceID", Value: 1}, {Key: "day", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("UniqueOpenCloudUsageResourceDay"),
		},
		{
			Keys:    bson.D{{Key: "subscriptionID", Value: 1}, {Key: "day", Value: 1}},
			Options: options.Index().SetName("OpenCloudUsageSubscriptionDay"),
		},
		{
			Keys:    bson.D{{Key: "sampledAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(45 * 24 * 60 * 60).SetName("ExpireOpenCloudUsageHistory"),
		},
	})
	return err
}

func (mongoDBConnection *MongoDBConnection) ensureOpenCloudRegistryIndexes(ctx context.Context) error {
	registry, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudRegistry)
	if err != nil {
		return err
	}
	_, err = registry.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "subscriptionID", Value: 1}, {Key: "resourceType", Value: 1}, {Key: "resourceID", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("UniqueOpenCloudRegistryResource"),
		},
		{
			Keys:    bson.D{{Key: "lastSeenAt", Value: 1}},
			Options: options.Index().SetName("OpenCloudRegistryLastSeen"),
		},
	})
	return err
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudProjectSpaces(ctx context.Context) ([]OpenCloudSpaceReference, error) {
	filter := bson.M{
		"roles":            bson.M{"$in": []string{string(RoleOwner)}},
		"openCloudDriveId": bson.M{"$exists": true, "$ne": ""},
	}
	opts := options.Find().SetProjection(bson.M{"_id": 0, "subscriptionID": 1, "openCloudDriveId": 1})
	var references []OpenCloudSpaceReference
	if err := mongoDBConnection.findDocuments(ctx, databaseSubscription, collectionSeats, filter, &references, opts); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(references))
	unique := references[:0]
	for _, reference := range references {
		if _, exists := seen[reference.DriveID]; exists {
			continue
		}
		seen[reference.DriveID] = struct{}{}
		unique = append(unique, reference)
	}
	return unique, nil
}

func (mongoDBConnection *MongoDBConnection) StoreOpenCloudDailyUsage(ctx context.Context, record OpenCloudUsageRecord) error {
	if record.SampledAt.IsZero() {
		return errors.New("OpenCloud sample time is required")
	}
	resourceID := strings.TrimSpace(record.ResourceID)
	if resourceID == "" {
		resourceID = strings.TrimSpace(record.DriveID)
	}
	if resourceID == "" {
		return errors.New("OpenCloud resource ID is required")
	}
	if strings.TrimSpace(record.ResourceType) == "" {
		record.ResourceType = "space"
	}
	if strings.TrimSpace(record.Scope) == "" {
		if record.ResourceType == "user" {
			record.Scope = "personal"
		} else {
			record.Scope = "project"
		}
	}
	record.ResourceID = resourceID
	record.DriveID = resourceID
	record.SampledAt = record.SampledAt.UTC()
	record.Day = time.Date(record.SampledAt.Year(), record.SampledAt.Month(), record.SampledAt.Day(), 0, 0, 0, 0, time.UTC)
	historyID := fmt.Sprintf("%s:%s:%s", record.ResourceType, record.ResourceID, record.Day.Format("2006-01-02"))
	history, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudHistory)
	if err != nil {
		return err
	}
	document := bson.M{
		"_id":            historyID,
		"driveId":        record.DriveID,
		"resourceType":   record.ResourceType,
		"resourceID":     record.ResourceID,
		"subscriptionID": record.SubscriptionID,
		"scope":          record.Scope,
		"driveType":      record.DriveType,
		"userId":         record.UserID,
		"userEmail":      record.UserEmail,
		"sampledAt":      record.SampledAt,
		"day":            record.Day,
		"usedBytes":      record.UsedBytes,
		"quotaBytes":     record.QuotaBytes,
		"remainingBytes": record.RemainingBytes,
		"state":          record.State,
		"disabled":       record.Disabled,
	}
	filter := bson.M{"_id": historyID}
	_, err = history.ReplaceOne(ctx, filter, document, options.Replace().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		_, err = history.ReplaceOne(ctx, filter, document)
	}
	if err != nil {
		return fmt.Errorf("store OpenCloud daily usage: %w", err)
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) UpsertOpenCloudResource(ctx context.Context, resource OpenCloudResourceRegistry) error {
	if resource.ResourceType == "" || resource.ResourceID == "" {
		return errors.New("OpenCloud resource type and resource ID are required")
	}
	if resource.LastSeenAt.IsZero() {
		resource.LastSeenAt = time.Now().UTC()
	}
	registry, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudRegistry)
	if err != nil {
		return err
	}
	filter := bson.M{
		"subscriptionID": resource.SubscriptionID,
		"resourceType":   resource.ResourceType,
		"resourceID":     resource.ResourceID,
	}
	document := bson.M{
		"subscriptionID": resource.SubscriptionID,
		"resourceType":   resource.ResourceType,
		"resourceID":     resource.ResourceID,
		"scope":          resource.Scope,
		"driveType":      resource.DriveType,
		"userUID":        resource.UserUID,
		"userEmail":      resource.UserEmail,
		"lastSeenAt":     resource.LastSeenAt,
	}
	_, err = registry.UpdateOne(ctx, filter, bson.M{"$set": document}, options.UpdateOne().SetUpsert(true))
	return err
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudResourcesBySubscription(ctx context.Context, subscriptionID string) ([]OpenCloudResourceRegistry, error) {
	if subscriptionID == "" {
		return nil, errors.New("subscription ID is required")
	}
	filter := bson.M{"subscriptionID": subscriptionID}
	var records []OpenCloudResourceRegistry
	if err := mongoDBConnection.findDocuments(ctx, databaseOpenCloud, collectionOpenCloudRegistry, filter, &records, options.Find()); err != nil {
		return nil, err
	}
	return records, nil
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudUsageHistory(ctx context.Context, driveID string, from, to time.Time) ([]OpenCloudUsageRecord, error) {
	if driveID == "" || from.IsZero() || to.IsZero() || !to.After(from) {
		return nil, errors.New("OpenCloud drive ID and valid time range are required")
	}
	filter := bson.M{
		"driveId": driveID,
		"day": bson.M{
			"$gte": from.UTC(),
			"$lt":  to.UTC(),
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "sampledAt", Value: 1}})
	var records []OpenCloudUsageRecord
	if err := mongoDBConnection.findDocuments(ctx, databaseOpenCloud, collectionOpenCloudHistory, filter, &records, opts); err != nil {
		return nil, err
	}
	return records, nil
}
