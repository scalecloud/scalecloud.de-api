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

func (webhookHandler *WebhookHandler) handleStripeWebhook(c *gin.Context) {
	var endpointSecret = webhookHandler.StripeConnection.EndpointSecret
	if endpointSecret == "" {
		webhookHandler.Log.Error("Missing endpoint secret")
		c.SecureJSON(http.StatusServiceUnavailable, gin.H{"message": "Service unavailable"})
	}
	payload, err := c.GetRawData()
	if err != nil {
		webhookHandler.Log.Error("Error getting raw data", zap.Error(err))
		c.SecureJSON(http.StatusNoContent, gin.H{"message": "Error getting raw data"})
	}
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		webhookHandler.Log.Error("Signature verification failed", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Signature verification failed"})
	}
	switch event.Type {
	case "payment_method.attached":
		err := handlePaymentMethodAttached(event, webhookHandler.Log)
		if err != nil {
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.created":
		err := handleSetupIntentCreated(event, webhookHandler.Log)
		if err != nil {
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.succeeded":
		err := handleSetupIntentSucceeded(event, webhookHandler.Log)
		if err != nil {
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	default:
		webhookHandler.Log.Warn("Unhandled event type", zap.Any("Unhandled event type", event.Type))
		c.SecureJSON(http.StatusNotImplemented, gin.H{"message": "Unhandled event type"})
	}
	webhookHandler.Log.Info("Handled webhook", zap.Any("Handled webhook", event.Type))
	c.Status(http.StatusOK)
}

func handlePaymentMethodAttached(event stripe.Event, log *zap.Logger) error {
	var paymentMethod stripe.PaymentMethod
	err := json.Unmarshal(event.Data.Raw, &paymentMethod)
	if err != nil {
		log.Error("Error unmarshalling setupIntent", zap.Error(err))
		return errors.New("Error unmarshalling setupIntent")
	}

	if paymentMethod.ID == "" {
		log.Error("ID not set")
		return errors.New("ID not set")
	}
	log.Debug("paymentMethod was updated", zap.Any("paymentMethodID", paymentMethod.ID))
	return nil
}

func handleSetupIntentCreated(event stripe.Event, log *zap.Logger) error {
	var setupIntent stripe.SetupIntent
	err := json.Unmarshal(event.Data.Raw, &setupIntent)
	if err != nil {
		log.Error("Error unmarshalling setupIntent", zap.Error(err))
		return errors.New("Error unmarshalling setupIntent")
	}

	if setupIntent.ID == "" {
		log.Error("ID not set")
		return errors.New("ID not set")
	}
	log.Debug("SetupIntentCreated", zap.Any("setupIntentID", setupIntent.ID))
	return nil
}

func handleSetupIntentSucceeded(event stripe.Event, log *zap.Logger) error {
	var setupIntent stripe.SetupIntent
	err := json.Unmarshal(event.Data.Raw, &setupIntent)
	if err != nil {
		log.Error("Error unmarshalling setupIntent", zap.Error(err))
		return errors.New("Error unmarshalling setupIntent")
	}

	if setupIntent.ID == "" {
		log.Error("ID not set")
		return errors.New("ID not set")
	}
	log.Debug("setupIntentID succeeded", zap.Any("setupIntentID", setupIntent.ID))
	cus := setupIntent.Customer
	if cus == nil {
		log.Error("Customer not set")
		return errors.New("Customer not set")
	}
	if cus.ID == "" {
		log.Error("Customer ID not set")
		return errors.New("Customer ID not set")
	}
	log.Debug("Customer", zap.Any("Customer", cus.ID))

	meta := setupIntent.Metadata
	if meta == nil {
		log.Error("Metadata not set")
		return errors.New("Metadata not set")
	}
	metaType := meta["metaType"]
	if metaType == "" {
		log.Error("Metadata type not set")
		return errors.New("Metadata type not set")
	}
	if metaType == string(stripemanager.CreateSubscription) {
		log.Info("CreateSubscription")
	} else if metaType == string(stripemanager.ChangePayment) {
		log.Info("ChangePayment")
	} else {
		log.Error("Unknown metadata type")
		return errors.New("Unknown metadata type")
	}
	return nil
}
