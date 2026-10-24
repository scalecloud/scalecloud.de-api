package stripemanager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
	"go.uber.org/zap"
)

var ErrOpenCloudUnavailable = errors.New("OpenCloud integration is not configured")
var ErrOpenCloudSpaceMissing = errors.New("OpenCloud space has not been created for this subscription")

type OpenCloudSubscriptionRequest struct {
	SubscriptionID string `json:"subscriptionID" binding:"required"`
}

type OpenCloudSpaceReply struct {
	SubscriptionID string                  `json:"subscriptionID"`
	Space          *opencloudmanager.Space `json:"space"`
}

type OpenCloudUsageReply struct {
	SubscriptionID string                        `json:"subscriptionID"`
	ProjectSpace   *OpenCloudDriveUsage          `json:"projectSpace,omitempty"`
	PersonalDrives []OpenCloudPersonalDriveUsage `json:"personalDrives"`
	SampledAt      time.Time                     `json:"sampledAt"`
}

type OpenCloudUsageHistoryReply struct {
	SubscriptionID string                `json:"subscriptionID"`
	History        []OpenCloudUsagePoint `json:"history"`
}

type OpenCloudDriveUsage struct {
	DriveID string                       `json:"driveId"`
	Usage   *opencloudmanager.SpaceUsage `json:"usage"`
}

type OpenCloudPersonalDriveUsage struct {
	UID         string                       `json:"uid"`
	Email       string                       `json:"email"`
	UserID      string                       `json:"userId,omitempty"`
	Provisioned bool                         `json:"provisioned"`
	Usage       *opencloudmanager.SpaceUsage `json:"usage,omitempty"`
}

type OpenCloudUsagePoint struct {
	Date           string    `json:"date"`
	SampledAt      time.Time `json:"sampledAt"`
	UsedBytes      *int64    `json:"usedBytes,omitempty"`
	QuotaBytes     *int64    `json:"quotaBytes,omitempty"`
	RemainingBytes *int64    `json:"remainingBytes,omitempty"`
	Disabled       bool      `json:"disabled"`
}

const openCloudUsageCacheTTL = 30 * time.Second
const openCloudUsageCacheMaxEntries = 10000

type openCloudUsageCacheEntry struct {
	reply     OpenCloudUsageReply
	expiresAt time.Time
}

type openCloudUsageFlight struct {
	done  chan struct{}
	reply OpenCloudUsageReply
	err   error
}

type OpenCloudQuotaReply struct {
	SubscriptionID string                  `json:"subscriptionID"`
	Space          *opencloudmanager.Space `json:"space"`
}

func (paymentHandler *PaymentHandler) CreateOpenCloudSpace(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudSubscriptionRequest) (OpenCloudSpaceReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudSpaceReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudSpaceReply{}, err
	}
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudSpaceReply{}, err
	}
	if owner.SubscriptionID == "" {
		return OpenCloudSpaceReply{}, errors.New("subscription owner seat not found")
	}
	var space *opencloudmanager.Space
	if owner.OpenCloudDriveID != "" {
		space, err = paymentHandler.OpenCloudConnection.GetSpace(ctx, owner.OpenCloudDriveID)
	} else {
		_, amount, unit, err := getSeatProductDetails(ctx, paymentHandler.StripeConnection, request.SubscriptionID)
		if err != nil {
			return OpenCloudSpaceReply{}, err
		}
		quotaBytes, err := storageQuotaBytes(amount, unit)
		if err != nil {
			return OpenCloudSpaceReply{}, err
		}
		marker := "scalecloud-subscription:" + request.SubscriptionID
		space, err = paymentHandler.OpenCloudConnection.FindProjectSpace(ctx, marker)
		if errors.Is(err, opencloudmanager.ErrNotFound) {
			space, err = paymentHandler.OpenCloudConnection.CreateSpace(ctx, "Subscription "+request.SubscriptionID, marker, quotaBytes)
		}
		if err != nil {
			return OpenCloudSpaceReply{}, err
		}
		if err := paymentHandler.MongoDBConnection.SetOpenCloudDriveID(ctx, request.SubscriptionID, space.ID); err != nil {
			return OpenCloudSpaceReply{}, err
		}
	}
	if err != nil {
		return OpenCloudSpaceReply{}, err
	}
	if err := paymentHandler.ensureLinkedSeatsInOpenCloudSpace(ctx, request.SubscriptionID, space.ID); err != nil {
		return OpenCloudSpaceReply{}, err
	}
	return OpenCloudSpaceReply{SubscriptionID: request.SubscriptionID, Space: space}, nil
}

func (paymentHandler *PaymentHandler) ensureLinkedSeatsInOpenCloudSpace(ctx context.Context, subscriptionID, driveID string) error {
	seats, err := paymentHandler.MongoDBConnection.GetAllSeats(ctx, subscriptionID)
	if err != nil {
		return err
	}
	for _, seat := range seats {
		if (seat.Status != "" && seat.Status != mongodbmanager.SeatStatusActive) || seat.OpenCloudUserID == "" {
			continue
		}
		if _, err := paymentHandler.OpenCloudConnection.EnsureMember(ctx, driveID, seat.OpenCloudUserID, opencloudmanager.SpaceRoleEditor); err != nil {
			return fmt.Errorf("add seat %s to OpenCloud subscription space: %w", seat.EMail, err)
		}
	}
	return nil
}

func (paymentHandler *PaymentHandler) ApplyOpenCloudStorage(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudSubscriptionRequest) (OpenCloudQuotaReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudQuotaReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudQuotaReply{}, err
	}
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudQuotaReply{}, err
	}
	if owner.OpenCloudDriveID == "" {
		return OpenCloudQuotaReply{}, ErrOpenCloudSpaceMissing
	}
	_, amount, unit, err := getSeatProductDetails(ctx, paymentHandler.StripeConnection, request.SubscriptionID)
	if err != nil {
		return OpenCloudQuotaReply{}, err
	}
	quotaBytes, err := storageQuotaBytes(amount, unit)
	if err != nil {
		return OpenCloudQuotaReply{}, err
	}
	space, err := paymentHandler.OpenCloudConnection.SetSpaceQuota(ctx, owner.OpenCloudDriveID, quotaBytes)
	if err != nil {
		return OpenCloudQuotaReply{}, err
	}
	return OpenCloudQuotaReply{SubscriptionID: request.SubscriptionID, Space: space}, nil
}

func (paymentHandler *PaymentHandler) GetOpenCloudUsageHistory(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudSubscriptionRequest) (OpenCloudUsageHistoryReply, error) {
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator, mongodbmanager.RoleBilling, mongodbmanager.RoleUser}); err != nil {
		return OpenCloudUsageHistoryReply{}, err
	}
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudUsageHistoryReply{}, err
	}
	if owner.OpenCloudDriveID == "" {
		return OpenCloudUsageHistoryReply{SubscriptionID: request.SubscriptionID, History: []OpenCloudUsagePoint{}}, nil
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	from := today.AddDate(0, 0, -29)
	records, err := paymentHandler.MongoDBConnection.GetOpenCloudUsageHistory(ctx, owner.OpenCloudDriveID, from, today.AddDate(0, 0, 1))
	if err != nil {
		return OpenCloudUsageHistoryReply{}, err
	}
	points := make([]OpenCloudUsagePoint, 0, len(records))
	for _, record := range records {
		points = append(points, openCloudUsagePoint(record))
	}
	return OpenCloudUsageHistoryReply{SubscriptionID: request.SubscriptionID, History: points}, nil
}

func openCloudUsagePoint(record mongodbmanager.OpenCloudUsageRecord) OpenCloudUsagePoint {
	return OpenCloudUsagePoint{Date: record.Day.Format("2006-01-02"), SampledAt: record.SampledAt, UsedBytes: record.UsedBytes, QuotaBytes: record.QuotaBytes, RemainingBytes: record.RemainingBytes, Disabled: record.Disabled}
}

func (paymentHandler *PaymentHandler) GetOpenCloudUsage(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudSubscriptionRequest) (OpenCloudUsageReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudUsageReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{
		mongodbmanager.RoleAdministrator,
		mongodbmanager.RoleBilling,
		mongodbmanager.RoleUser,
	}); err != nil {
		return OpenCloudUsageReply{}, err
	}
	return paymentHandler.cachedOpenCloudUsage(ctx, request.SubscriptionID)
}

func (paymentHandler *PaymentHandler) cachedOpenCloudUsage(ctx context.Context, subscriptionID string) (OpenCloudUsageReply, error) {
	return paymentHandler.getOrLoadOpenCloudUsage(ctx, subscriptionID, paymentHandler.loadOpenCloudUsage)
}

func (paymentHandler *PaymentHandler) getOrLoadOpenCloudUsage(ctx context.Context, subscriptionID string, load func(context.Context, string) (OpenCloudUsageReply, error)) (OpenCloudUsageReply, error) {
	paymentHandler.openCloudUsageMu.Lock()
	if paymentHandler.openCloudUsageCache == nil {
		paymentHandler.openCloudUsageCache = make(map[string]openCloudUsageCacheEntry)
	}
	if paymentHandler.openCloudUsageFlights == nil {
		paymentHandler.openCloudUsageFlights = make(map[string]*openCloudUsageFlight)
	}
	now := time.Now()
	for key, entry := range paymentHandler.openCloudUsageCache {
		if !now.Before(entry.expiresAt) {
			delete(paymentHandler.openCloudUsageCache, key)
		}
	}
	if entry, ok := paymentHandler.openCloudUsageCache[subscriptionID]; ok {
		paymentHandler.openCloudUsageMu.Unlock()
		return entry.reply, nil
	}
	if flight, ok := paymentHandler.openCloudUsageFlights[subscriptionID]; ok {
		paymentHandler.openCloudUsageMu.Unlock()
		select {
		case <-ctx.Done():
			return OpenCloudUsageReply{}, ctx.Err()
		case <-flight.done:
			return flight.reply, flight.err
		}
	}
	flight := &openCloudUsageFlight{done: make(chan struct{})}
	paymentHandler.openCloudUsageFlights[subscriptionID] = flight
	paymentHandler.openCloudUsageMu.Unlock()

	reply, err := load(ctx, subscriptionID)
	paymentHandler.openCloudUsageMu.Lock()
	flight.reply, flight.err = reply, err
	if err == nil {
		paymentHandler.pruneOpenCloudUsageCache(time.Now())
		if len(paymentHandler.openCloudUsageCache) >= openCloudUsageCacheMaxEntries {
			var oldestKey string
			var oldestExpiry time.Time
			for key, entry := range paymentHandler.openCloudUsageCache {
				if oldestKey == "" || entry.expiresAt.Before(oldestExpiry) {
					oldestKey, oldestExpiry = key, entry.expiresAt
				}
			}
			delete(paymentHandler.openCloudUsageCache, oldestKey)
		}
		paymentHandler.openCloudUsageCache[subscriptionID] = openCloudUsageCacheEntry{reply: reply, expiresAt: time.Now().Add(openCloudUsageCacheTTL)}
	}
	delete(paymentHandler.openCloudUsageFlights, subscriptionID)
	close(flight.done)
	paymentHandler.openCloudUsageMu.Unlock()
	return reply, err
}

func (paymentHandler *PaymentHandler) pruneOpenCloudUsageCache(now time.Time) {
	for key, entry := range paymentHandler.openCloudUsageCache {
		if !now.Before(entry.expiresAt) {
			delete(paymentHandler.openCloudUsageCache, key)
		}
	}
}

func (paymentHandler *PaymentHandler) loadOpenCloudUsage(ctx context.Context, subscriptionID string) (OpenCloudUsageReply, error) {
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, subscriptionID)
	if err != nil {
		return OpenCloudUsageReply{}, err
	}
	if owner.SubscriptionID == "" {
		return OpenCloudUsageReply{}, errors.New("subscription owner seat not found")
	}
	reply := OpenCloudUsageReply{SubscriptionID: subscriptionID, PersonalDrives: []OpenCloudPersonalDriveUsage{}, SampledAt: time.Now().UTC()}
	if owner.OpenCloudDriveID != "" {
		usage, err := paymentHandler.OpenCloudConnection.GetSpaceUsage(ctx, owner.OpenCloudDriveID)
		if err != nil {
			return OpenCloudUsageReply{}, err
		}
		paymentHandler.Log.Info("Fetched current OpenCloud project-space usage",
			zap.String("subscription_id", subscriptionID),
			zap.String("drive_id", owner.OpenCloudDriveID),
			zap.Any("used_bytes", usage.Used),
			zap.Any("quota_bytes", usage.Total),
		)
		reply.ProjectSpace = &OpenCloudDriveUsage{DriveID: owner.OpenCloudDriveID, Usage: usage}
	}
	seats, err := paymentHandler.MongoDBConnection.GetAllSeats(ctx, subscriptionID)
	if err != nil {
		return OpenCloudUsageReply{}, err
	}
	seenUsers := make(map[string]struct{}, len(seats))
	for _, seat := range seats {
		if seat.Status != "" && seat.Status != mongodbmanager.SeatStatusActive {
			continue
		}
		if seat.OpenCloudUserID == "" {
			reply.PersonalDrives = append(reply.PersonalDrives, OpenCloudPersonalDriveUsage{UID: seat.UID, Email: seat.EMail})
			continue
		}
		if _, exists := seenUsers[seat.OpenCloudUserID]; exists {
			continue
		}
		seenUsers[seat.OpenCloudUserID] = struct{}{}
		usage, err := paymentHandler.OpenCloudConnection.GetUserDriveUsage(ctx, seat.OpenCloudUserID)
		if errors.Is(err, opencloudmanager.ErrNotFound) {
			reply.PersonalDrives = append(reply.PersonalDrives, OpenCloudPersonalDriveUsage{UID: seat.UID, Email: seat.EMail, UserID: seat.OpenCloudUserID})
			continue
		}
		if err != nil {
			return OpenCloudUsageReply{}, err
		}
		paymentHandler.Log.Info("Fetched current OpenCloud personal-drive usage",
			zap.String("subscription_id", subscriptionID),
			zap.String("opencloud_user_id", seat.OpenCloudUserID),
			zap.Any("used_bytes", usage.Used),
			zap.Any("quota_bytes", usage.Total),
		)
		reply.PersonalDrives = append(reply.PersonalDrives, OpenCloudPersonalDriveUsage{UID: seat.UID, Email: seat.EMail, UserID: seat.OpenCloudUserID, Provisioned: true, Usage: usage})
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) SampleOpenCloudUsage(ctx context.Context) error {
	if paymentHandler.OpenCloudConnection == nil {
		return ErrOpenCloudUnavailable
	}
	spaces, err := paymentHandler.MongoDBConnection.GetOpenCloudProjectSpaces(ctx)
	if err != nil {
		return err
	}
	var firstErr error
	if len(spaces) == 0 {
		discoveredSpaces, listErr := paymentHandler.OpenCloudConnection.ListProjectSpaces(ctx)
		if listErr == nil {
			for _, space := range discoveredSpaces {
				if space.ID == "" {
					continue
				}
				spaces = append(spaces, mongodbmanager.OpenCloudSpaceReference{SubscriptionID: "", DriveID: space.ID})
				if upsertErr := paymentHandler.MongoDBConnection.UpsertOpenCloudResource(ctx, mongodbmanager.OpenCloudResourceRegistry{
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
		existing, existingErr := paymentHandler.MongoDBConnection.GetOpenCloudUsageHistory(ctx, space.DriveID, today, today.AddDate(0, 0, 1))
		if existingErr != nil {
			if firstErr == nil {
				firstErr = existingErr
			}
			continue
		}
		if len(existing) != 0 {
			continue
		}
		usage, usageErr := paymentHandler.OpenCloudConnection.GetSpaceUsage(ctx, space.DriveID)
		if usageErr == nil {
			usageErr = paymentHandler.MongoDBConnection.StoreOpenCloudDailyUsage(ctx, mongodbmanager.OpenCloudUsageRecord{
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

	allUsers, userErr := paymentHandler.OpenCloudConnection.ListUsers(ctx)
	if userErr != nil && firstErr == nil {
		firstErr = userErr
	} else if userErr == nil {
		for _, user := range allUsers {
			if user.ID == "" {
				continue
			}
			if err := paymentHandler.MongoDBConnection.UpsertOpenCloudResource(ctx, mongodbmanager.OpenCloudResourceRegistry{
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
			personalUsage, usageErr := paymentHandler.OpenCloudConnection.GetUserDriveUsage(ctx, user.ID)
			if usageErr == nil {
				usageErr = paymentHandler.MongoDBConnection.StoreOpenCloudDailyUsage(ctx, mongodbmanager.OpenCloudUsageRecord{
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

func (paymentHandler *PaymentHandler) DisableOpenCloudSpace(ctx context.Context, subscriptionID string) error {
	if paymentHandler.OpenCloudConnection == nil {
		return nil
	}
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, subscriptionID)
	if err != nil || owner.OpenCloudDriveID == "" {
		return err
	}
	return paymentHandler.OpenCloudConnection.DisableSpace(ctx, owner.OpenCloudDriveID)
}

func (paymentHandler *PaymentHandler) ActivateOpenCloudSeats(ctx context.Context, authentikUserPK int64, username string) (int, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return 0, ErrOpenCloudUnavailable
	}
	seats, err := paymentHandler.MongoDBConnection.GetActiveSeatsByAuthentikUserPK(ctx, authentikUserPK)
	if err != nil || len(seats) == 0 {
		return 0, err
	}
	user, err := paymentHandler.OpenCloudConnection.GetUser(ctx, username)
	if err != nil {
		return 0, err
	}
	if user.ID == "" {
		return 0, errors.New("OpenCloud user response did not include an ID")
	}
	activated := 0
	var firstErr error
	for _, seat := range seats {
		owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, seat.SubscriptionID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := paymentHandler.MongoDBConnection.SetOpenCloudUserID(ctx, seat.SubscriptionID, authentikUserPK, user.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if owner.OpenCloudDriveID == "" {
			continue
		}
		if _, err := paymentHandler.OpenCloudConnection.EnsureMember(ctx, owner.OpenCloudDriveID, user.ID, opencloudmanager.SpaceRoleEditor); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		activated++
	}
	return activated, firstErr
}

func storageQuotaBytes(amount int64, unit string) (int64, error) {
	if amount <= 0 {
		return 0, opencloudmanager.ErrInvalidQuota
	}
	var multiplier int64
	switch strings.ToUpper(strings.TrimSpace(unit)) {
	case "B":
		multiplier = 1
	case "KB":
		multiplier = 1000
	case "MB":
		multiplier = 1000 * 1000
	case "GB":
		multiplier = 1000 * 1000 * 1000
	case "TB":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "KIB":
		multiplier = 1 << 10
	case "MIB":
		multiplier = 1 << 20
	case "GIB":
		multiplier = 1 << 30
	case "TIB":
		multiplier = 1 << 40
	default:
		return 0, fmt.Errorf("unsupported storage unit %q", unit)
	}
	if amount > math.MaxInt64/multiplier {
		return 0, errors.New("storage quota exceeds supported byte range")
	}
	return amount * multiplier, nil
}
