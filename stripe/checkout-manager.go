package stripe

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/scalecloud/scalecloud.de-api/mongo"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"go.uber.org/zap"
)

func CreateCheckoutSession(c context.Context, token string, productmodel ProductModel) (CheckoutModel, error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutModel{}, err
	}
	filter := mongo.User{
		UID: tokenDetails.UID,
	}
	customerID, err := searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return CheckoutModel{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutModel{}, errors.New("Customer ID is empty")
	}
	stripe.Key = getStripeKey()

	price, err := getPrice(c, productmodel.ProductID)
	if err != nil {
		logger.Error("Error getting price", zap.Error(err))
		return CheckoutModel{}, err
	}
	domain := "https://scalecloud.de/checkout"
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(productmodel.Quantity),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(domain + "/success.html"),
		CancelURL:  stripe.String(domain + "/cancel.html"),
		Customer:   stripe.String(customerID),
	}

	session, err := session.New(params)
	if err != nil {
		logger.Error("Error creating session", zap.Error(err))
		return CheckoutModel{}, err
	}

	checkoutModel := CheckoutModel{
		URL: session.URL,
	}
	return checkoutModel, nil
}

func createCustomerAndUser(c context.Context, tokenDetails firebase.TokenDetails) (mongo.User, error) {
	customer, err := CreateCustomer(c, tokenDetails.Email)
	if err != nil {
		logger.Error("Error creating customer", zap.Error(err))
		return mongo.User{}, err
	} else {
		logger.Info("New Customer was created with Customer.ID", zap.Any("customer.ID", customer.ID))
		newUser := mongo.User{
			UID:        tokenDetails.UID,
			CustomerID: customer.ID,
		}
		err := mongo.CreateUser(c, newUser)
		if err != nil {
			logger.Error("Error creating user in MongoDB.", zap.Error(err))
			return mongo.User{}, err
		} else {
			logger.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser, nil
		}
	}
}

func searchOrCreateCustomer(c context.Context, filter mongo.User, tokenDetails firebase.TokenDetails) (string, error) {
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Info("Could not find user in MongoDB. Going to create new Customer in MongoDB Database 'stripe' collection 'users'.")
		logger.Debug("err", zap.Error(err))
		newUser, err := createCustomerAndUser(c, tokenDetails)
		if err != nil {
			logger.Error("Error creating user", zap.Error(err))
			return "", err
		} else {
			logger.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser.CustomerID, nil
		}
	} else {
		logger.Info("User was found in MongoDB with customerID", zap.Any("customerID", customerID))
		return customerID, nil
	}
}
