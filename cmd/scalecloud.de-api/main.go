package main

import (
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting application")

}
