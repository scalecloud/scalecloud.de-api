package main

import (
	"github.com/scalecloud/scalecloud.de-api/tree/main/api"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting App.")
	api.InitApi()
	logger.Info("App ended.")
}
