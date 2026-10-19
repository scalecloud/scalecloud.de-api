package stripemanager

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/scalecloud/scalecloud.de-api/authentikmanager"
	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/emailmanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v87"
	"go.uber.org/zap"
)

var (
	ErrInvitationUnavailable     = errors.New("invitation is invalid, expired, or already claimed")
	ErrInvitationEmailMismatch   = errors.New("signed-in email does not match the invitation")
	ErrInvitationEmailUnverified = errors.New("signed-in email is not verified")
	ErrInvitationSubscription    = errors.New("invitation subscription is unavailable")
)

const seatInvitationLifetime = 7 * 24 * time.Hour

type invitationClaimStore interface {
	GetSeatByInvitationTokenHash(context.Context, string) (mongomanager.Seat, error)
	BeginSeatClaim(context.Context, string, string, string, int64, time.Time) (mongomanager.Seat, error)
	CompleteSeatClaim(context.Context, string, string) error
	GetSeat(context.Context, string, string) (mongomanager.Seat, error)
}

type invitationClaimAuthentik interface {
	FindUserByEmail(context.Context, string) (authentikmanager.User, error)
	AddUserPKToSubscription(context.Context, string, int64) error
	RemoveUserFromSubscription(context.Context, string, int64) error
}

type invitationClaimStripe interface {
	GetSubscriptionByID(context.Context, string) (*stripe.Subscription, error)
	GetProduct(context.Context, string) (*stripe.Product, error)
}

type invitationClaimDependencies struct {
	store     invitationClaimStore
	authentik invitationClaimAuthentik
	stripe    invitationClaimStripe
	logger    *zap.Logger
}

type invitationCreateStore interface {
	CreatePendingSeatWithinCapacity(context.Context, mongomanager.Seat, int64) error
	DeletePendingSeatByInvitationTokenHash(context.Context, string) error
}

type invitationCreateStripe interface {
	GetSubscriptionByID(context.Context, string) (*stripe.Subscription, error)
	GetProduct(context.Context, string) (*stripe.Product, error)
}

type invitationCreateMailer interface {
	SendEMail(emailmanager.EMail) error
}

type invitationCreateDependencies struct {
	store    invitationCreateStore
	stripe   invitationCreateStripe
	mailer   invitationCreateMailer
	claimURL string
	logger   *zap.Logger
}

func (paymentHandler *PaymentHandler) CreateSeatInvitation(ctx context.Context, subscriptionID, email string, roles []mongomanager.Role, maxSeats int64) (AddSeatReply, error) {
	return createSeatInvitation(ctx, invitationCreateDependencies{
		store:    paymentHandler.MongoConnection,
		stripe:   paymentHandler.StripeConnection,
		mailer:   paymentHandler.EMailConnection,
		claimURL: paymentHandler.InvitationClaimURL,
		logger:   paymentHandler.Log,
	}, subscriptionID, email, roles, maxSeats)
}

func createSeatInvitation(ctx context.Context, dependencies invitationCreateDependencies, subscriptionID, email string, roles []mongomanager.Role, maxSeats int64) (AddSeatReply, error) {
	token, tokenHash, err := createSeatInvitationToken()
	if err != nil {
		return AddSeatReply{}, err
	}
	expiresAt := time.Now().UTC().Add(seatInvitationLifetime)
	seat := mongomanager.Seat{
		SubscriptionID:      subscriptionID,
		EMail:               normalizeInvitationEmail(email),
		Roles:               roles,
		Status:              mongomanager.SeatStatusPending,
		InvitationTokenHash: tokenHash,
		InvitationExpiresAt: &expiresAt,
	}
	if err := dependencies.store.CreatePendingSeatWithinCapacity(ctx, seat, maxSeats); err != nil {
		return AddSeatReply{}, err
	}

	claimURL := dependencies.claimURL + "?token=" + url.QueryEscape(token)
	productName := "your subscription"
	if subscription, err := dependencies.stripe.GetSubscriptionByID(ctx, subscriptionID); err == nil && len(subscription.Items.Data) > 0 && subscription.Items.Data[0].Price != nil && subscription.Items.Data[0].Price.Product != nil {
		if product, err := dependencies.stripe.GetProduct(ctx, subscription.Items.Data[0].Price.Product.ID); err == nil && product.Name != "" {
			productName = product.Name
		}
	}
	body := fmt.Sprintf("<p>You have been invited to join %s on Scalecloud.</p><p><a href=\"%s\">Review and claim your seat</a></p><p>This invitation expires in seven days.</p>", html.EscapeString(productName), html.EscapeString(claimURL))
	if err := dependencies.mailer.SendEMail(emailmanager.EMail{
		To:      []string{email},
		Subject: "You have been invited to Scalecloud",
		Body:    body,
	}); err != nil {
		if cleanupErr := dependencies.store.DeletePendingSeatByInvitationTokenHash(ctx, tokenHash); cleanupErr != nil {
			dependencies.logger.Error("Could not remove pending seat after invitation email failed", zap.Error(cleanupErr))
		}
		return AddSeatReply{}, err
	}

	return AddSeatReply{SubscriptionID: subscriptionID, Success: true, EMail: email}, nil
}

func (paymentHandler *PaymentHandler) PreviewSeatInvitation(ctx context.Context, token string) (SeatInvitationPreviewReply, error) {
	if len(token) < 40 || len(token) > 128 {
		return SeatInvitationPreviewReply{}, ErrInvitationUnavailable
	}
	seat, err := paymentHandler.MongoConnection.GetPendingSeatByInvitationTokenHash(ctx, hashSeatInvitationToken(token), time.Now().UTC())
	if err != nil {
		return SeatInvitationPreviewReply{}, ErrInvitationUnavailable
	}
	productName, storageAmount, storageUnit, err := getSeatProductDetails(ctx, paymentHandler.StripeConnection, seat.SubscriptionID)
	if err != nil {
		if errors.Is(err, ErrInvitationSubscription) {
			return SeatInvitationPreviewReply{}, ErrInvitationUnavailable
		}
		return SeatInvitationPreviewReply{}, err
	}
	return SeatInvitationPreviewReply{
		ProductName:   productName,
		StorageAmount: storageAmount,
		StorageUnit:   storageUnit,
		EMail:         seat.EMail,
		Roles:         seat.Roles,
		ExpiresAt:     seat.InvitationExpiresAt.Unix(),
	}, nil
}

func (paymentHandler *PaymentHandler) ClaimSeatInvitation(ctx context.Context, token string, tokenDetails authmanager.TokenDetails) (SeatInvitationClaimReply, error) {
	return claimSeatInvitation(ctx, token, tokenDetails, invitationClaimDependencies{
		store:     paymentHandler.MongoConnection,
		authentik: paymentHandler.AuthentikConnection,
		stripe:    paymentHandler.StripeConnection,
		logger:    paymentHandler.Log,
	})
}

func claimSeatInvitation(ctx context.Context, token string, tokenDetails authmanager.TokenDetails, dependencies invitationClaimDependencies) (SeatInvitationClaimReply, error) {
	if !tokenDetails.EmailVerified {
		return SeatInvitationClaimReply{}, ErrInvitationEmailUnverified
	}
	if tokenDetails.UID == "" || tokenDetails.EMail == "" || len(token) < 40 || len(token) > 128 {
		return SeatInvitationClaimReply{}, ErrInvitationUnavailable
	}
	email := normalizeInvitationEmail(tokenDetails.EMail)
	tokenHash := hashSeatInvitationToken(token)
	seat, err := dependencies.store.GetSeatByInvitationTokenHash(ctx, tokenHash)
	if err != nil {
		return SeatInvitationClaimReply{}, ErrInvitationUnavailable
	}
	if normalizeInvitationEmail(seat.EMail) != email {
		return SeatInvitationClaimReply{}, ErrInvitationEmailMismatch
	}
	productName, storageAmount, storageUnit, err := getSeatProductDetails(ctx, dependencies.stripe, seat.SubscriptionID)
	if err != nil {
		if errors.Is(err, ErrInvitationSubscription) {
			return SeatInvitationClaimReply{}, ErrInvitationUnavailable
		}
		return SeatInvitationClaimReply{}, err
	}
	switch seat.Status {
	case mongomanager.SeatStatusPending:
		if seat.InvitationExpiresAt == nil || !seat.InvitationExpiresAt.After(time.Now().UTC()) {
			return SeatInvitationClaimReply{}, ErrInvitationUnavailable
		}
		user, err := dependencies.authentik.FindUserByEmail(ctx, email)
		if err != nil {
			return SeatInvitationClaimReply{}, err
		}
		if !user.IsActive {
			return SeatInvitationClaimReply{}, errors.New("authentik user is inactive")
		}
		seat, err = dependencies.store.BeginSeatClaim(ctx, tokenHash, email, tokenDetails.UID, user.PK, time.Now().UTC())
		if err != nil {
			claimingSeat, lookupErr := dependencies.store.GetSeatByInvitationTokenHash(ctx, tokenHash)
			if lookupErr != nil || claimingSeat.Status != mongomanager.SeatStatusClaiming || claimingSeat.UID != tokenDetails.UID {
				return SeatInvitationClaimReply{}, ErrInvitationUnavailable
			}
			seat = claimingSeat
		}
	case mongomanager.SeatStatusClaiming:
		if seat.UID != tokenDetails.UID || seat.AuthentikUserPK == 0 {
			return SeatInvitationClaimReply{}, ErrInvitationUnavailable
		}
	default:
		return SeatInvitationClaimReply{}, ErrInvitationUnavailable
	}

	if err := dependencies.authentik.AddUserPKToSubscription(ctx, seat.SubscriptionID, seat.AuthentikUserPK); err != nil {
		if rollbackErr := dependencies.authentik.RemoveUserFromSubscription(ctx, seat.SubscriptionID, seat.AuthentikUserPK); rollbackErr != nil {
			dependencies.logger.Error("Could not roll back Authentik group membership after invite claim failed", zap.Error(rollbackErr))
		}
		return SeatInvitationClaimReply{}, err
	}
	if err := dependencies.store.CompleteSeatClaim(ctx, tokenHash, seat.UID); err != nil {
		claimedSeat, lookupErr := dependencies.store.GetSeat(ctx, seat.SubscriptionID, seat.UID)
		if lookupErr == nil && claimedSeat.Status == mongomanager.SeatStatusActive && claimedSeat.AuthentikUserPK == seat.AuthentikUserPK {
			seat = claimedSeat
		} else {
			if rollbackErr := dependencies.authentik.RemoveUserFromSubscription(ctx, seat.SubscriptionID, seat.AuthentikUserPK); rollbackErr != nil {
				dependencies.logger.Error("Could not roll back Authentik group membership after seat activation failed", zap.Error(rollbackErr))
			}
			return SeatInvitationClaimReply{}, err
		}
	} else {
		seat.Status = mongomanager.SeatStatusActive
	}

	return SeatInvitationClaimReply{
		SubscriptionID: seat.SubscriptionID,
		ProductName:    productName,
		StorageAmount:  storageAmount,
		StorageUnit:    storageUnit,
		Roles:          seat.Roles,
		Success:        true,
	}, nil
}

func (paymentHandler *PaymentHandler) RemoveSeatFromSubscriptionGroup(ctx context.Context, seat mongomanager.Seat) error {
	if seat.Status == mongomanager.SeatStatusPending {
		return nil
	}
	userPK, err := paymentHandler.getAuthentikUserPK(ctx, seat)
	if errors.Is(err, authentikmanager.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return paymentHandler.AuthentikConnection.RemoveUserFromSubscription(ctx, seat.SubscriptionID, userPK)
}

func (paymentHandler *PaymentHandler) AddSeatToSubscriptionGroup(ctx context.Context, seat mongomanager.Seat) error {
	userPK, err := paymentHandler.getAuthentikUserPK(ctx, seat)
	if errors.Is(err, authentikmanager.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return paymentHandler.AuthentikConnection.AddUserPKToSubscription(ctx, seat.SubscriptionID, userPK)
}

func (paymentHandler *PaymentHandler) getAuthentikUserPK(ctx context.Context, seat mongomanager.Seat) (int64, error) {
	if seat.AuthentikUserPK != 0 {
		return seat.AuthentikUserPK, nil
	}
	user, err := paymentHandler.AuthentikConnection.FindUserByEmail(ctx, seat.EMail)
	if err != nil {
		return 0, err
	}
	return user.PK, nil
}

func getSeatProductDetails(ctx context.Context, stripeConnection invitationClaimStripe, subscriptionID string) (string, int64, string, error) {
	stripeSubscription, err := stripeConnection.GetSubscriptionByID(ctx, subscriptionID)
	if err != nil || len(stripeSubscription.Items.Data) == 0 || stripeSubscription.Items.Data[0].Price == nil || stripeSubscription.Items.Data[0].Price.Product == nil {
		return "", 0, "", errors.New("subscription product not found")
	}
	if stripeSubscription.Status != stripe.SubscriptionStatusActive && stripeSubscription.Status != stripe.SubscriptionStatusTrialing {
		return "", 0, "", ErrInvitationSubscription
	}
	product, err := stripeConnection.GetProduct(ctx, stripeSubscription.Items.Data[0].Price.Product.ID)
	if err != nil {
		return "", 0, "", err
	}
	storageAmount, err := strconv.ParseInt(product.Metadata["storageAmount"], 10, 64)
	if err != nil {
		return "", 0, "", errors.New("subscription storage amount not found")
	}
	storageUnit := product.Metadata["storageUnit"]
	if product.Name == "" || storageUnit == "" {
		return "", 0, "", errors.New("subscription product details are incomplete")
	}
	return product.Name, storageAmount, storageUnit, nil
}

func createSeatInvitationToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, hashSeatInvitationToken(token), nil
}

func hashSeatInvitationToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func normalizeInvitationEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
