package main

import (
	"os"

	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func Init() {
	logger.Info("Checking if keys folder exists and is complete.")

}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
