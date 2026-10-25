package opencloudusage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
)

type Collector struct {
	mongoDB   *mongodbmanager.MongoDBConnection
	openCloud *opencloudmanager.Client
}

func NewCollector(mongoDB *mongodbmanager.MongoDBConnection, openCloud *opencloudmanager.Client) *Collector {
	return &Collector{mongoDB: mongoDB, openCloud: openCloud}
}

func (collector *Collector) Sample(ctx context.Context) error {
	if collector.openCloud == nil {
		return errors.New("OpenCloud integration is not configured")
	}
	if collector.mongoDB == nil {
		return errors.New("MongoDB connection is not configured")
	}

	spaces, err := collector.mongoDB.GetOpenCloudProjectSpaces(ctx)
	if err != nil {
		return err
	}
	var firstErr error
	if len(spaces) == 0 {
		discoveredSpaces, listErr := collector.openCloud.ListProjectSpaces(ctx)
		if listErr == nil {
			for _, space := range discoveredSpaces {
				if space.ID == "" {
					continue
				}
				spaces = append(spaces, mongodbmanager.OpenCloudSpaceReference{SubscriptionID: "", DriveID: space.ID})
				if upsertErr := collector.mongoDB.UpsertOpenCloudResource(ctx, mongodbmanager.OpenCloudResourceRegistry{
					SubscriptionID: "",
					ResourceType:   "space",
					ResourceID:     space.ID,
					Scope:          "project",
					DriveType:      space.DriveType,
					LastSeenAt:     time.Now().UTC(),
				}); upsertErr != nil && firstErr == nil {
					firstErr = fmt.Errorf("register OpenCloud space %s: %w", space.ID, upsertErr)
				}
			}
		} else if firstErr == nil {
			firstErr = listErr
		}
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, space := range spaces {
		existing, existingErr := collector.mongoDB.GetOpenCloudUsageHistory(ctx, space.DriveID, today, today.AddDate(0, 0, 1))
		if existingErr != nil {
			if firstErr == nil {
				firstErr = existingErr
			}
			continue
		}
		if len(existing) != 0 {
			continue
		}
		usage, usageErr := collector.openCloud.GetSpaceUsage(ctx, space.DriveID)
		if usageErr == nil {
			usageErr = collector.mongoDB.StoreOpenCloudDailyUsage(ctx, mongodbmanager.OpenCloudUsageRecord{
				ResourceType:   "space",
				ResourceID:     space.DriveID,
				SubscriptionID: space.SubscriptionID,
				Scope:          "project",
				DriveType:      "project",
				DriveID:        space.DriveID,
				SampledAt:      time.Now().UTC(),
				UsedBytes:      usage.Used,
				QuotaBytes:     usage.Total,
				RemainingBytes: usage.Remaining,
				State:          usage.State,
				Disabled:       usage.Disabled,
			})
		}
		if usageErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("sample OpenCloud space %s: %w", space.DriveID, usageErr)
		}
	}

	allUsers, userErr := collector.openCloud.ListUsers(ctx)
	if userErr != nil && firstErr == nil {
		firstErr = userErr
	} else if userErr == nil {
		for _, user := range allUsers {
			if user.ID == "" {
				continue
			}
			if err := collector.mongoDB.UpsertOpenCloudResource(ctx, mongodbmanager.OpenCloudResourceRegistry{
				SubscriptionID: "",
				ResourceType:   "user",
				ResourceID:     user.ID,
				Scope:          "personal",
				UserUID:        user.ID,
				UserEmail:      user.Mail,
				LastSeenAt:     time.Now().UTC(),
			}); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("register OpenCloud user %s: %w", user.ID, err)
			}
			personalUsage, usageErr := collector.openCloud.GetUserDriveUsage(ctx, user.ID)
			if usageErr == nil {
				usageErr = collector.mongoDB.StoreOpenCloudDailyUsage(ctx, mongodbmanager.OpenCloudUsageRecord{
					ResourceType:   "user",
					ResourceID:     user.ID,
					SubscriptionID: "",
					Scope:          "personal",
					DriveType:      "personal",
					DriveID:        user.ID,
					UserID:         user.ID,
					UserEmail:      user.Mail,
					SampledAt:      time.Now().UTC(),
					UsedBytes:      personalUsage.Used,
					QuotaBytes:     personalUsage.Total,
					RemainingBytes: personalUsage.Remaining,
					State:          personalUsage.State,
					Disabled:       personalUsage.Disabled,
				})
			}
			if usageErr != nil && firstErr == nil {
				firstErr = fmt.Errorf("sample OpenCloud user %s: %w", user.ID, usageErr)
			}
		}
	}
	return firstErr
}
