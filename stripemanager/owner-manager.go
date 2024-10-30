package stripemanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/subscription"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) handleOwnerTransfer(c context.Context, tokenDetails firebasemanager.TokenDetails, seatUpdateRequest mongomanager.Seat) error {
	if !isSeatUpdateOwnerTransfer(seatUpdateRequest) {
		return nil
	}
	ownerSeat, err := paymentHandler.MongoConnection.GetOwnerSeat(c, seatUpdateRequest.SubscriptionID)
	if err != nil {
		return err
	}
	if isOwnerSeatUpdate(ownerSeat, seatUpdateRequest) {
		return nil
	}
	err = hasOwnerTriggeredOwnerTransfer(tokenDetails, ownerSeat)
	if err != nil {
		return err
	}
	err = paymentHandler.isSeatDestinationVerified(c, seatUpdateRequest)
	if err != nil {
		return err
	}
	err = paymentHandler.isSeatDestinationStripeCustomer(c, seatUpdateRequest)
	if err != nil {
		return err
	}
	err = paymentHandler.hasCustomerOnlyOneActiveSubscription(c, ownerSeat)
	if err != nil {
		return err
	}
	err = paymentHandler.handleStripeOwnerTransfer(c, tokenDetails, seatUpdateRequest, ownerSeat)
	if err != nil {
		return err
	}
	return nil
}

func isSeatUpdateOwnerTransfer(seatUpdateRequest mongomanager.Seat) bool {
	return mongomanager.ContainsRole(seatUpdateRequest, []mongomanager.Role{mongomanager.RoleOwner})
}

func hasOwnerTriggeredOwnerTransfer(tokenDetails firebasemanager.TokenDetails, ownerSeat mongomanager.Seat) error {
	if tokenDetails.UID != ownerSeat.UID {
		return errors.New("only owner can transfer owner role")
	}
	return nil
}

func isOwnerSeatUpdate(ownerSeat mongomanager.Seat, seatUpdateRequest mongomanager.Seat) bool {
	return mongomanager.ContainsRole(seatUpdateRequest, []mongomanager.Role{mongomanager.RoleOwner}) &&
		ownerSeat.UID == seatUpdateRequest.UID

}

func (paymentHandler *PaymentHandler) isSeatDestinationVerified(c context.Context, seatUpdateRequest mongomanager.Seat) error {
	seatCustomerDestination, err := paymentHandler.MongoConnection.GetSeat(c, seatUpdateRequest.SubscriptionID, seatUpdateRequest.UID)
	if err != nil {
		return err
	}
	if seatCustomerDestination.EMailVerified == nil || !*seatCustomerDestination.EMailVerified {
		return errors.New("new owner's E-Mail is not verified")
	}
	return nil
}

func (paymentHandler *PaymentHandler) isSeatDestinationStripeCustomer(c context.Context, seatUpdateRequest mongomanager.Seat) error {
	exists, err := paymentHandler.existsCustomerByUID(c, seatUpdateRequest.UID)
	if err != nil {
		paymentHandler.Log.Error("Error checking if customer exists by UID", zap.Error(err))
		return errors.New("error checking if customer exists by UID")
	}
	if exists {
		return errors.New("transfer of ownership not possible please contact support")
	}
	return nil
}

func (paymentHandler *PaymentHandler) hasCustomerOnlyOneActiveSubscription(c context.Context, ownerSeat mongomanager.Seat) error {
	ownerCustomerID, err := paymentHandler.GetCustomerIDByUID(c, ownerSeat.UID)
	if err != nil {
		paymentHandler.Log.Error("Error retrieving customerID by UID", zap.Error(err))
		return errors.New("could not retrieve customerID by UID")
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(ownerCustomerID),
	}
	i := subscription.List(params)

	activeSubscriptionsCount := 0
	for i.Next() {
		sub := i.Subscription()
		switch sub.Status {
		case stripe.SubscriptionStatusActive:
			activeSubscriptionsCount++
		case stripe.SubscriptionStatusIncompleteExpired:
			paymentHandler.Log.Info("Incomplete expired subscriptions are ignored", zap.String("status", string(sub.Status)))
		default:
			return paymentHandler.handleSubscriptionStatusError(sub.Status)
		}
	}

	if err := i.Err(); err != nil {
		paymentHandler.Log.Error("Error listing subscriptions for customer", zap.Error(err))
		return errors.New("error listing subscriptions for customer")
	}

	if activeSubscriptionsCount != 1 {
		paymentHandler.Log.Error("Customer does not have exactly one active subscription", zap.Int("activeSubscriptionsCount", activeSubscriptionsCount))
		return errors.New("ownership cannot be transferred if the customer has more than one active subscription, please contact support")
	}

	return nil
}

func (paymentHandler *PaymentHandler) handleSubscriptionStatusError(status stripe.SubscriptionStatus) error {
	errorMessages := map[stripe.SubscriptionStatus]string{
		stripe.SubscriptionStatusCanceled:   "ownership cannot be transferred if the subscription is canceled",
		stripe.SubscriptionStatusIncomplete: "ownership cannot be transferred if the subscription is incomplete",
		stripe.SubscriptionStatusPastDue:    "ownership cannot be transferred if the subscription is past due",
		stripe.SubscriptionStatusPaused:     "ownership cannot be transferred if the subscription is paused",
		stripe.SubscriptionStatusTrialing:   "ownership cannot be transferred if the subscription is in trial period",
		stripe.SubscriptionStatusUnpaid:     "ownership cannot be transferred if the subscription is unpaid",
	}

	if msg, exists := errorMessages[status]; exists {
		return errors.New(msg)
	}

	paymentHandler.Log.Error("Unhandled subscription status", zap.String("status", string(status)))
	return errors.New("ownership cannot be transferred, please contact support")
}

func (paymentHandler *PaymentHandler) handleStripeOwnerTransfer(c context.Context, tokenDetails firebasemanager.TokenDetails, seatUpdateRequest, ownerSeat mongomanager.Seat) error {
	paymentHandler.Log.Info("Owner transfer initiated", zap.Any("seatUpdateRequest", seatUpdateRequest))
	customerSourceID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return err
	}
	err = paymentHandler.updateCustomerEMailToNewOwner(ownerSeat, seatUpdateRequest, customerSourceID)
	if err != nil {
		return err
	}
	paymentHandler.removeSourceCustomerOwner(c, ownerSeat)
	paymentHandler.sendConfirmationMail()

	return nil
}

func (paymentHandler *PaymentHandler) updateCustomerEMailToNewOwner(ownerSeat, seatUpdateRequest mongomanager.Seat, customerSourceID string) error {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	params := &stripe.CustomerParams{
		Email: stripe.String(seatUpdateRequest.EMail),
		Metadata: map[string]string{
			fmt.Sprintf("transfer_ownership_%s", timestamp): fmt.Sprintf("Ownership was transferred from %s to %s for subscription %s", ownerSeat.EMail, seatUpdateRequest.EMail, ownerSeat.SubscriptionID),
		},
	}
	_, err := customer.Update(customerSourceID, params)
	if err != nil {
		paymentHandler.Log.Error("Error updating customer", zap.Error(err))
		return errors.New("error updating customer")
	}
	paymentHandler.Log.Error("implement update user in MongoDB")
	return nil
}

func (paymentHandler *PaymentHandler) removeSourceCustomerOwner(c context.Context, sourceSeat mongomanager.Seat) error {
	var filteredRoles []mongomanager.Role
	for _, role := range sourceSeat.Roles {
		if role != mongomanager.RoleOwner {
			filteredRoles = append(filteredRoles, role)
		}
	}
	sourceSeatUpdate := mongomanager.Seat{
		SubscriptionID: sourceSeat.SubscriptionID,
		UID:            sourceSeat.UID,
		EMail:          sourceSeat.EMail,
		EMailVerified:  sourceSeat.EMailVerified,
		Roles:          filteredRoles,
	}
	return paymentHandler.MongoConnection.UpdateSeat(c, sourceSeatUpdate)
}

func (paymentHandler *PaymentHandler) sendConfirmationMail() {
	paymentHandler.Log.Warn("Send confirmation mail to the new owner")
	paymentHandler.Log.Warn("Send confirmation mail to the old owner")
}
