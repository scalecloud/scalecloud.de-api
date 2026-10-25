package opencloudusage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
)

type fakeMongoDB struct {
	spaces       []mongodbmanager.OpenCloudSpaceReference
	history      map[string][]mongodbmanager.OpenCloudUsageRecord
	records      []mongodbmanager.OpenCloudUsageRecord
	resources    []mongodbmanager.OpenCloudResourceRegistry
	historyError error
}

func (fake *fakeMongoDB) GetOpenCloudProjectSpaces(context.Context) ([]mongodbmanager.OpenCloudSpaceReference, error) {
	return fake.spaces, nil
}

func (fake *fakeMongoDB) GetOpenCloudUsageHistory(_ context.Context, driveID string, _, _ time.Time) ([]mongodbmanager.OpenCloudUsageRecord, error) {
	if fake.historyError != nil {
		return nil, fake.historyError
	}
	return fake.history[driveID], nil
}

func (fake *fakeMongoDB) StoreOpenCloudDailyUsage(_ context.Context, record mongodbmanager.OpenCloudUsageRecord) error {
	fake.records = append(fake.records, record)
	return nil
}

func (fake *fakeMongoDB) UpsertOpenCloudResource(_ context.Context, resource mongodbmanager.OpenCloudResourceRegistry) error {
	fake.resources = append(fake.resources, resource)
	return nil
}

type fakeOpenCloud struct {
	spaces       []opencloudmanager.Space
	users        []opencloudmanager.User
	spaceUsage   map[string]*opencloudmanager.SpaceUsage
	userUsage    map[string]*opencloudmanager.SpaceUsage
	spaceCalls   []string
	userCalls    []string
	listSpaceErr error
	listUsersErr error
}

func (fake *fakeOpenCloud) ListProjectSpaces(context.Context) ([]opencloudmanager.Space, error) {
	return fake.spaces, fake.listSpaceErr
}

func (fake *fakeOpenCloud) GetSpaceUsage(_ context.Context, driveID string) (*opencloudmanager.SpaceUsage, error) {
	fake.spaceCalls = append(fake.spaceCalls, driveID)
	return fake.spaceUsage[driveID], nil
}

func (fake *fakeOpenCloud) ListUsers(context.Context) ([]opencloudmanager.User, error) {
	return fake.users, fake.listUsersErr
}

func (fake *fakeOpenCloud) GetUserDriveUsage(_ context.Context, userID string) (*opencloudmanager.SpaceUsage, error) {
	fake.userCalls = append(fake.userCalls, userID)
	return fake.userUsage[userID], nil
}

func TestCollectorSampleStoresProjectAndPersonalUsage(t *testing.T) {
	used, quota, remaining := int64(10), int64(100), int64(90)
	mongoDB := &fakeMongoDB{
		spaces:  []mongodbmanager.OpenCloudSpaceReference{{SubscriptionID: "subscription-1", DriveID: "drive-1"}},
		history: map[string][]mongodbmanager.OpenCloudUsageRecord{},
	}
	openCloud := &fakeOpenCloud{
		spaceUsage: map[string]*opencloudmanager.SpaceUsage{
			"drive-1": {Used: &used, Total: &quota, Remaining: &remaining, State: "normal"},
		},
		users: []opencloudmanager.User{{ID: "user-1", Mail: "user@example.com"}},
		userUsage: map[string]*opencloudmanager.SpaceUsage{
			"user-1": {Used: &used, Total: &quota, Remaining: &remaining, State: "normal"},
		},
	}

	if err := NewCollector(mongoDB, openCloud).Sample(context.Background()); err != nil {
		t.Fatalf("Sample() error = %v", err)
	}

	if len(openCloud.spaceCalls) != 1 || openCloud.spaceCalls[0] != "drive-1" {
		t.Fatalf("space usage calls = %v, want [drive-1]", openCloud.spaceCalls)
	}
	if len(openCloud.userCalls) != 1 || openCloud.userCalls[0] != "user-1" {
		t.Fatalf("user usage calls = %v, want [user-1]", openCloud.userCalls)
	}
	if len(mongoDB.records) != 2 {
		t.Fatalf("stored records = %d, want 2", len(mongoDB.records))
	}

	projectRecord := mongoDB.records[0]
	if projectRecord.ResourceType != "space" || projectRecord.ResourceID != "drive-1" || projectRecord.SubscriptionID != "subscription-1" || projectRecord.Scope != "project" {
		t.Errorf("project record identity = %+v", projectRecord)
	}
	if projectRecord.UsedBytes == nil || *projectRecord.UsedBytes != used || projectRecord.QuotaBytes == nil || *projectRecord.QuotaBytes != quota || projectRecord.SampledAt.IsZero() {
		t.Errorf("project record usage fields = %+v", projectRecord)
	}

	personalRecord := mongoDB.records[1]
	if personalRecord.ResourceType != "user" || personalRecord.ResourceID != "user-1" || personalRecord.UserID != "user-1" || personalRecord.UserEmail != "user@example.com" || personalRecord.Scope != "personal" {
		t.Errorf("personal record identity = %+v", personalRecord)
	}
	if personalRecord.UsedBytes == nil || *personalRecord.UsedBytes != used || personalRecord.SampledAt.IsZero() {
		t.Errorf("personal record usage fields = %+v", personalRecord)
	}
	if len(mongoDB.resources) != 1 || mongoDB.resources[0].ResourceType != "user" || mongoDB.resources[0].ResourceID != "user-1" {
		t.Errorf("registered resources = %+v, want user-1", mongoDB.resources)
	}
}

func TestCollectorSampleSkipsAlreadySampledProjectSpace(t *testing.T) {
	mongoDB := &fakeMongoDB{
		spaces: []mongodbmanager.OpenCloudSpaceReference{{DriveID: "drive-1"}},
		history: map[string][]mongodbmanager.OpenCloudUsageRecord{
			"drive-1": {{ResourceType: "space", ResourceID: "drive-1"}},
		},
	}
	openCloud := &fakeOpenCloud{
		spaceUsage: map[string]*opencloudmanager.SpaceUsage{},
		userUsage:  map[string]*opencloudmanager.SpaceUsage{},
	}

	if err := NewCollector(mongoDB, openCloud).Sample(context.Background()); err != nil {
		t.Fatalf("Sample() error = %v", err)
	}
	if len(openCloud.spaceCalls) != 0 {
		t.Errorf("space usage calls = %v, want none for already-sampled space", openCloud.spaceCalls)
	}
	if len(mongoDB.records) != 0 {
		t.Errorf("stored records = %d, want 0", len(mongoDB.records))
	}
}

func TestCollectorSampleReturnsFirstErrorAndContinues(t *testing.T) {
	historyErr := errors.New("history unavailable")
	mongoDB := &fakeMongoDB{
		spaces: []mongodbmanager.OpenCloudSpaceReference{
			{DriveID: "drive-1"},
			{DriveID: "drive-2"},
		},
		history:      map[string][]mongodbmanager.OpenCloudUsageRecord{},
		historyError: historyErr,
	}
	openCloud := &fakeOpenCloud{
		users:      []opencloudmanager.User{{ID: "user-1"}},
		spaceUsage: map[string]*opencloudmanager.SpaceUsage{},
		userUsage: map[string]*opencloudmanager.SpaceUsage{
			"user-1": {},
		},
	}

	err := NewCollector(mongoDB, openCloud).Sample(context.Background())
	if err == nil || !strings.Contains(err.Error(), historyErr.Error()) {
		t.Fatalf("Sample() error = %v, want history error", err)
	}
	if len(openCloud.userCalls) != 1 || openCloud.userCalls[0] != "user-1" {
		t.Errorf("user usage calls = %v, want processing to continue for user-1", openCloud.userCalls)
	}
	if len(mongoDB.records) != 1 || mongoDB.records[0].ResourceType != "user" {
		t.Errorf("stored records = %+v, want user record despite project history errors", mongoDB.records)
	}
}

func TestCollectorSampleRequiresDependencies(t *testing.T) {
	tests := []struct {
		name      string
		mongoDB   MongoDB
		openCloud OpenCloud
		wantError string
	}{
		{name: "MongoDB", openCloud: &fakeOpenCloud{}, wantError: "MongoDB connection is not configured"},
		{name: "OpenCloud", mongoDB: &fakeMongoDB{}, wantError: "OpenCloud integration is not configured"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewCollector(test.mongoDB, test.openCloud).Sample(context.Background())
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("Sample() error = %v, want %q", err, test.wantError)
			}
		})
	}
}
