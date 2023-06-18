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

func GetSubscriptionsOverview(c context.Context, token string) ([]stripe.SubscriptionOverview, error) {
	logger.Debug("GetSubscriptionsOverview")
	return stripe.GetSubscriptionsOverview(c, token)
}

func GetSubscriptionByID(c context.Context, token, subscriptionID string) (stripe.SubscriptionDetail, error) {
	logger.Debug("GetSubscriptionByID")
	return stripe.GetSubscriptionByID(c, token, subscriptionID)
}

func GetBillingPortal(c context.Context, token string) (stripe.BillingPortalModel, error) {
	logger.Debug("GetBillingPortal")
	return stripe.GetBillingPortal(c, token)
}

func CreateCheckoutSession(c context.Context, token string, checkoutModelPortalRequest stripe.CheckoutModelPortalRequest) (stripe.CheckoutModelPortalReply, error) {
	logger.Debug("CreateCheckoutSession")
	return stripe.CreateCheckoutSession(c, token, checkoutModelPortalRequest)
}

func CreateCheckoutSubscription(c context.Context, token string, checkoutIntegrationRequest stripe.CheckoutPaymentIntentRequest) (stripe.CheckoutPaymentIntentReply, error) {
	logger.Debug("CreateCheckoutSubscription")
	return stripe.CreateCheckoutSubscription(c, token, checkoutIntegrationRequest)
}

func UpdateCheckoutSubscription(c context.Context, token string, checkoutIntegrationUpdateRequest stripe.CheckoutPaymentIntentUpdateRequest) (stripe.CheckoutPaymentIntentUpdateReply, error) {
	logger.Debug("UpdateCheckoutSubscription")
	return stripe.UpdateCheckoutSubscription(c, token, checkoutIntegrationUpdateRequest)
}

func GetCheckoutProduct(c context.Context, token string, checkoutProductRequest stripe.CheckoutProductRequest) (stripe.CheckoutProductReply, error) {
	logger.Debug("GetCheckoutProduct")
	return stripe.GetCheckoutProduct(c, token, checkoutProductRequest)
}

func CreateCheckoutSetupIntent(c context.Context, token string, checkoutSetupIntentRequest stripe.CheckoutSetupIntentRequest) (stripe.CheckoutSetupIntentReply, error) {
	logger.Debug("CreateCheckoutSetupIntent")
	return stripe.CreateCheckoutSetupIntent(c, token, checkoutSetupIntentRequest)
}

func ResumeSubscription(c context.Context, token string, subscriptionResumeRequest stripe.SubscriptionResumeRequest) (stripe.SubscriptionResumeReply, error) {
	logger.Debug("ResumeSubscription")
	return stripe.ResumeSubscription(c, token, subscriptionResumeRequest)
}

func CancelSubscription(c context.Context, token string, subscriptionCancelRequest stripe.SubscriptionCancelRequest) (stripe.SubscriptionCancelReply, error) {
	logger.Debug("CancelSubscription")
	return stripe.CancelSubscription(c, token, subscriptionCancelRequest)
}

func GetSubscriptionPaymentMethod(c context.Context, token string, subscriptionPaymentMethodRequest stripe.SubscriptionPaymentMethodRequest) (stripe.SubscriptionPaymentMethodReply, error) {
	logger.Debug("GetSubscriptionPaymentMethod")
	return stripe.GetSubscriptionPaymentMethod(c, token, subscriptionPaymentMethodRequest)
}

func ChangeSubscriptionPaymentMethod(c context.Context, token string, changeSubscriptionPaymentMethodRequest stripe.ChangeSubscriptionPaymentMethodRequest) (stripe.ChangeSubscriptionPaymentMethodReply, error) {
	logger.Debug("ChangeSubscriptionPaymentMethod")
	return stripe.ChangeSubscriptionPaymentMethod(c, token, changeSubscriptionPaymentMethodRequest)
}
