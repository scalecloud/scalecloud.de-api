package main

import (
	"flag"

	"github.com/scalecloud/scalecloud.de-api/apimanager"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var log, err = zap.NewProduction()
	if err != nil {
		log.Fatal("Error initializing production logger", zap.Error(err))
	}
	production, proxyIP := parseFlags(log)
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
