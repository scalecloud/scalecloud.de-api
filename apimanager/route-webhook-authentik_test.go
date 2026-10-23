package apimanager

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestAuthentikWebhookRejectsInvalidToken(t *testing.T) {
	_, router := newAuthentikWebhookTestRouter("expected-token")
	request := httptest.NewRequest(http.MethodPost, "/webhook/authentik", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer invalid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAuthentikWebhookUnavailableWithoutToken(t *testing.T) {
	_, router := newAuthentikWebhookTestRouter("")
	request := httptest.NewRequest(http.MethodPost, "/webhook/authentik", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestAuthentikWebhookRejectsInvalidPayload(t *testing.T) {
	_, router := newAuthentikWebhookTestRouter("expected-token")
	request := httptest.NewRequest(http.MethodPost, "/webhook/authentik", strings.NewReader(`{"event_uuid":"event-1","action":"login"}`))
	request.Header.Set("Authorization", "Bearer expected-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func newAuthentikWebhookTestRouter(token string) (*Api, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	api := &Api{
		log: zap.NewNop(),
		webhookHandler: &WebhookHandler{
			AuthentikWebhookToken: token,
		},
	}
	router := gin.New()
	router.POST("/webhook/authentik", api.authentikWebhookRequired, api.handleAuthentikWebhook)
	return api, router
}
