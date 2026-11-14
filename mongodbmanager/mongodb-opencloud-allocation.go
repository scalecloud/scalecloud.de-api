package mongodbmanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

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

func (mongoDBConnection *MongoDBConnection) DeleteOpenCloudResource(ctx context.Context, subscriptionID, resourceID string) error {
	if subscriptionID == "" || resourceID == "" {
		return errors.New("subscription ID and OpenCloud resource ID are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudRegistry)
	if err != nil {
		return err
	}
	result, err := collection.DeleteOne(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"resourceType":   "space",
		"scope":          "allocation",
		"resourceID":     resourceID,
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("OpenCloud space allocation not found")
	}
	return nil
}
