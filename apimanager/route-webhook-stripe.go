package apimanager

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/subscription"
	"github.com/stripe/stripe-go/v87/webhook"
	"go.uber.org/zap"
)

func (api *Api) StripeRequired(c *gin.Context) {
	isPost(c, api.log)

	token, hasAuth := getStripeToken(c)
	if hasAuth && token != "" {
		c.Next()
	} else {
		api.log.Warn("Unauthorized")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func isPost(c *gin.Context, log *zap.Logger) {
	if c.Request.Method != http.MethodPost {
		log.Warn("Method not allowed", zap.String("Method", c.Request.Method))
		c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}

func getStripeToken(c *gin.Context) (string, bool) {
	token := c.Request.Header.Get("Stripe-Signature")
	if token == "" {
		return "", false
	} else {
		return token, true
	}
}

func (api *Api) handleStripeWebhook(c *gin.Context) {
	var endpointSecret = api.webhookHandler.StripeConnection.EndpointSecret
	if endpointSecret == "" {
		api.log.Error("Missing endpoint secret")
		c.SecureJSON(http.StatusServiceUnavailable, gin.H{"message": "Service unavailable"})
		return
	}
	payload, err := c.GetRawData()
	if err != nil {
		api.log.Warn("Could not read Stripe webhook request")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "Error getting raw data"})
		return
	}
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		api.log.Warn("Stripe webhook signature verification failed")
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Signature verification failed"})
		return
	}
	switch event.Type {
	case "payment_method.attached":
		err := api.handlePaymentMethodAttached(c, event)
		if err != nil {
			api.log.Error("Could not handle payment method attached webhook", zap.String("event_type", string(event.Type)), zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.created":
		err := api.handleSetupIntentCreated(c, event)
		if err != nil {
			api.log.Error("Could not handle setup intent created webhook", zap.String("event_type", string(event.Type)), zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.succeeded":
		err := api.handleSetupIntentSucceeded(c, event)
		if err != nil {
			api.log.Error("Could not handle setup intent succeeded webhook", zap.String("event_type", string(event.Type)), zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "customer.subscription.created":
		err := api.handleCustomerSubscriptionCreated(c, event)
		if err != nil {
			api.log.Error("Could not handle subscription created webhook", zap.String("event_type", string(event.Type)), zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "customer.subscription.deleted":
		err := api.handleCustomerSubscriptionDeleted(c, event)
		if err != nil {
			api.log.Error("Could not handle subscription deleted webhook", zap.String("event_type", string(event.Type)), zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	default:
		api.log.Warn("Unhandled Stripe webhook event", zap.String("event_type", string(event.Type)))
		c.SecureJSON(http.StatusNotImplemented, gin.H{"message": "Unhandled event type"})
	}
	api.log.Info("Handled webhook", zap.Any("Handled webhook", event.Type))
}

func (api *Api) handlePaymentMethodAttached(c context.Context, event stripe.Event) error {
	var request stripe.PaymentMethod
	err := json.Unmarshal(event.Data.Raw, &request)
	if err != nil {
		return err
	}
	err = api.validate.Struct(request)
	if err != nil {
		return err
	}
	api.log.Debug("paymentMethod was updated", zap.Any("paymentMethodID", request.ID))
	return nil
}

func (api *Api) handleSetupIntentCreated(c context.Context, event stripe.Event) error {
	var request stripe.SetupIntent
	err := json.Unmarshal(event.Data.Raw, &request)
	if err != nil {
		return err
	}
	err = api.validate.Struct(request)
	if err != nil {
		return err
	}
	api.log.Debug("SetupIntentCreated", zap.Any("setupIntentID", request.ID))
	return nil
}

func (api *Api) handleSetupIntentSucceeded(c context.Context, event stripe.Event) error {
	var request stripe.SetupIntent
	err := json.Unmarshal(event.Data.Raw, &request)
	if err != nil {
		return err
	}
	err = api.validate.Struct(request)
	if err != nil {
		return err
	}
	cus := request.Customer
	if cus == nil {
		return errors.New("customer not set")
	}
	if cus.ID == "" {
		return errors.New("customer ID not set")
	}
	api.log.Debug("Customer", zap.Any("Customer", cus.ID))

	meta := request.Metadata
	if meta == nil {
		return errors.New("metadata not set")
	}
	metaKey := meta["setupIntentMeta"]
	if metaKey == "" {
		return errors.New("metadata type not set")
	}
	if metaKey == string(stripemanager.CreateSubscription) {
		api.log.Info("createSubscription")
	} else if metaKey == string(stripemanager.ChangePayment) {
		if err := api.paymentHandler.ChangePaymentDefault(c, request); err != nil {
			return err
		}
		if err := api.paymentHandler.ChangeCustomerAddress(c, request); err != nil {
			return err
		}
	} else {
		return errors.New("unknown metadata type")
	}
	return nil
}

func (api *Api) handleCustomerSubscriptionCreated(c context.Context, event stripe.Event) error {
	err := api.handleAddingTrialUsed(c, event)
	if err != nil {
		return err
	}
	return nil
}

func (api *Api) handleCustomerSubscriptionDeleted(c context.Context, event stripe.Event) error {
	err := api.handleSubscriptionEnded(c, event)
	if err != nil {
		return err
	}
	return nil
}

func (api *Api) handleAddingTrialUsed(c context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	err := json.Unmarshal(event.Data.Raw, &sub)
	if err != nil {
		return err
	}
	err = api.validate.Struct(sub)
	if err != nil {
		return err
	}
	status := sub.Status
	if status != stripe.SubscriptionStatusTrialing {
		api.log.Info("Subscription status is not trialing, no need for action.", zap.Any("status", status))
		return nil
	}
	quantity := sub.Items.Data[0].Quantity
	if quantity != 1 {
		return errors.New("Quantity is not 1. It should not be possible to start a subscription with a trial period. SubscriptionID: " + sub.ID)
	}
	if sub.Items.Data[0].Price == nil {
		return errors.New("price not set")
	} else if sub.Items.Data[0].Price.Product == nil {
		return errors.New("product not set")
	} else if sub.Items.Data[0].Price.Product.ID == "" {
		return errors.New("Product.ID name not set")
	}
	prod, err := api.paymentHandler.StripeConnection.GetProduct(c, sub.Items.Data[0].Price.Product.ID)
	if err != nil {
		return err
	}
	metaDataProduct := prod.Metadata

	productType, ok := metaDataProduct["productType"]
	if !ok {
		return errors.New("productType not found for product: " + prod.ID)
	}
	cus, err := stripemanager.GetCustomerByID(c, sub.Customer.ID)
	if err != nil {
		return err
	}
	if cus.InvoiceSettings == nil {
		return errors.New("InvoiceSettings not set")
	} else if cus.InvoiceSettings.DefaultPaymentMethod == nil {
		return errors.New("DefaultPaymentMethod not set")
	} else if cus.InvoiceSettings.DefaultPaymentMethod.ID == "" {
		return errors.New("DefaultPaymentMethod.ID not set")
	}
	paymentMethod, err := api.paymentHandler.StripeConnection.GetPaymentMethod(c, cus.InvoiceSettings.DefaultPaymentMethod.ID)
	if err != nil {
		return err
	}
	cardFingerprint := ""
	if paymentMethod.Card != nil {
		cardFingerprint = paymentMethod.Card.Fingerprint
	}
	paypalPayerEmail := ""
	if paymentMethod.Paypal != nil {
		paypalPayerEmail = paymentMethod.Paypal.PayerEmail
	}
	sepaDebitFingerprint := ""
	if paymentMethod.SEPADebit != nil {
		sepaDebitFingerprint = paymentMethod.SEPADebit.Fingerprint
	}
	trial := mongodbmanager.Trial{
		SubscriptionID:         sub.ID,
		ProductType:            productType,
		CustomerID:             cus.ID,
		PaymentCardFingerprint: cardFingerprint,
		PaymentPayPalEMail:     paypalPayerEmail,
		PaymentSEPAFingerprint: sepaDebitFingerprint,
	}
	err = api.paymentHandler.MongoDBConnection.CreateTrial(c, trial)
	if err != nil {
		return err
	}
	return nil
}

func (api *Api) handleSubscriptionEnded(c context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	err := json.Unmarshal(event.Data.Raw, &sub)
	if err != nil {
		return err
	}
	err = api.validate.Struct(sub)
	if err != nil {
		return err
	}
	status := sub.Status
	if status != stripe.SubscriptionStatusCanceled {
		api.log.Warn("Subscription is not canceled but customer.subscription.deleted was called.", zap.Any("SubscriptionID", sub.ID))
		return errors.New("Subscription is not canceled but customer.subscription.deleted was called. SubscriptionID: " + sub.ID)
	}
	err = api.removeStripeUser(c, sub.Customer.ID)
	if err != nil {
		return err
	}
	err = api.removeSubscriptionSeats(c, sub.ID)
	if err != nil {
		return err
	}
	return nil
}

func (api *Api) removeStripeUser(c context.Context, customerID string) error {
	stripe.Key = api.paymentHandler.StripeConnection.Key
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	iter := subscription.List(params)
	for iter.Next() {
		subscription := iter.Subscription()
		if subscription.Status != stripe.SubscriptionStatusCanceled {
			api.log.Info("No need to remove user as there is an active subscription", zap.Any("SubscriptionID", subscription.ID))
			return nil
		}
	}
	err := api.paymentHandler.MongoDBConnection.DeleteUser(c, customerID)
	if err != nil {
		return err
	}
	api.log.Info("User removed because all subscriptions are canceled", zap.Any("CustomerID", customerID))
	return nil
}

func (api *Api) removeSubscriptionSeats(c context.Context, subscriptionID string) error {
	seats, err := api.paymentHandler.MongoDBConnection.GetAllSeats(c, subscriptionID)
	if err != nil {
		return err
	}
	for _, seat := range seats {
		if seat.Status == mongodbmanager.SeatStatusPending {
			err = api.paymentHandler.MongoDBConnection.DeletePendingSeat(c, seat.SubscriptionID, seat.EMail)
		} else {
			err = api.paymentHandler.RemoveSeatFromSubscriptionGroup(c, seat)
			if err == nil {
				err = api.paymentHandler.MongoDBConnection.DeleteSeat(c, seat)
			}
		}
		if err != nil {
			return err
		}
	}
	api.log.Info("All seats removed", zap.Any("SubscriptionID", subscriptionID))
	return nil
}
