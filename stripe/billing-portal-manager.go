package stripe

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/billingportal/session"
	"go.uber.org/zap"
)

func GetBillingPortal(c context.Context, token string) (billingPortalModel BillingPortalModel, err error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return BillingPortalModel{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return BillingPortalModel{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return BillingPortalModel{}, errors.New("Customer ID is empty")
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String("https://www.scalecloud.de/dashboard"),
	}
	session, err := session.New(params)
	if err != nil {
		logger.Error("Error creating session", zap.Error(err))
		return BillingPortalModel{}, err
	}
	if session.Customer != customerID {
		logger.Error("Customer ID does not match")
		return BillingPortalModel{}, errors.New("Customer ID does not match")
	}
	if session.URL == "" {
		logger.Error("URL is empty")
		return BillingPortalModel{}, errors.New("URL is empty")
	}
	billingPortalModel.URL = session.URL
	return billingPortalModel, nil
}
