package main

import (
	"os"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/scalecloud/scalecloud.de-api/apimanager"
	"github.com/scalecloud/scalecloud.de-api/appconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	} else if !os.IsNotExist(err) {
		panic(err)
	}

	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	appConfig, err := appconfig.LoadFromEnv()
	if err != nil {
		log.Fatal("Invalid application configuration", zap.Error(err))
	}

	sentryClient, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "https://8195a374d52c2473d306fc8af2517849@o4508966853083136.ingest.de.sentry.io/4508972534661200",
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		Environment:      appConfig.Environment.String(),
	})
	if err != nil {
		log.Fatal("Error initializing production logger", zap.Error(err))
	}
	sentry.CurrentHub().BindClient(sentryClient)
	log = modifyToSentryLogger(log, sentryClient)
	defer sentry.Flush(2 * time.Second)

	if appConfig.Environment == appconfig.Local {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		config.Level.SetLevel(zapcore.InfoLevel)
		log, err = config.Build()
		if err != nil {
			log.Fatal("Error initializing development logger", zap.Error(err))
		}
		log = modifyToSentryLogger(log, sentryClient)
		log.Info("Logging switched to development mode.")
	} else {
		log.Info("Logging running in deployed environment", zap.String("environment", appConfig.Environment.String()))
	}
	log.Info("Starting App.")
	api, err := apimanager.InitAPI(log, appConfig)
	if err != nil {
		log.Fatal("Error initializing API", zap.Error(err))
	}
	defer func() {
		log.Info("Closing MongoDB Client.")
		api.CloseMongoClient()
	}()
	api.RunAPI()
	log.Info("App finished.")
}

func modifyToSentryLogger(log *zap.Logger, client *sentry.Client) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level:             zapcore.ErrorLevel,
		EnableBreadcrumbs: true,
		BreadcrumbLevel:   zapcore.InfoLevel,
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(client))

	if err != nil {
		panic(err)
	}

	log = zapsentry.AttachCoreToLogger(core, log)

	return log.With(zapsentry.NewScope())
}
