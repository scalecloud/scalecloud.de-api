package stripemanager

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/customer"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func (paymentHandler *PaymentHandler) GetCustomerByUID(ctx context.Context, uid string) (customerDetails *stripe.Customer, err error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	return GetCustomerByID(ctx, customerID)
}

func GetCustomerByID(ctx context.Context, customerID string) (customerDetails *stripe.Customer, err error) {
	customer, error := customer.Get(
		customerID,
		nil,
	)
	if error != nil {
		return &stripe.Customer{}, errors.New("customer not found")
	}
	return customer, nil
}

func (paymentHandler *PaymentHandler) existsCustomerByUID(ctx context.Context, uid string) (bool, error) {
	filter := mongomanager.User{
		UID: uid,
	}
	userSearch, err := paymentHandler.MongoConnection.GetUser(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}
	customerID := userSearch.CustomerID
	if customerID == "" {
		return false, errors.New("customer ID is empty")
	}
	return true, nil
}

func (paymentHandler *PaymentHandler) GetCustomerIDByUID(ctx context.Context, uid string) (string, error) {
	filter := mongomanager.User{
		UID: uid,
	}
	userSearch, err := paymentHandler.MongoConnection.GetUser(ctx, filter)
	if err != nil {
		return "", err
	}
	customerID := userSearch.CustomerID
	if customerID == "" {
		return "", errors.New("customer ID is empty")
	}
	return customerID, nil
}

func (stripeConnection *StripeConnection) CreateCustomer(ctx context.Context, email, uid string) (*stripe.Customer, error) {
	stripe.Key = stripeConnection.Key
	if email == "" {
		return nil, errors.New("email is required")
	}
	if uid == "" {
		return nil, errors.New("user UID is required")
	}
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Metadata: map[string]string{
			"authentik_subject": uid,
		},
	}
	params.Context = ctx
	params.SetIdempotencyKey(stripeCustomerIdempotencyKey(uid))
	newCustomer, err := customer.New(params)
	if err != nil {
		return nil, err
	}
	return newCustomer, nil
}

func stripeCustomerIdempotencyKey(uid string) string {
	hash := sha256.Sum256([]byte(uid))
	return "scalecloud-customer-" + hex.EncodeToString(hash[:])
}

func (paymentHandler *PaymentHandler) getOrCreateCustomer(ctx context.Context, email, uid string) (string, error) {
	return getOrCreateStripeCustomer(ctx, paymentHandler.MongoConnection, paymentHandler.StripeConnection, email, uid)
}

type stripeCustomerMappingStore interface {
	GetUser(context.Context, mongomanager.User) (mongomanager.User, error)
	EnsureUser(context.Context, string) error
	ClaimUserProvisioning(context.Context, string, string, time.Time, time.Time) (bool, error)
	SetUserCustomerID(context.Context, string, string, string) error
	ReleaseUserProvisioning(context.Context, string, string) error
}

type stripeCustomerCreator interface {
	CreateCustomer(context.Context, string, string) (*stripe.Customer, error)
}

var ErrCustomerProvisioningInProgress = errors.New("Stripe customer provisioning is already in progress")

const customerProvisioningLeaseDuration = 2 * time.Minute
const customerProvisioningWaitDuration = 5 * time.Second
const customerProvisioningPollInterval = 25 * time.Millisecond

func getOrCreateStripeCustomer(ctx context.Context, store stripeCustomerMappingStore, stripeCreator stripeCustomerCreator, email, uid string) (string, error) {
	if uid == "" {
		return "", errors.New("user UID is required")
	}

	user, err := store.GetUser(ctx, mongomanager.User{UID: uid})
	if err == nil && user.CustomerID != "" {
		return user.CustomerID, nil
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return "", fmt.Errorf("look up Stripe customer mapping: %w", err)
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		if err := store.EnsureUser(ctx, uid); err != nil {
			return "", fmt.Errorf("create Stripe customer placeholder: %w", err)
		}
	}

	waitTimer := time.NewTimer(customerProvisioningWaitDuration)
	defer waitTimer.Stop()
	ticker := time.NewTicker(customerProvisioningPollInterval)
	defer ticker.Stop()
	for {
		user, err = store.GetUser(ctx, mongomanager.User{UID: uid})
		if err != nil {
			return "", fmt.Errorf("look up Stripe customer mapping: %w", err)
		}
		if user.CustomerID != "" {
			return user.CustomerID, nil
		}

		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			return "", fmt.Errorf("create customer provisioning token: %w", err)
		}
		token := hex.EncodeToString(tokenBytes)
		now := time.Now().UTC()
		claimed, err := store.ClaimUserProvisioning(ctx, uid, token, now, now.Add(customerProvisioningLeaseDuration))
		if err != nil {
			return "", fmt.Errorf("claim customer provisioning lease: %w", err)
		}
		if claimed {
			return createStripeCustomerUnderLease(ctx, store, stripeCreator, email, uid, token)
		}

		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", ErrCustomerProvisioningInProgress
		case <-waitTimer.C:
			return "", ErrCustomerProvisioningInProgress
		case <-ticker.C:
		}
	}
}

func createStripeCustomerUnderLease(ctx context.Context, store stripeCustomerMappingStore, stripeCreator stripeCustomerCreator, email, uid, token string) (string, error) {
	stripeCustomer, err := stripeCreator.CreateCustomer(ctx, email, uid)
	if err != nil {
		return "", releaseCustomerLease(ctx, store, uid, token, fmt.Errorf("create Stripe customer: %w", err))
	}
	if stripeCustomer == nil || stripeCustomer.ID == "" {
		return "", releaseCustomerLease(ctx, store, uid, token, errors.New("Stripe returned a customer without an ID"))
	}

	if err := store.SetUserCustomerID(ctx, uid, token, stripeCustomer.ID); err != nil {
		if existingUser, lookupErr := store.GetUser(ctx, mongomanager.User{UID: uid}); lookupErr == nil && existingUser.CustomerID != "" {
			return existingUser.CustomerID, nil
		}
		return "", releaseCustomerLease(ctx, store, uid, token, fmt.Errorf("save Stripe customer mapping: %w", err))
	}
	return stripeCustomer.ID, nil
}

func releaseCustomerLease(ctx context.Context, store stripeCustomerMappingStore, uid, token string, cause error) error {
	if err := store.ReleaseUserProvisioning(ctx, uid, token); err != nil {
		return errors.Join(cause, fmt.Errorf("release customer provisioning lease: %w", err))
	}
	return cause
}
