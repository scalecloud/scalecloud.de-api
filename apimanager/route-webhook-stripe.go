package apimanager

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/webhook"
	"go.uber.org/zap"
)

func (api *Api) StripeRequired(c *gin.Context) {
	isPost(c, api.log)

	token, hasAuth := getStripeToken(c)
	if hasAuth && token != "" {
		api.log.Debug("Has Stripe Signature", zap.String("token:", token))
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
	}
	payload, err := c.GetRawData()
	if err != nil {
		api.log.Error("Error getting raw data", zap.Error(err))
		c.SecureJSON(http.StatusNoContent, gin.H{"message": "Error getting raw data"})
	}
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		api.log.Error("Signature verification failed", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Signature verification failed"})
	}
	switch event.Type {
	case "payment_method.attached":
		err := api.handlePaymentMethodAttached(event)
		if err != nil {
			api.log.Error("Error handling payment_method.attached", zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.created":
		err := api.handleSetupIntentCreated(event)
		if err != nil {
			api.log.Error("Error handling setup_intent.created", zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.succeeded":
		err := api.handleSetupIntentSucceeded(event)
		if err != nil {
			api.log.Error("Error handling setup_intent.succeeded", zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	default:
		api.log.Warn("Unhandled event type", zap.Any("Unhandled event type", event.Type))
		c.SecureJSON(http.StatusNotImplemented, gin.H{"message": "Unhandled event type"})
	}
	api.log.Info("Handled webhook", zap.Any("Handled webhook", event.Type))
}

func (api *Api) handlePaymentMethodAttached(event stripe.Event) error {
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

func (api *Api) handleSetupIntentCreated(event stripe.Event) error {
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

func (api *Api) handleSetupIntentSucceeded(event stripe.Event) error {
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
		return errors.New("Customer not set")
	}
	if cus.ID == "" {
		return errors.New("Customer ID not set")
	}
	api.log.Debug("Customer", zap.Any("Customer", cus.ID))

	meta := request.Metadata
	if meta == nil {
		return errors.New("Metadata not set")
	}
	metaType := meta["metaType"]
	if metaType == "" {
		return errors.New("Metadata type not set")
	}
	if metaType == string(stripemanager.CreateSubscription) {
		api.log.Info("CreateSubscription")
	} else if metaType == string(stripemanager.ChangePayment) {
		api.log.Info("ChangePayment")
	} else {
		return errors.New("Unknown metadata type")
	}
	return nil
}
