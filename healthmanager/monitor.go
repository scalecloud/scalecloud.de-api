package healthmanager

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Check struct {
	Name     string
	Required bool
	Run      func(context.Context) error
}

type ServiceStatus struct {
	Status    string    `json:"status"`
	Required  bool      `json:"required"`
	CheckedAt time.Time `json:"checkedAt,omitempty"`
}

type Snapshot struct {
	Status   string                   `json:"status"`
	Services map[string]ServiceStatus `json:"services"`
}

type Monitor struct {
	checks   []Check
	interval time.Duration
	timeout  time.Duration
	logger   *zap.Logger

	mu       sync.RWMutex
	statuses map[string]ServiceStatus
}

func NewMonitor(checks []Check, interval, timeout time.Duration, logger *zap.Logger) *Monitor {
	return &Monitor{
		checks:   checks,
		interval: interval,
		timeout:  timeout,
		logger:   logger,
		statuses: make(map[string]ServiceStatus, len(checks)),
	}
}

func (monitor *Monitor) CheckNow(ctx context.Context) Snapshot {
	var waitGroup sync.WaitGroup
	for _, check := range monitor.checks {
		check := check
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			checkContext, cancel := context.WithTimeout(ctx, monitor.timeout)
			defer cancel()
			err := check.Run(checkContext)
			monitor.record(check, err)
		}()
	}
	waitGroup.Wait()
	return monitor.Snapshot()
}

func (monitor *Monitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(monitor.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				monitor.CheckNow(ctx)
			}
		}
	}()
}

func (monitor *Monitor) Snapshot() Snapshot {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	services := make(map[string]ServiceStatus, len(monitor.checks))
	ready := true
	for _, check := range monitor.checks {
		status, exists := monitor.statuses[check.Name]
		if !exists {
			status = ServiceStatus{Status: "unknown", Required: check.Required}
		}
		services[check.Name] = status
		if check.Required && status.Status != "healthy" {
			ready = false
		}
	}
	result := "ready"
	if !ready {
		result = "not_ready"
	}
	return Snapshot{Status: result, Services: services}
}

func (monitor *Monitor) record(check Check, err error) {
	status := ServiceStatus{Status: "healthy", Required: check.Required, CheckedAt: time.Now().UTC()}
	if err != nil {
		status.Status = "unhealthy"
	}

	monitor.mu.Lock()
	previous, exists := monitor.statuses[check.Name]
	monitor.statuses[check.Name] = status
	monitor.mu.Unlock()

	if !exists || previous.Status != status.Status {
		if err != nil {
			monitor.logger.Warn("Service health changed", zap.String("service", check.Name), zap.String("status", status.Status), zap.Error(err))
		} else {
			monitor.logger.Info("Service health changed", zap.String("service", check.Name), zap.String("status", status.Status))
		}
	}
}
