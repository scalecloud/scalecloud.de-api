package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/billingportal/session"
)

func (paymentHandler *PaymentHandler) GetBillingPortal(c context.Context, tokenDetails firebasemanager.TokenDetails) (billingPortalModel BillingPortalModel, err error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return BillingPortalModel{}, err
	}
	if customerID == "" {
		return BillingPortalModel{}, errors.New("Customer ID is empty")
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String("https://www.scalecloud.de/dashboard"),
	}
	session, err := session.New(params)
	if err != nil {
		return BillingPortalModel{}, err
	}
	if session.Customer != customerID {
		return BillingPortalModel{}, errors.New("Customer ID does not match")
	}
	if session.URL == "" {
		return BillingPortalModel{}, errors.New("URL is empty")
	}
	billingPortalModel.URL = session.URL
	return billingPortalModel, nil
}
