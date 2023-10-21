package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/billingportal/session"
)

func (paymentHandler *PaymentHandler) GetBillingPortal(c context.Context, tokenDetails firebasemanager.TokenDetails) (billingPortalModel BillingPortalReply, err error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return BillingPortalReply{}, err
	}
	if customerID == "" {
		return BillingPortalReply{}, errors.New("Customer ID is empty")
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String("https://www.scalecloud.de/dashboard"),
	}
	session, err := session.New(params)
	if err != nil {
		return BillingPortalReply{}, err
	}
	if session.Customer != customerID {
		return BillingPortalReply{}, errors.New("Customer ID does not match")
	}
	if session.URL == "" {
		return BillingPortalReply{}, errors.New("URL is empty")
	}
	billingPortalModel.URL = session.URL
	return billingPortalModel, nil
}
