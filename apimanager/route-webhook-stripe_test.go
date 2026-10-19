package apimanager

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
	"go.uber.org/zap"
)

const testStripeWebhookSecret = "whsec_test_endpoint_secret"

func TestStripeWebhookRejectsInvalidSignature(t *testing.T) {
	_, router := newStripeWebhookTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
	request.Header.Set("Stripe-Signature", "t=1,v1=invalid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

func TestStripeWebhookReturnsServerErrorWhenEventProcessingFails(t *testing.T) {
	_, router := newStripeWebhookTestRouter()
	payload := fmt.Sprintf(`{"id":"evt_test","object":"event","api_version":%q,"created":1700000000,"data":{"object":{}},"livemode":false,"pending_webhooks":1,"type":"setup_intent.succeeded"}`, stripe.APIVersion)
	request := signedStripeWebhookRequest(t, payload)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func newStripeWebhookTestRouter() (*Api, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	api := &Api{
		log:      zap.NewNop(),
		validate: validator.New(validator.WithRequiredStructEnabled()),
		webhookHandler: &WebhookHandler{
			StripeConnection: &stripemanager.StripeConnection{EndpointSecret: testStripeWebhookSecret},
			Log:              zap.NewNop(),
		},
	}
	router := gin.New()
	router.POST("/webhook", api.handleStripeWebhook)
	return api, router
}

func signedStripeWebhookRequest(t *testing.T, payload string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	signedPayload := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   []byte(payload),
		Secret:    testStripeWebhookSecret,
		Timestamp: time.Now(),
	})
	request.Header.Set("Stripe-Signature", signedPayload.Header)
	return request
}
