package apimanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/healthmanager"
	"go.uber.org/zap"
)

func TestHealthReadyReturnsServiceStatusesAndIgnoresOptionalFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mongoErr := errors.New("database unavailable")
	monitor := healthmanager.NewMonitor([]healthmanager.Check{
		{Name: "mongodb", Required: true, Run: func(context.Context) error { return mongoErr }},
		{Name: "smtp", Required: false, Run: func(context.Context) error { return errors.New("smtp unavailable") }},
	}, time.Second, time.Second, zap.NewNop())
	api := &Api{healthMonitor: monitor}
	router := gin.New()
	router.GET("/health/ready", api.healthReady)

	monitor.CheckNow(context.Background())
	if response := requestHealth(router); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 while MongoDB is unhealthy, got %d", response.Code)
	}

	mongoErr = nil
	monitor.CheckNow(context.Background())
	response := requestHealth(router)
	if response.Code != http.StatusOK {
		t.Fatalf("expected optional SMTP failure not to block readiness, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"smtp":{"status":"unhealthy","required":false`) {
		t.Fatalf("expected response to report SMTP as unhealthy and optional, got %s", response.Body.String())
	}
}

func requestHealth(router http.Handler) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(response, request)
	return response
}
