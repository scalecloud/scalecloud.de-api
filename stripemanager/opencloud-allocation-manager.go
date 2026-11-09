package stripemanager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
	"github.com/stripe/stripe-go/v87"
	"go.uber.org/zap"
)

const openCloudAllocatedSpaceScope = "allocation"

type OpenCloudAllocationOverviewReply struct {
	SubscriptionID string                    `json:"subscriptionID" validate:"required"`
	TotalQuantity  int64                     `json:"totalQuantity" validate:"gte=1"`
	StorageAmount  int64                     `json:"storageAmount" validate:"gte=1"`
	StorageUnit    string                    `json:"storageUnit" validate:"required"`
	Owner          OpenCloudOwnerAllocation  `json:"owner" validate:"required"`
	Users          []OpenCloudUserAllocation `json:"users" validate:"required"`
	Spaces         []OpenCloudManagedSpace   `json:"spaces" validate:"required"`
}

type OpenCloudOwnerAllocation struct {
	UID             string                          `json:"uid" validate:"required"`
	Email           string                          `json:"email" validate:"required"`
	OpenCloudLinked bool                            `json:"openCloudLinked"`
	StorageQuantity int64                           `json:"storageQuantity" validate:"gte=0"`
	UsedQuantity    float64                         `json:"usedQuantity"`
	DailyUsage      []OpenCloudAllocationUsagePoint `json:"dailyUsage" validate:"required"`
}

type OpenCloudUserAllocation struct {
	UID             string                          `json:"uid" validate:"required"`
	Email           string                          `json:"email" validate:"required"`
	Roles           []mongodbmanager.Role           `json:"roles" validate:"required"`
	OpenCloudLinked bool                            `json:"openCloudLinked"`
	StorageQuantity int64                           `json:"storageQuantity" validate:"gte=0"`
	UsedQuantity    float64                         `json:"usedQuantity"`
	DailyUsage      []OpenCloudAllocationUsagePoint `json:"dailyUsage" validate:"required"`
}

type OpenCloudManagedSpace struct {
	ID              string                          `json:"id" validate:"required"`
	Name            string                          `json:"name" validate:"required"`
	StorageQuantity int64                           `json:"storageQuantity" validate:"gte=1"`
	MemberCount     int64                           `json:"memberCount" validate:"gte=0"`
	Members         []OpenCloudManagedSpaceMember   `json:"members" validate:"required"`
	UsedQuantity    float64                         `json:"usedQuantity"`
	DailyUsage      []OpenCloudAllocationUsagePoint `json:"dailyUsage" validate:"required"`
}

type OpenCloudManagedSpaceMember struct {
	PermissionID string `json:"permissionID" validate:"required"`
	UID          string `json:"uid,omitempty"`
	UserID       string `json:"userID,omitempty"`
	GroupID      string `json:"groupID,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	Email        string `json:"email,omitempty"`
}

type OpenCloudAllocationUsagePoint struct {
	Date         string  `json:"date" validate:"required"`
	UsedQuantity float64 `json:"usedQuantity"`
}

type OpenCloudInviteUserRequest struct {
	SubscriptionID  string `json:"subscriptionID" binding:"required" validate:"required"`
	Email           string `json:"email" binding:"required,email" validate:"required,email"`
	StorageQuantity int64  `json:"storageQuantity" binding:"required,min=1" validate:"gte=1"`
}

type OpenCloudUpdateOwnerStorageRequest struct {
	SubscriptionID  string `json:"subscriptionID" binding:"required" validate:"required"`
	StorageQuantity int64  `json:"storageQuantity" binding:"required,min=1" validate:"gte=1"`
}

type OpenCloudUpdateUserStorageRequest struct {
	SubscriptionID  string `json:"subscriptionID" binding:"required" validate:"required"`
	UID             string `json:"uid" binding:"required" validate:"required"`
	StorageQuantity int64  `json:"storageQuantity" binding:"required,min=1" validate:"gte=1"`
}

type OpenCloudUpdateUserRolesRequest struct {
	SubscriptionID string                `json:"subscriptionID" binding:"required" validate:"required"`
	UID            string                `json:"uid" binding:"required" validate:"required"`
	Roles          []mongodbmanager.Role `json:"roles" binding:"required,min=1" validate:"required,min=1"`
}

type OpenCloudCreateManagedSpaceRequest struct {
	SubscriptionID  string `json:"subscriptionID" binding:"required" validate:"required"`
	Name            string `json:"name" binding:"required" validate:"required,max=80"`
	StorageQuantity int64  `json:"storageQuantity" binding:"required,min=1" validate:"gte=1"`
}

type OpenCloudUpdateSpaceStorageRequest struct {
	SubscriptionID  string `json:"subscriptionID" binding:"required" validate:"required"`
	SpaceID         string `json:"spaceID" binding:"required" validate:"required"`
	StorageQuantity int64  `json:"storageQuantity" binding:"required,min=1" validate:"gte=1"`
}

type OpenCloudAddSpaceMemberRequest struct {
	SubscriptionID string `json:"subscriptionID" binding:"required" validate:"required"`
	SpaceID        string `json:"spaceID" binding:"required" validate:"required"`
	UID            string `json:"uid" binding:"required" validate:"required"`
}

type OpenCloudRemoveSpaceMemberRequest struct {
	SubscriptionID string `json:"subscriptionID" binding:"required" validate:"required"`
	SpaceID        string `json:"spaceID" binding:"required" validate:"required"`
	PermissionID   string `json:"permissionID" binding:"required" validate:"required"`
}

type OpenCloudAllocationMutationReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
}

type OpenCloudManagedSpaceReply struct {
	SubscriptionID string                  `json:"subscriptionID" validate:"required"`
	Space          *opencloudmanager.Space `json:"space" validate:"required"`
}

type openCloudSubscriptionCapacity struct {
	quantity      int64
	storageAmount int64
	storageUnit   string
	unitBytes     int64
	totalBytes    int64
}

func (paymentHandler *PaymentHandler) GetOpenCloudAllocationOverview(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudSubscriptionRequest) (OpenCloudAllocationOverviewReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationOverviewReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationOverviewReply{}, err
	}
	capacity, err := paymentHandler.openCloudSubscriptionCapacity(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationOverviewReply{}, err
	}
	owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationOverviewReply{}, err
	}
	if owner.SubscriptionID == "" {
		return OpenCloudAllocationOverviewReply{}, errors.New("subscription owner seat not found")
	}
	seats, err := paymentHandler.MongoDBConnection.GetAllSeats(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationOverviewReply{}, err
	}
	resources, err := paymentHandler.MongoDBConnection.GetOpenCloudResourcesBySubscription(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationOverviewReply{}, err
	}

	reply := OpenCloudAllocationOverviewReply{
		SubscriptionID: request.SubscriptionID,
		TotalQuantity:  capacity.quantity,
		StorageAmount:  capacity.storageAmount,
		StorageUnit:    capacity.storageUnit,
		Owner: OpenCloudOwnerAllocation{
			UID:             owner.UID,
			Email:           owner.EMail,
			OpenCloudLinked: owner.OpenCloudUserID != "",
			StorageQuantity: owner.OpenCloudStorageBytes / capacity.unitBytes,
			DailyUsage:      []OpenCloudAllocationUsagePoint{},
		},
		Users:  make([]OpenCloudUserAllocation, 0, len(seats)),
		Spaces: make([]OpenCloudManagedSpace, 0),
	}
	uidByOpenCloudUserID := make(map[string]string, len(seats)+1)
	emailByOpenCloudUserID := make(map[string]string, len(seats)+1)
	if owner.OpenCloudUserID != "" {
		uid := owner.UID
		if uid == "" {
			uid = owner.EMail
		}
		uidByOpenCloudUserID[owner.OpenCloudUserID] = uid
		emailByOpenCloudUserID[owner.OpenCloudUserID] = owner.EMail
	}
	for _, seat := range seats {
		if seat.OpenCloudUserID == "" || (seat.Status != "" && seat.Status != mongodbmanager.SeatStatusActive) {
			continue
		}
		uid := seat.UID
		if uid == "" {
			uid = seat.EMail
		}
		uidByOpenCloudUserID[seat.OpenCloudUserID] = uid
		emailByOpenCloudUserID[seat.OpenCloudUserID] = seat.EMail
	}
	if owner.OpenCloudUserID != "" {
		reply.Owner.UsedQuantity, reply.Owner.DailyUsage, err = paymentHandler.openCloudDailyUsage(ctx, owner.OpenCloudUserID, capacity.unitBytes)
		if err != nil {
			return OpenCloudAllocationOverviewReply{}, err
		}
	}
	for _, seat := range seats {
		if mongodbmanager.ContainsRole(seat, []mongodbmanager.Role{mongodbmanager.RoleOwner}) {
			continue
		}
		if seat.Status != "" && seat.Status != mongodbmanager.SeatStatusActive && seat.Status != mongodbmanager.SeatStatusPending && seat.Status != mongodbmanager.SeatStatusClaiming {
			continue
		}
		uid := seat.UID
		if uid == "" {
			uid = seat.EMail
		}
		user := OpenCloudUserAllocation{
			UID:             uid,
			Email:           seat.EMail,
			Roles:           seat.Roles,
			OpenCloudLinked: seat.OpenCloudUserID != "" && seat.Status == mongodbmanager.SeatStatusActive,
			StorageQuantity: seat.OpenCloudStorageBytes / capacity.unitBytes,
			DailyUsage:      []OpenCloudAllocationUsagePoint{},
		}
		if seat.OpenCloudUserID != "" {
			user.UsedQuantity, user.DailyUsage, err = paymentHandler.openCloudDailyUsage(ctx, seat.OpenCloudUserID, capacity.unitBytes)
			if err != nil {
				return OpenCloudAllocationOverviewReply{}, err
			}
		}
		reply.Users = append(reply.Users, user)
	}
	for _, resource := range resources {
		if resource.ResourceType != "space" || resource.Scope != openCloudAllocatedSpaceScope {
			continue
		}
		space := OpenCloudManagedSpace{
			ID:              resource.ResourceID,
			Name:            resource.Name,
			StorageQuantity: resource.StorageQuotaBytes / capacity.unitBytes,
			Members:         make([]OpenCloudManagedSpaceMember, 0),
			DailyUsage:      []OpenCloudAllocationUsagePoint{},
		}
		space.UsedQuantity, space.DailyUsage, err = paymentHandler.openCloudDailyUsage(ctx, resource.ResourceID, capacity.unitBytes)
		if err != nil {
			return OpenCloudAllocationOverviewReply{}, err
		}
		members, err := paymentHandler.OpenCloudConnection.ListMembers(ctx, resource.ResourceID)
		if err != nil {
			return OpenCloudAllocationOverviewReply{}, err
		}
		space.MemberCount = int64(len(members))
		for _, member := range members {
			spaceMember := OpenCloudManagedSpaceMember{
				PermissionID: member.PermissionID,
				UserID:       member.UserID,
				GroupID:      member.GroupID,
				DisplayName:  member.DisplayName,
			}
			if member.UserID != "" {
				spaceMember.UID = uidByOpenCloudUserID[member.UserID]
				spaceMember.Email = emailByOpenCloudUserID[member.UserID]
			}
			space.Members = append(space.Members, spaceMember)
		}
		reply.Spaces = append(reply.Spaces, space)
	}
	sort.Slice(reply.Users, func(i, j int) bool { return reply.Users[i].Email < reply.Users[j].Email })
	sort.Slice(reply.Spaces, func(i, j int) bool { return reply.Spaces[i].Name < reply.Spaces[j].Name })
	return reply, nil
}

func (paymentHandler *PaymentHandler) openCloudDailyUsage(ctx context.Context, driveID string, unitBytes int64) (float64, []OpenCloudAllocationUsagePoint, error) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	from := today.AddDate(0, 0, -29)
	records, err := paymentHandler.MongoDBConnection.GetOpenCloudUsageHistory(ctx, driveID, from, today.AddDate(0, 0, 1))
	if err != nil {
		return 0, nil, err
	}
	points := make([]OpenCloudAllocationUsagePoint, 0, len(records))
	var usedQuantity float64
	for _, record := range records {
		if record.UsedBytes == nil {
			continue
		}
		usedQuantity = float64(*record.UsedBytes) / float64(unitBytes)
		points = append(points, OpenCloudAllocationUsagePoint{
			Date:         record.Day.Format("2006-01-02"),
			UsedQuantity: usedQuantity,
		})
	}
	return usedQuantity, points, nil
}

func (paymentHandler *PaymentHandler) InviteOpenCloudUser(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudInviteUserRequest) (AddSeatReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return AddSeatReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return AddSeatReply{}, err
	}
	parsedEmail, err := mail.ParseAddress(request.Email)
	if err != nil || !strings.EqualFold(parsedEmail.Address, request.Email) {
		return AddSeatReply{}, errors.New("E-Mail is invalid")
	}
	capacity, err := paymentHandler.openCloudSubscriptionCapacity(ctx, request.SubscriptionID)
	if err != nil {
		return AddSeatReply{}, err
	}
	storageBytes, err := openCloudQuantityBytes(request.StorageQuantity, capacity.unitBytes)
	if err != nil {
		return AddSeatReply{}, err
	}
	return paymentHandler.CreateOpenCloudSeatInvitation(
		ctx,
		request.SubscriptionID,
		normalizeInvitationEmail(parsedEmail.Address),
		tokenDetails.EMail,
		[]mongodbmanager.Role{mongodbmanager.RoleUser},
		capacity.quantity,
		storageBytes,
		capacity.totalBytes,
	)
}

func (paymentHandler *PaymentHandler) UpdateOpenCloudOwnerStorage(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudUpdateOwnerStorageRequest) (OpenCloudAllocationMutationReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationMutationReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	capacity, err := paymentHandler.openCloudSubscriptionCapacity(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	newBytes, err := openCloudQuantityBytes(request.StorageQuantity, capacity.unitBytes)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	err = paymentHandler.MongoDBConnection.WithOpenCloudAllocationLock(ctx, request.SubscriptionID, func() error {
		owner, err := paymentHandler.MongoDBConnection.GetOwnerSeat(ctx, request.SubscriptionID)
		if err != nil {
			return err
		}
		if owner.SubscriptionID == "" {
			return errors.New("subscription owner seat not found")
		}
		usedBytes, err := paymentHandler.MongoDBConnection.GetOpenCloudStorageUsedBytes(ctx, request.SubscriptionID)
		if err != nil {
			return err
		}
		if err := validateOpenCloudReplacement(capacity.totalBytes, usedBytes, owner.OpenCloudStorageBytes, newBytes); err != nil {
			return err
		}
		newUserBytes, err := paymentHandler.openCloudUserQuotaAfterReplacement(ctx, owner, newBytes)
		if err != nil {
			return err
		}
		if err := paymentHandler.MongoDBConnection.SetOpenCloudOwnerStorageBytes(ctx, request.SubscriptionID, newBytes); err != nil {
			return err
		}
		if owner.OpenCloudUserID != "" {
			if _, err := paymentHandler.OpenCloudConnection.SetUserDriveQuota(ctx, owner.OpenCloudUserID, newUserBytes); err != nil {
				if rollbackErr := paymentHandler.MongoDBConnection.SetOpenCloudOwnerStorageBytes(ctx, request.SubscriptionID, owner.OpenCloudStorageBytes); rollbackErr != nil {
					return fmt.Errorf("apply OpenCloud personal quota: %v; restore stored allocation: %w", err, rollbackErr)
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	return OpenCloudAllocationMutationReply{SubscriptionID: request.SubscriptionID, Success: true}, nil
}

func (paymentHandler *PaymentHandler) UpdateOpenCloudUserStorage(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudUpdateUserStorageRequest) (OpenCloudAllocationMutationReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationMutationReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	capacity, err := paymentHandler.openCloudSubscriptionCapacity(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	newBytes, err := openCloudQuantityBytes(request.StorageQuantity, capacity.unitBytes)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	err = paymentHandler.MongoDBConnection.WithOpenCloudAllocationLock(ctx, request.SubscriptionID, func() error {
		seat, err := paymentHandler.getOpenCloudSeat(ctx, request.SubscriptionID, request.UID)
		if err != nil {
			return err
		}
		if mongodbmanager.ContainsRole(seat, []mongodbmanager.Role{mongodbmanager.RoleOwner}) {
			return errors.New("use owner storage allocation for the subscription owner")
		}
		usedBytes, err := paymentHandler.MongoDBConnection.GetOpenCloudStorageUsedBytes(ctx, request.SubscriptionID)
		if err != nil {
			return err
		}
		if err := validateOpenCloudReplacement(capacity.totalBytes, usedBytes, seat.OpenCloudStorageBytes, newBytes); err != nil {
			return err
		}
		newUserBytes, err := paymentHandler.openCloudUserQuotaAfterReplacement(ctx, seat, newBytes)
		if err != nil {
			return err
		}
		identity := request.UID
		if seat.UID == "" {
			identity = seat.EMail
		}
		if err := paymentHandler.MongoDBConnection.SetOpenCloudSeatStorageBytes(ctx, request.SubscriptionID, identity, newBytes); err != nil {
			return err
		}
		if seat.OpenCloudUserID != "" {
			if _, err := paymentHandler.OpenCloudConnection.SetUserDriveQuota(ctx, seat.OpenCloudUserID, newUserBytes); err != nil {
				if rollbackErr := paymentHandler.MongoDBConnection.SetOpenCloudSeatStorageBytes(ctx, request.SubscriptionID, identity, seat.OpenCloudStorageBytes); rollbackErr != nil {
					return fmt.Errorf("apply OpenCloud personal quota: %v; restore stored allocation: %w", err, rollbackErr)
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	return OpenCloudAllocationMutationReply{SubscriptionID: request.SubscriptionID, Success: true}, nil
}

func (paymentHandler *PaymentHandler) UpdateOpenCloudUserRoles(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudUpdateUserRolesRequest) (OpenCloudAllocationMutationReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationMutationReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	seat, err := paymentHandler.getOpenCloudSeat(ctx, request.SubscriptionID, request.UID)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	if mongodbmanager.ContainsRole(seat, []mongodbmanager.Role{mongodbmanager.RoleOwner}) {
		return OpenCloudAllocationMutationReply{}, errors.New("subscription owner roles cannot be changed here")
	}
	if err := validateOpenCloudUserRoles(request.Roles); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	if err := paymentHandler.MongoDBConnection.UpdateSeatRoles(ctx, request.SubscriptionID, request.UID, request.Roles); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	return OpenCloudAllocationMutationReply{SubscriptionID: request.SubscriptionID, Success: true}, nil
}

func validateOpenCloudUserRoles(roles []mongodbmanager.Role) error {
	if len(roles) == 0 {
		return errors.New("at least one user role is required")
	}
	for _, role := range roles {
		switch role {
		case mongodbmanager.RoleAdministrator, mongodbmanager.RoleUser, mongodbmanager.RoleBilling:
		default:
			return fmt.Errorf("invalid role for OpenCloud user: %q", role)
		}
	}
	return nil
}

func (paymentHandler *PaymentHandler) CreateOpenCloudManagedSpace(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudCreateManagedSpaceRequest) (OpenCloudManagedSpaceReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudManagedSpaceReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudManagedSpaceReply{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return OpenCloudManagedSpaceReply{}, errors.New("OpenCloud space name is required")
	}
	capacity, err := paymentHandler.openCloudSubscriptionCapacity(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudManagedSpaceReply{}, err
	}
	quotaBytes, err := openCloudQuantityBytes(request.StorageQuantity, capacity.unitBytes)
	if err != nil {
		return OpenCloudManagedSpaceReply{}, err
	}
	var createdSpace *opencloudmanager.Space
	err = paymentHandler.MongoDBConnection.WithOpenCloudAllocationLock(ctx, request.SubscriptionID, func() error {
		usedBytes, err := paymentHandler.MongoDBConnection.GetOpenCloudStorageUsedBytes(ctx, request.SubscriptionID)
		if err != nil {
			return err
		}
		if err := validateOpenCloudReplacement(capacity.totalBytes, usedBytes, 0, quotaBytes); err != nil {
			return err
		}
		marker, err := openCloudAllocationMarker(request.SubscriptionID)
		if err != nil {
			return err
		}
		createdSpace, err = paymentHandler.OpenCloudConnection.CreateSpace(ctx, name, marker, quotaBytes)
		if err != nil {
			return err
		}
		resource := mongodbmanager.OpenCloudResourceRegistry{
			SubscriptionID:    request.SubscriptionID,
			ResourceType:      "space",
			ResourceID:        createdSpace.ID,
			Scope:             openCloudAllocatedSpaceScope,
			DriveType:         "project",
			Name:              name,
			StorageQuotaBytes: quotaBytes,
			LastSeenAt:        time.Now().UTC(),
		}
		if err := paymentHandler.MongoDBConnection.UpsertOpenCloudResource(ctx, resource); err != nil {
			if cleanupErr := paymentHandler.OpenCloudConnection.DisableSpace(ctx, createdSpace.ID); cleanupErr != nil {
				paymentHandler.Log.Error("Could not clean up OpenCloud space after storage registration failed", zap.Error(cleanupErr))
			}
			return err
		}
		return nil
	})
	if err != nil {
		return OpenCloudManagedSpaceReply{}, err
	}
	return OpenCloudManagedSpaceReply{SubscriptionID: request.SubscriptionID, Space: createdSpace}, nil
}

func (paymentHandler *PaymentHandler) UpdateOpenCloudSpaceStorage(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudUpdateSpaceStorageRequest) (OpenCloudAllocationMutationReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationMutationReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	capacity, err := paymentHandler.openCloudSubscriptionCapacity(ctx, request.SubscriptionID)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	newBytes, err := openCloudQuantityBytes(request.StorageQuantity, capacity.unitBytes)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	err = paymentHandler.MongoDBConnection.WithOpenCloudAllocationLock(ctx, request.SubscriptionID, func() error {
		allocation, err := paymentHandler.MongoDBConnection.GetOpenCloudSpaceAllocation(ctx, request.SubscriptionID, request.SpaceID)
		if err != nil {
			return err
		}
		usedBytes, err := paymentHandler.MongoDBConnection.GetOpenCloudStorageUsedBytes(ctx, request.SubscriptionID)
		if err != nil {
			return err
		}
		if err := validateOpenCloudReplacement(capacity.totalBytes, usedBytes, allocation.StorageQuotaBytes, newBytes); err != nil {
			return err
		}
		usage, err := paymentHandler.OpenCloudConnection.GetSpaceUsage(ctx, request.SpaceID)
		if err != nil {
			return err
		}
		if usage.Used != nil && *usage.Used > newBytes {
			return errors.New("OpenCloud space is using more storage than the requested quota")
		}
		oldQuotaBytes := allocation.StorageQuotaBytes
		if _, err := paymentHandler.OpenCloudConnection.SetSpaceQuota(ctx, request.SpaceID, newBytes); err != nil {
			return err
		}
		allocation.StorageQuotaBytes = newBytes
		allocation.LastSeenAt = time.Now().UTC()
		if err := paymentHandler.MongoDBConnection.UpsertOpenCloudResource(ctx, allocation); err != nil {
			if _, rollbackErr := paymentHandler.OpenCloudConnection.SetSpaceQuota(ctx, request.SpaceID, oldQuotaBytes); rollbackErr != nil {
				return fmt.Errorf("store OpenCloud space allocation: %v; restore OpenCloud quota: %w", err, rollbackErr)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	return OpenCloudAllocationMutationReply{SubscriptionID: request.SubscriptionID, Success: true}, nil
}

func (paymentHandler *PaymentHandler) AddOpenCloudSpaceMember(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudAddSpaceMemberRequest) (OpenCloudAllocationMutationReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationMutationReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	if _, err := paymentHandler.MongoDBConnection.GetOpenCloudSpaceAllocation(ctx, request.SubscriptionID, request.SpaceID); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	seat, err := paymentHandler.getOpenCloudSeat(ctx, request.SubscriptionID, request.UID)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	if seat.SubscriptionID == "" || (seat.Status != "" && seat.Status != mongodbmanager.SeatStatusActive) || seat.OpenCloudUserID == "" {
		return OpenCloudAllocationMutationReply{}, errors.New("an active linked subscription member is required")
	}
	if _, err := paymentHandler.OpenCloudConnection.EnsureMember(ctx, request.SpaceID, seat.OpenCloudUserID, opencloudmanager.SpaceRoleEditor); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	return OpenCloudAllocationMutationReply{SubscriptionID: request.SubscriptionID, Success: true}, nil
}

func (paymentHandler *PaymentHandler) RemoveOpenCloudSpaceMember(ctx context.Context, tokenDetails authmanager.TokenDetails, request OpenCloudRemoveSpaceMemberRequest) (OpenCloudAllocationMutationReply, error) {
	if paymentHandler.OpenCloudConnection == nil {
		return OpenCloudAllocationMutationReply{}, ErrOpenCloudUnavailable
	}
	if err := paymentHandler.MongoDBConnection.HasPermission(ctx, tokenDetails, request.SubscriptionID, []mongodbmanager.Role{mongodbmanager.RoleAdministrator}); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	if _, err := paymentHandler.MongoDBConnection.GetOpenCloudSpaceAllocation(ctx, request.SubscriptionID, request.SpaceID); err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	members, err := paymentHandler.OpenCloudConnection.ListMembers(ctx, request.SpaceID)
	if err != nil {
		return OpenCloudAllocationMutationReply{}, err
	}
	for _, member := range members {
		if member.PermissionID == request.PermissionID {
			if err := paymentHandler.OpenCloudConnection.RemoveMember(ctx, request.SpaceID, request.PermissionID); err != nil {
				return OpenCloudAllocationMutationReply{}, err
			}
			return OpenCloudAllocationMutationReply{SubscriptionID: request.SubscriptionID, Success: true}, nil
		}
	}
	return OpenCloudAllocationMutationReply{}, errors.New("OpenCloud member not found in space")
}

func (paymentHandler *PaymentHandler) openCloudUserQuotaAfterReplacement(ctx context.Context, seat mongodbmanager.Seat, newBytes int64) (int64, error) {
	if seat.OpenCloudUserID == "" {
		return 0, nil
	}
	currentBytes, err := paymentHandler.MongoDBConnection.GetOpenCloudUserStorageBytes(ctx, seat.OpenCloudUserID)
	if err != nil {
		return 0, err
	}
	if currentBytes < seat.OpenCloudStorageBytes || newBytes > math.MaxInt64-(currentBytes-seat.OpenCloudStorageBytes) {
		return 0, errors.New("invalid aggregate OpenCloud user allocation")
	}
	quotaBytes := currentBytes - seat.OpenCloudStorageBytes + newBytes
	usage, err := paymentHandler.OpenCloudConnection.GetUserDriveUsage(ctx, seat.OpenCloudUserID)
	if err != nil {
		return 0, err
	}
	if usage.Used != nil && *usage.Used > quotaBytes {
		return 0, errors.New("OpenCloud personal drive is using more storage than the requested quota")
	}
	return quotaBytes, nil
}

func (paymentHandler *PaymentHandler) getOpenCloudSeat(ctx context.Context, subscriptionID, uidOrEmail string) (mongodbmanager.Seat, error) {
	if strings.Contains(uidOrEmail, "@") {
		seat, err := paymentHandler.MongoDBConnection.GetSeatByEmail(ctx, subscriptionID, normalizeInvitationEmail(uidOrEmail))
		if err != nil {
			return mongodbmanager.Seat{}, err
		}
		if seat.SubscriptionID == "" {
			return mongodbmanager.Seat{}, errors.New("subscription seat not found")
		}
		return seat, nil
	}
	seat, err := paymentHandler.MongoDBConnection.GetSeat(ctx, subscriptionID, uidOrEmail)
	if err != nil {
		return mongodbmanager.Seat{}, err
	}
	if seat.SubscriptionID == "" {
		return mongodbmanager.Seat{}, errors.New("subscription seat not found")
	}
	return seat, nil
}

func (paymentHandler *PaymentHandler) openCloudSubscriptionCapacity(ctx context.Context, subscriptionID string) (openCloudSubscriptionCapacity, error) {
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(ctx, subscriptionID)
	if err != nil || subscription == nil || len(subscription.Items.Data) == 0 || subscription.Items.Data[0].Price == nil || subscription.Items.Data[0].Price.Product == nil {
		return openCloudSubscriptionCapacity{}, errors.New("OpenCloud subscription not found")
	}
	if subscription.Status != stripe.SubscriptionStatusActive && subscription.Status != stripe.SubscriptionStatusTrialing {
		return openCloudSubscriptionCapacity{}, ErrInvitationSubscription
	}
	product, err := paymentHandler.StripeConnection.GetProduct(ctx, subscription.Items.Data[0].Price.Product.ID)
	if err != nil {
		return openCloudSubscriptionCapacity{}, err
	}
	if product.Metadata["productType"] != "OpenCloud" {
		return openCloudSubscriptionCapacity{}, errors.New("subscription is not an OpenCloud product")
	}
	storageAmount, err := parsePositiveInt64(product.Metadata["storageAmount"])
	if err != nil {
		return openCloudSubscriptionCapacity{}, errors.New("OpenCloud storage amount is not configured")
	}
	storageUnit := strings.TrimSpace(product.Metadata["storageUnit"])
	unitBytes, err := storageQuotaBytes(storageAmount, storageUnit)
	if err != nil {
		return openCloudSubscriptionCapacity{}, err
	}
	quantity := subscription.Items.Data[0].Quantity
	if quantity < 1 || quantity > math.MaxInt64/unitBytes {
		return openCloudSubscriptionCapacity{}, errors.New("OpenCloud subscription quantity is invalid")
	}
	return openCloudSubscriptionCapacity{
		quantity:      quantity,
		storageAmount: storageAmount,
		storageUnit:   storageUnit,
		unitBytes:     unitBytes,
		totalBytes:    quantity * unitBytes,
	}, nil
}

func parsePositiveInt64(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed < 1 {
		return 0, errors.New("value must be a positive integer")
	}
	return parsed, nil
}

func openCloudQuantityBytes(quantity, unitBytes int64) (int64, error) {
	if quantity < 1 || unitBytes < 1 || quantity > math.MaxInt64/unitBytes {
		return 0, errors.New("OpenCloud storage quantity is invalid")
	}
	return quantity * unitBytes, nil
}

func validateOpenCloudReplacement(capacityBytes, usedBytes, replacedBytes, newBytes int64) error {
	if capacityBytes < 1 || usedBytes < replacedBytes || newBytes < 1 || usedBytes-replacedBytes > capacityBytes || newBytes > capacityBytes-(usedBytes-replacedBytes) {
		return mongodbmanager.ErrOpenCloudStorageCapacityReached
	}
	return nil
}

func openCloudAllocationMarker(subscriptionID string) (string, error) {
	randomBytes := make([]byte, 12)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("scalecloud-subscription:%s:opencloud-space:%s", subscriptionID, hex.EncodeToString(randomBytes)), nil
}
