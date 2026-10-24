package stripemanager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
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
	SubscriptionID string                  `json:"subscriptionID"`
	Provisioned    bool                    `json:"provisioned"`
	Usage          *opencloudmanager.Usage `json:"usage,omitempty"`
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
	if owner.OpenCloudDriveID != "" {
		space, err := paymentHandler.OpenCloudConnection.GetSpace(ctx, owner.OpenCloudDriveID)
		if err != nil {
			return OpenCloudSpaceReply{}, err
		}
		return OpenCloudSpaceReply{SubscriptionID: request.SubscriptionID, Space: space}, nil
	}
	_, amount, unit, err := getSeatProductDetails(ctx, paymentHandler.StripeConnection, request.SubscriptionID)
	if err != nil {
		return OpenCloudSpaceReply{}, err
	}
	quotaBytes, err := storageQuotaBytes(amount, unit)
	if err != nil {
		return OpenCloudSpaceReply{}, err
	}
	marker := "scalecloud-subscription:" + request.SubscriptionID
	space, err := paymentHandler.OpenCloudConnection.FindProjectSpace(ctx, marker)
	if errors.Is(err, opencloudmanager.ErrNotFound) {
		space, err = paymentHandler.OpenCloudConnection.CreateSpace(ctx, "Subscription "+request.SubscriptionID, marker, quotaBytes)
	}
	if err != nil {
		return OpenCloudSpaceReply{}, err
	}
	if err := paymentHandler.MongoDBConnection.SetOpenCloudDriveID(ctx, request.SubscriptionID, space.ID); err != nil {
		return OpenCloudSpaceReply{}, err
	}
	return OpenCloudSpaceReply{SubscriptionID: request.SubscriptionID, Space: space}, nil
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
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudUsageReply{}, err
	}
	if owner.OpenCloudDriveID == "" {
		return OpenCloudUsageReply{SubscriptionID: request.SubscriptionID, Provisioned: false}, nil
	}
	usage, err := paymentHandler.OpenCloudConnection.GetSpaceUsage(ctx, owner.OpenCloudDriveID)
	if err != nil {
		return OpenCloudUsageReply{}, err
	}
	return OpenCloudUsageReply{SubscriptionID: request.SubscriptionID, Provisioned: true, Usage: usage}, nil
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
		if owner.OpenCloudDriveID == "" {
			continue
		}
		if _, err := paymentHandler.OpenCloudConnection.EnsureMember(ctx, owner.OpenCloudDriveID, user.ID, opencloudmanager.RoleEditor); err != nil {
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
