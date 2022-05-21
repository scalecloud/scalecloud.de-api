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
	logger.Info("GetSubscriptionsOverview")
	customer := "customer_1"
	return stripe.GetSubscriptionsOverview(c, customer)
}

func GetSubscriptionByID(c context.Context, id string) (subscriptionDetail stripe.SubscriptionDetail, err error) {
	logger.Info("GetSubscriptionByID")
	customer := "customer_1"
	return stripe.GetSubscriptionByID(c, id, customer)
}
