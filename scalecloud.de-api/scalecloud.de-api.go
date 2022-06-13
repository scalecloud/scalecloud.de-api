package scalecloud

import (
	"context"
	"os"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/scalecloud/scalecloud.de-api/mongo"
	"github.com/scalecloud/scalecloud.de-api/stripe"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func Init() {
	logger.Info("Init scalecloud.de-api")
	firebase.InitFirebase()
	mongo.InitMongo()
	stripe.InitStripe()
}

func IsAuthenticated(ctx context.Context, token string) bool {
	return firebase.VerifyIDToken(ctx, token)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetSubscriptionsOverview(c context.Context) (subscriptionsOverview []stripe.SubscriptionOverview, err error) {
	logger.Debug("GetSubscriptionsOverview")
	customerID := "cus_IJNox8VXgkX2gU"
	return stripe.GetSubscriptionsOverview(c, customerID)
}

func GetSubscriptionByID(c context.Context, subscriptionID string) (subscriptionDetail stripe.SubscriptionDetail, err error) {
	logger.Debug("GetSubscriptionByID")
	customerID := "cus_IJNox8VXgkX2gU"
	return stripe.GetSubscriptionByID(c, customerID, subscriptionID)
}

func GetBillingPortal(c context.Context) (subscriptionDetail stripe.BillingPortalModel, err error) {
	logger.Debug("GetBillingPortal")
	customerID := "cus_IJNox8VXgkX2gU"
	return stripe.GetBillingPortal(c, customerID)
}

func CreateCheckoutSession(c context.Context, token, productID string) (checkoutModel stripe.CheckoutModel, err error) {
	logger.Debug("CreateCheckoutSession")
	return stripe.CreateCheckoutSession(c, token, productID)
}
