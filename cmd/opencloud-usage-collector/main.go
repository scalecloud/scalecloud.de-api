package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const sampleInterval = time.Minute

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "OpenCloud usage collector stopped:", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			return fmt.Errorf("load .env: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check .env: %w", err)
	}

	log, err := newLogger()
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mongodbmanager.CheckMongoDBConfig(log); err != nil {
		return fmt.Errorf("invalid MongoDB configuration: %w", err)
	}
	mongoDBConnection, err := mongodbmanager.InitMongoDBConnection(ctx, log)
	if err != nil {
		return fmt.Errorf("initialize MongoDB connection: %w", err)
	}
	defer func() {
		disconnectContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongoDBConnection.Client.Disconnect(disconnectContext); err != nil {
			log.Error("Error closing MongoDB", zap.Error(err))
		}
	}()

	if err := mongoDBConnection.CheckHealth(ctx); err != nil {
		return fmt.Errorf("MongoDB health check failed: %w", err)
	}

	openCloudConnection, configured, err := opencloudmanager.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("invalid OpenCloud configuration: %w", err)
	}
	if !configured {
		return errors.New("OpenCloud is not configured; set OPENCLOUD_BASE_URL, OPENCLOUD_USERNAME, and OPENCLOUD_APP_TOKEN")
	}
	if err := openCloudConnection.CheckHealth(ctx); err != nil {
		return fmt.Errorf("OpenCloud health check failed: %w", err)
	}

	paymentHandler := &stripemanager.PaymentHandler{
		MongoDBConnection:   mongoDBConnection,
		OpenCloudConnection: openCloudConnection,
		Log:                 log.Named("paymenthandler"),
	}
	log.Info("Starting OpenCloud usage collector", zap.Duration("interval", sampleInterval))
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	for {
		if err := paymentHandler.SampleOpenCloudUsage(ctx); err != nil && ctx.Err() == nil {
			log.Warn("OpenCloud usage sampling failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			log.Info("Stopping OpenCloud usage collector")
			return nil
		case <-ticker.C:
		}
	}
}

func newLogger() (*zap.Logger, error) {
	levelText := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if levelText == "" {
		levelText = "info"
	}
	switch levelText {
	case "debug", "info", "warn", "error":
	default:
		return nil, errors.New("LOG_LEVEL must be debug, info, warn, or error")
	}

	config := zap.NewProductionConfig()
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return nil, err
	}
	config.Level.SetLevel(level)
	return config.Build()
}
