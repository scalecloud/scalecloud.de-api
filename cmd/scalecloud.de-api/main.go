package main

import (
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
	api, err := apimanager.InitAPI(log)
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
