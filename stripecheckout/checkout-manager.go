package stripecheckout

import (
	"context"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripecustomer"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func createCustomerAndUser(c context.Context, tokenDetails firebasemanager.TokenDetails) (mongomanager.User, error) {
	customer, err := stripecustomer.CreateCustomer(c, tokenDetails.EMail)
	if err != nil {
		logger.Error("Error creating customer", zap.Error(err))
		return mongomanager.User{}, err
	} else {
		logger.Info("New Customer was created with Customer.ID", zap.Any("customer.ID", customer.ID))
		newUser := mongomanager.User{
			UID:        tokenDetails.UID,
			CustomerID: customer.ID,
		}
		err := mongomanager.CreateUser(c, newUser)
		if err != nil {
			logger.Error("Error creating user in MongoDB.", zap.Error(err))
			return mongomanager.User{}, err
		} else {
			logger.Info("New User was created in MongoDB with User.ID", zap.Any("user.ID", newUser.UID))
			return newUser, nil
		}
	}
}

func searchOrCreateCustomer(c context.Context, filter mongomanager.User, tokenDetails firebasemanager.TokenDetails) (string, error) {
	customerID, err := stripecustomer.GetCustomerIDByUID(c, tokenDetails.UID)
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
