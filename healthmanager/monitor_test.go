package healthmanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func TestHealthFailuresUseSeverityBasedOnRequirement(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	monitor := NewMonitor([]Check{
		{Name: "mongo", Required: true, Run: func(context.Context) error { return errors.New("unavailable") }},
		{Name: "smtp", Required: false, Run: func(context.Context) error { return errors.New("unavailable") }},
	}, time.Second, time.Second, zap.New(core))

	monitor.CheckNow(context.Background())
	if count := logs.FilterLevelExact(zapcore.ErrorLevel).FilterMessage("Service health changed").Len(); count != 1 {
		t.Fatalf("error-level health logs = %d, want 1", count)
	}
	if count := logs.FilterLevelExact(zapcore.WarnLevel).FilterMessage("Service health changed").Len(); count != 1 {
		t.Fatalf("warning-level health logs = %d, want 1", count)
	}
}
