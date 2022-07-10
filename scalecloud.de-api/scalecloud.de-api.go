package scalecloud

import (
	"context"

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

func GetSubscriptionsOverview(c context.Context, token string) (subscriptionsOverview []stripe.SubscriptionOverview, err error) {
	logger.Debug("GetSubscriptionsOverview")
	return stripe.GetSubscriptionsOverview(c, token)
}

func GetSubscriptionByID(c context.Context, token, subscriptionID string) (subscriptionDetail stripe.SubscriptionDetail, err error) {
	logger.Debug("GetSubscriptionByID")
	return stripe.GetSubscriptionByID(c, token, subscriptionID)
}

func GetBillingPortal(c context.Context, token string) (subscriptionDetail stripe.BillingPortalModel, err error) {
	logger.Debug("GetBillingPortal")
	return stripe.GetBillingPortal(c, token)
}

func CreateCheckoutSession(c context.Context, token string, productmodel stripe.ProductModel) (checkoutModel stripe.CheckoutModel, err error) {
	logger.Debug("CreateCheckoutSession")
	return stripe.CreateCheckoutSession(c, token, productmodel)
}

func CreateCheckoutSubscription(c context.Context, token string, productmodel stripe.ProductModel) (checkoutSubscriptionModel stripe.CheckoutSubscriptionModel, err error) {
	logger.Debug("CreateCheckoutSubscription")
	return stripe.CreateCheckoutSubscription(c, token, productmodel)
}
