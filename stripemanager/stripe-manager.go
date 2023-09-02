package stripemanager

import (
	"os"

	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

const keyFile = "keys/stripe-secret-key.txt"
const endpointSecretFile = "keys/stripe-endpoint-secrets.txt"

func InitStripe() {
	logger.Info("Init stripe")
	if fileExists(keyFile) {
		logger.Info("Keyfile exists. ", zap.String("file", keyFile))
	} else {
		logger.Error("Keyfile does not exist. ", zap.String("file", keyFile))
		os.Exit(1)
	}
	if fileExists(endpointSecretFile) {
		logger.Info("EndpointSecretFile exists. ", zap.String("file", endpointSecretFile))
	} else {
		logger.Error("endpointSecretFile does not exist. ", zap.String("file", endpointSecretFile))
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
	content, err := os.ReadFile(keyFile)
	if err != nil {
		logger.Error("Error reading file", zap.Error(err))
	}
	key := string(content)
	return key
}

func GetStripeEndpointSecret() string {
	content, err := os.ReadFile(endpointSecretFile)
	if err != nil {
		logger.Error("Error reading file", zap.Error(err))
	}
	key := string(content)
	return key
}
