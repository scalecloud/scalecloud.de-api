package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
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
		//var paymentMethod stripe.Payment
	case "setup_intent.succeeded":
		/*  var set stripe.
		err = json.Unmarshal(event.Data.Raw, &setupIntent)
		if err != nil {
			logger.Error("Error unmarshalling setupIntent", zap.Error(err))
			c.SecureJSON(http.StatusBadRequest, gin.H{"message": "Error unmarshalling setupIntent"})
		}  */

		// Then define and call a function to handle the event setup_intent.succeeded
	// ... handle other event types
	default:
		logger.Warn("Unhandled event type", zap.String("event.Type", event.Type))
		c.SecureJSON(http.StatusNotImplemented, gin.H{"message": "Unhandled event type"})
	}

	var request stripemanager.ChangePaymentRequest
	if err := c.BindJSON(&request); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if request.SubscriptionID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	/* reply, error := stripe.GetChangePaymentSetupIntent(c, token, request)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	} */
	/* logger.Info("Handled webhook", zap.Any("Handled webhook", reply)) */
	c.Status(http.StatusOK)
}
