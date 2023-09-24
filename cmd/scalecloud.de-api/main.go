package main

import (
	"flag"
	"os"

	"github.com/scalecloud/scalecloud.de-api/apimanager"
	"go.uber.org/zap"
)

func main() {
	var log, err = zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	log.Info("Starting App.")
	production, proxyIP := parseFlags(log)

	api, err := apimanager.InitAPI(log, production, proxyIP)
	if err != nil {
		log.Fatal("Error initializing API", zap.Error(err))
	}
	defer func() {
		log.Info("Closing MongoDB Client.")
		api.CloseMongoClient()
	}()
	api.RunAPI()
	log.Info("App ended.")
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
