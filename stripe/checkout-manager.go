package stripe

import (
	"context"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/scalecloud/scalecloud.de-api/mongo"
	"go.uber.org/zap"
)

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
