package stripebillingportal

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/stripecustomer"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/billingportal/session"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func GetBillingPortal(c context.Context, token string) (billingPortalModel BillingPortalModel, err error) {
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return BillingPortalModel{}, err
	}
	customerID, err := stripecustomer.GetCustomerIDByUID(c, tokenDetails.UID)
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
