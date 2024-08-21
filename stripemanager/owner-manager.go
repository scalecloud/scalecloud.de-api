package stripemanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/subscription"
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
	err = hasOwnerTriggeredOwnerTransfer(tokenDetails, ownerSeat)
	if err != nil {
		return err
	}
	err = paymentHandler.isCustomerDestinationVerified(c, seatUpdateRequest)
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

func (paymentHandler *PaymentHandler) isCustomerDestinationVerified(c context.Context, seatUpdateRequest mongomanager.Seat) error {
	seatCustomerDestination, err := paymentHandler.MongoConnection.GetSeat(c, seatUpdateRequest.SubscriptionID, seatUpdateRequest.UID)
	if err != nil {
		return err
	}
	if seatCustomerDestination.EMailVerified == nil || !*seatCustomerDestination.EMailVerified {
		return errors.New("new owner's E-Mail is not verified")
	}
	return nil
}

func (paymentHandler *PaymentHandler) handleStripeOwnerTransfer(c context.Context, tokenDetails firebasemanager.TokenDetails, seatUpdateRequest, ownerSeat mongomanager.Seat) error {
	paymentHandler.Log.Info("Owner transfer initiated", zap.Any("seatUpdateRequest", seatUpdateRequest))

	// Retrieve the subscription
	sub, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, seatUpdateRequest.SubscriptionID)
	if err != nil {
		return err
	}
	if sub.Status != stripe.SubscriptionStatusActive {
		return errors.New("subscription is not active")
	}
	if sub.Status == stripe.SubscriptionStatusTrialing {
		return errors.New("subscription is in trial period")
	}
	paymentHandler.Log.Error("customerSourceID or customerDestinationID are wrong, fix and test!")
	// Get the source customer ID
	customerSourceID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return err
	}

	// Search or create the destination customer
	customerDestinationID, err := paymentHandler.searchOrCreateCustomer(c, seatUpdateRequest.EMail, seatUpdateRequest.UID)
	if err != nil {
		return err
	}

	// Update the subscription to associate it with the new customer
	err = paymentHandler.updateSubscriptionToNewCustomer(customerSourceID, customerDestinationID, seatUpdateRequest.SubscriptionID)
	if err != nil {
		return err
	}

	err = paymentHandler.addCustomerTransferNote(customerSourceID, customerSourceID, customerDestinationID, seatUpdateRequest.SubscriptionID)
	if err != nil {
		return err
	}
	err = paymentHandler.addCustomerTransferNote(customerDestinationID, customerSourceID, customerDestinationID, seatUpdateRequest.SubscriptionID)
	if err != nil {
		return err
	}
	paymentHandler.removeSourceCustomerOwner(c, ownerSeat, tokenDetails)
	paymentHandler.sendConfirmationMail()

	return nil
}

func (paymentHandler *PaymentHandler) updateSubscriptionToNewCustomer(customerSourceID, customerDestinationID, subscriptionID string) error {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	updateParams := &stripe.SubscriptionParams{
		Metadata: map[string]string{
			fmt.Sprintf("transfer_ownership_%s", timestamp): fmt.Sprintf("Transferred this subscription from customerSourceID %s to customerDestinationID %s", customerSourceID, customerDestinationID),
		},
	}
	_, err := subscription.Update(subscriptionID, updateParams)
	if err != nil {
		paymentHandler.Log.Error("Error transfering subscription to new owner", zap.Error(err))
		return errors.New("error adding transfer note to customer")
	}
	return nil
}

func (paymentHandler *PaymentHandler) addCustomerTransferNote(customerIDToUpdate, customerSourceID, customerDestinationID, subscriptionID string) error {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	customerUpdateParams := &stripe.CustomerParams{
		Metadata: map[string]string{
			fmt.Sprintf("transfer_ownership_%s", timestamp): fmt.Sprintf("Transferred the subscription %s from customerSourceID %s to customerDestinationID %s", subscriptionID, customerSourceID, customerDestinationID),
		},
	}
	_, err := customer.Update(customerIDToUpdate, customerUpdateParams)
	if err != nil {
		paymentHandler.Log.Error("Error adding transfer note to customer", zap.Error(err))
		return errors.New("error adding transfer note to customer")
	}
	return nil
}

func (paymentHandler *PaymentHandler) removeSourceCustomerOwner(c context.Context, sourceSeat mongomanager.Seat, tokenDetails firebasemanager.TokenDetails) error {
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
