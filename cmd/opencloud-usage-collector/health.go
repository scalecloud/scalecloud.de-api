package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type collectorHealth struct {
	mu            sync.RWMutex
	maxAge        time.Duration
	ready         bool
	lastSuccessAt time.Time
}

type collectorReadiness struct {
	Status        string    `json:"status"`
	LastSuccessAt time.Time `json:"lastSuccessAt,omitempty"`
}

func newCollectorHealth(maxAge time.Duration) *collectorHealth {
	return &collectorHealth{maxAge: maxAge}
}

func (health *collectorHealth) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch request.URL.Path {
	case "/health/live":
		response.WriteHeader(http.StatusOK)
	case "/health/ready":
		health.mu.RLock()
		ready := health.ready
		lastSuccessAt := health.lastSuccessAt
		health.mu.RUnlock()
		ready = ready && time.Since(lastSuccessAt) <= health.maxAge

		statusCode := http.StatusOK
		status := "ready"
		if !ready {
			statusCode = http.StatusServiceUnavailable
			status = "not_ready"
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(statusCode)
		_ = json.NewEncoder(response).Encode(collectorReadiness{
			Status:        status,
			LastSuccessAt: lastSuccessAt,
		})
	default:
		http.NotFound(response, request)
	}
}

func (health *collectorHealth) recordCollection(resultErr error, completedAt time.Time) {
	health.mu.Lock()
	defer health.mu.Unlock()
	health.ready = resultErr == nil
	if resultErr == nil {
		health.lastSuccessAt = completedAt.UTC()
	}
}
