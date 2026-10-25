package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCollectorHealthEndpoints(t *testing.T) {
	health := newCollectorHealth(time.Minute)

	liveResponse := requestCollectorHealth(health, "/health/live")
	if liveResponse.Code != http.StatusOK {
		t.Fatalf("liveness status = %d, want %d", liveResponse.Code, http.StatusOK)
	}

	readyResponse := requestCollectorHealth(health, "/health/ready")
	if readyResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("initial readiness status = %d, want %d", readyResponse.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(readyResponse.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("initial readiness body = %s, want not_ready", readyResponse.Body.String())
	}

	successTime := time.Now().In(time.FixedZone("CET", 3600))
	health.recordCollection(nil, successTime)
	readyResponse = requestCollectorHealth(health, "/health/ready")
	if readyResponse.Code != http.StatusOK {
		t.Fatalf("readiness after successful collection = %d, want %d", readyResponse.Code, http.StatusOK)
	}
	var readiness collectorReadiness
	if err := json.Unmarshal(readyResponse.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if readiness.Status != "ready" {
		t.Fatalf("readiness status = %q, want ready", readiness.Status)
	}
	if !readiness.LastSuccessAt.Equal(successTime.UTC()) || readiness.LastSuccessAt.Location() != time.UTC {
		t.Errorf("last success time = %v, want %v in UTC", readiness.LastSuccessAt, successTime.UTC())
	}

	health.recordCollection(errors.New("collection failed"), time.Now())
	readyResponse = requestCollectorHealth(health, "/health/ready")
	if readyResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness after failed collection = %d, want %d", readyResponse.Code, http.StatusServiceUnavailable)
	}
}

func TestCollectorReadinessFailsWhenLastSuccessIsStale(t *testing.T) {
	health := newCollectorHealth(time.Minute)
	health.recordCollection(nil, time.Now().Add(-2*time.Minute))

	response := requestCollectorHealth(health, "/health/ready")
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness with stale collection = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestCollectorHealthRejectsNonGetAndUnknownPaths(t *testing.T) {
	health := newCollectorHealth(time.Minute)

	request := httptest.NewRequest(http.MethodPost, "/health/live", nil)
	response := httptest.NewRecorder()
	health.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST liveness status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}

	response = requestCollectorHealth(health, "/unknown")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown path status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func requestCollectorHealth(health http.Handler, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	health.ServeHTTP(response, request)
	return response
}
