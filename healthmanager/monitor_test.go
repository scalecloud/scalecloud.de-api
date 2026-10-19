package healthmanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSnapshotReadinessUsesRequiredChecksOnly(t *testing.T) {
	coreErr := errors.New("unavailable")
	monitor := NewMonitor([]Check{
		{Name: "mongo", Required: true, Run: func(context.Context) error { return coreErr }},
		{Name: "smtp", Required: false, Run: func(context.Context) error { return errors.New("unavailable") }},
	}, time.Second, time.Second, zap.NewNop())

	first := monitor.CheckNow(context.Background())
	if first.Status != "not_ready" {
		t.Fatalf("expected not_ready, got %q", first.Status)
	}
	if first.Services["smtp"].Status != "unhealthy" {
		t.Fatalf("expected SMTP status to be reported, got %q", first.Services["smtp"].Status)
	}

	monitor = NewMonitor([]Check{
		{Name: "mongo", Required: true, Run: func(context.Context) error { return nil }},
		{Name: "smtp", Required: false, Run: func(context.Context) error { return errors.New("unavailable") }},
	}, time.Second, time.Second, zap.NewNop())
	if got := monitor.CheckNow(context.Background()).Status; got != "ready" {
		t.Fatalf("expected optional SMTP failure not to block readiness, got %q", got)
	}
}

func TestSnapshotIsNotReadyBeforeChecksRun(t *testing.T) {
	monitor := NewMonitor([]Check{
		{Name: "stripe", Required: true, Run: func(context.Context) error { return nil }},
	}, time.Second, time.Second, zap.NewNop())
	if got := monitor.Snapshot().Status; got != "not_ready" {
		t.Fatalf("expected unknown required check to be not_ready, got %q", got)
	}
}
