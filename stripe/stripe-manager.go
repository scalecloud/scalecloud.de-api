package stripe

import (
	"io/ioutil"
	"os"

	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

const keyFile = "keys/stripe-secret-key.txt"

func InitStripe() {
	logger.Info("Init stripe")
	if fileExists(keyFile) {
		logger.Info("Keyfile exists. ", zap.String("file", keyFile))
	} else {
		logger.Error("Keyfile does not exist. ", zap.String("file", keyFile))
		os.Exit(1)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getStripeKey() string {
	content, err := ioutil.ReadFile(keyFile)
	if err != nil {
		logger.Error("Error reading file", zap.Error(err))
	}
	key := string(content)
	return key
}
