package main

import (
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting application")
	Init()
}

func Init() {
	logger.Info("Checking if keys folder exists and is complete.")
}
