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

func StripeRequired(c *gin.Context) {
	isPost(c)

	token, hasAuth := getStripeToken(c)
	if hasAuth && token != "" {
		logger.Debug("Has Stripe Signature", zap.String("token:", token))
		c.Next()
	} else {
		logger.Warn("Unauthorized")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func isPost(c *gin.Context) {
	if c.Request.Method != http.MethodPost {
		logger.Warn("Method not allowed", zap.String("Method", c.Request.Method))
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

func handleStripeWebhook(c *gin.Context) {

	var endpointSecret = stripemanager.GetStripeEndpointSecret()
	if endpointSecret == "" {
		logger.Error("Missing endpoint secret")
		c.SecureJSON(http.StatusServiceUnavailable, gin.H{"message": "Service unavailable"})
	}

	payload, err := c.GetRawData()
	if err != nil {
		logger.Error("Error getting raw data", zap.Error(err))
		c.SecureJSON(http.StatusNoContent, gin.H{"message": "Error getting raw data"})
	}
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		logger.Error("Signature verification failed", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Signature verification failed"})
	}

	switch event.Type {
	case "payment_method.attached":
		err := handlePaymentMethodAttached(event)
		if err != nil {
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.created":
		err := handleSetupIntentCreated(event)
		if err != nil {
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	case "setup_intent.succeeded":
		err := handleSetupIntentSucceeded(event)
		if err != nil {
			c.SecureJSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	default:
		logger.Warn("Unhandled event type", zap.Any("Unhandled event type", event.Type))
		c.SecureJSON(http.StatusNotImplemented, gin.H{"message": "Unhandled event type"})
	}
	logger.Info("Handled webhook", zap.Any("Handled webhook", event.Type))
	c.Status(http.StatusOK)
}

func handlePaymentMethodAttached(event stripe.Event) error {
	var paymentMethod stripe.PaymentMethod
	err := json.Unmarshal(event.Data.Raw, &paymentMethod)
	if err != nil {
		logger.Error("Error unmarshalling setupIntent", zap.Error(err))
		return errors.New("Error unmarshalling setupIntent")
	}

	if paymentMethod.ID == "" {
		logger.Error("ID not set")
		return errors.New("ID not set")
	}
	logger.Debug("paymentMethod was updated", zap.Any("paymentMethodID", paymentMethod.ID))
	return nil
}

func handleSetupIntentCreated(event stripe.Event) error {
	var setupIntent stripe.SetupIntent
	err := json.Unmarshal(event.Data.Raw, &setupIntent)
	if err != nil {
		logger.Error("Error unmarshalling setupIntent", zap.Error(err))
		return errors.New("Error unmarshalling setupIntent")
	}

	if setupIntent.ID == "" {
		logger.Error("ID not set")
		return errors.New("ID not set")
	}
	logger.Info("SetupIntentCreated", zap.Any("setupIntentID", setupIntent.ID))
	return nil
}

func handleSetupIntentSucceeded(event stripe.Event) error {
	var setupIntent stripe.SetupIntent
	err := json.Unmarshal(event.Data.Raw, &setupIntent)
	if err != nil {
		logger.Error("Error unmarshalling setupIntent", zap.Error(err))
		return errors.New("Error unmarshalling setupIntent")
	}

	if setupIntent.ID == "" {
		logger.Error("ID not set")
		return errors.New("ID not set")
	}
	logger.Info("setupIntentID succeeded", zap.Any("setupIntentID", setupIntent.ID))
	return nil
}
