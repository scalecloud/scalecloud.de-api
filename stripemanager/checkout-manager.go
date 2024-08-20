package stripemanager

import (
	"context"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) createCustomerAndUser(c context.Context, eMail, uid string) (mongomanager.User, error) {
	customer, err := paymentHandler.StripeConnection.CreateCustomer(c, eMail)
	if err != nil {
		paymentHandler.Log.Error("Error creating customer", zap.Error(err))
		return mongomanager.User{}, err
	} else {
		paymentHandler.Log.Info("New Customer was created with Customer.ID", zap.Any("customer.ID", customer.ID))
		newUser := mongomanager.User{
			UID:        uid,
			CustomerID: customer.ID,
		}
		err := paymentHandler.MongoConnection.CreateUser(c, newUser)
		if err != nil {
			paymentHandler.Log.Error("Error creating user in MongoDB.", zap.Error(err))
			return mongomanager.User{}, err
		} else {
			paymentHandler.Log.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser, nil
		}
	}
}

func (paymentHandler *PaymentHandler) searchOrCreateCustomer(c context.Context, eMail, uid string) (string, error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, uid)
	if err != nil {
		paymentHandler.Log.Info("Could not find user in MongoDB. Going to create new Customer in MongoDB Database 'stripe' collection 'users'.")
		paymentHandler.Log.Debug("err", zap.Error(err))
		newUser, err := paymentHandler.createCustomerAndUser(c, eMail, uid)
		if err != nil {
			paymentHandler.Log.Error("Error creating user", zap.Error(err))
			return "", err
		} else {
			paymentHandler.Log.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser.CustomerID, nil
		}
	} else {
		paymentHandler.Log.Info("User was found in MongoDB with customerID", zap.Any("customerID", customerID))
		return customerID, nil
	}
}
