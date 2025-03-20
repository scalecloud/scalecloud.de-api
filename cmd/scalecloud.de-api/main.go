package main

import (
	"flag"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/scalecloud/scalecloud.de-api/apimanager"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var log, err = zap.NewProduction()
	production, proxyIP := parseFlags(log)

	sentryClient, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "https://8195a374d52c2473d306fc8af2517849@o4508966853083136.ingest.de.sentry.io/4508972534661200",
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		Environment:      getEnvironment(production),
	})
	if err != nil {
		log.Fatal("Error initializing production logger", zap.Error(err))
	}
	log = modifyToSentryLogger(log, sentryClient)
	defer sentry.Flush(2 * time.Second)

	if production {
		log.Info("Logging running in production mode.")
	} else {
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
	}
	log.Info("Starting App.")
	api, err := apimanager.InitAPI(log, production, proxyIP)
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

func getEnvironment(production bool) string {
	if production {
		return "production"
	}
	return "development"
}

func parseFlags(log *zap.Logger) (bool, string) {
	var production bool
	var proxyIP string
	flag.BoolVar(&production, "production", false, "Running in production mode. This will create certificates and a trusted proxy.")
	log.Info("Is production?", zap.Bool("isProduction", production))
	flag.StringVar(&proxyIP, "proxyIP", "", "The IP of the proxy. This is needed for the trusted proxy.")
	log.Info("Proxy IP", zap.String("proxyIP", proxyIP))
	flag.Parse()
	return production, proxyIP
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
