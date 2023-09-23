package stripemanager

import (
	"context"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) createCustomerAndUser(c context.Context, tokenDetails firebasemanager.TokenDetails) (mongomanager.User, error) {
	customer, err := stripeConnection.CreateCustomer(c, tokenDetails.EMail)
	if err != nil {
		stripeConnection.Log.Error("Error creating customer", zap.Error(err))
		return mongomanager.User{}, err
	} else {
		stripeConnection.Log.Info("New Customer was created with Customer.ID", zap.Any("customer.ID", customer.ID))
		newUser := mongomanager.User{
			UID:        tokenDetails.UID,
			CustomerID: customer.ID,
		}
		err := mongomanager.CreateUser(c, newUser)
		if err != nil {
			stripeConnection.Log.Error("Error creating user in MongoDB.", zap.Error(err))
			return mongomanager.User{}, err
		} else {
			stripeConnection.Log.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser, nil
		}
	}
}

func (stripeConnection *StripeConnection) searchOrCreateCustomer(c context.Context, filter mongomanager.User, tokenDetails firebasemanager.TokenDetails) (string, error) {
	customerID, err := GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		stripeConnection.Log.Info("Could not find user in MongoDB. Going to create new Customer in MongoDB Database 'stripe' collection 'users'.")
		stripeConnection.Log.Debug("err", zap.Error(err))
		newUser, err := stripeConnection.createCustomerAndUser(c, tokenDetails)
		if err != nil {
			stripeConnection.Log.Error("Error creating user", zap.Error(err))
			return "", err
		} else {
			stripeConnection.Log.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser.CustomerID, nil
		}
	} else {
		stripeConnection.Log.Info("User was found in MongoDB with customerID", zap.Any("customerID", customerID))
		return customerID, nil
	}
}
