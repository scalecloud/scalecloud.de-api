package mongodbmanager

import (
	"context"
	"errors"
	"fmt"
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
	DriveID        string    `bson:"driveId" json:"driveId"`
	SubscriptionID string    `bson:"subscriptionID" json:"subscriptionID"`
	DriveType      string    `bson:"driveType" json:"driveType"`
	SampledAt      time.Time `bson:"sampledAt" json:"sampledAt"`
	Day            time.Time `bson:"day" json:"day"`
	UsedBytes      *int64    `bson:"usedBytes,omitempty" json:"usedBytes,omitempty"`
	QuotaBytes     *int64    `bson:"quotaBytes,omitempty" json:"quotaBytes,omitempty"`
	RemainingBytes *int64    `bson:"remainingBytes,omitempty" json:"remainingBytes,omitempty"`
	State          string    `bson:"state,omitempty" json:"state,omitempty"`
	Disabled       bool      `bson:"disabled" json:"disabled"`
}

func (mongoDBConnection *MongoDBConnection) ensureOpenCloudUsageIndexes(ctx context.Context) error {
	history, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudHistory)
	if err != nil {
		return err
	}
	_, err = history.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "driveId", Value: 1}, {Key: "day", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("UniqueOpenCloudUsageDriveDay"),
		},
		{
			Keys:    bson.D{{Key: "sampledAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(45 * 24 * 60 * 60).SetName("ExpireOpenCloudUsageHistory"),
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
	if record.DriveID == "" || record.SampledAt.IsZero() {
		return errors.New("OpenCloud drive and sample time are required")
	}
	record.SampledAt = record.SampledAt.UTC()
	record.Day = time.Date(record.SampledAt.Year(), record.SampledAt.Month(), record.SampledAt.Day(), 0, 0, 0, 0, time.UTC)
	historyID := fmt.Sprintf("%s:%s", record.DriveID, record.Day.Format("2006-01-02"))
	history, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudHistory)
	if err != nil {
		return err
	}
	document := bson.M{
		"_id":            historyID,
		"driveId":        record.DriveID,
		"subscriptionID": record.SubscriptionID,
		"driveType":      record.DriveType,
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
