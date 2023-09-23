package secret

import (
	"errors"
	"os"

	"go.uber.org/zap"
)

const keyFile = "keys/stripe-secret-key.txt"
const endpointSecretFile = "keys/stripe-endpoint-secrets.txt"

func CheckStripeKeyFiles(log *zap.Logger) error {
	log.Info("Init stripe")
	if fileExists(keyFile) {
		log.Info("Keyfile exists. ", zap.String("file", keyFile))
	} else {
		return errors.New("Keyfile does not exist")
	}
	if fileExists(endpointSecretFile) {
		log.Info("EndpointSecretFile exists. ", zap.String("file", endpointSecretFile))
	} else {
		return errors.New("endpointSecretFile does not exist")
	}
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func GetStripeKey() (string, error) {
	content, err := os.ReadFile(keyFile)
	if err != nil {
		return "", err
	}
	key := string(content)
	return key, nil
}

func GetStripeEndpointSecret() (string, error) {
	content, err := os.ReadFile(endpointSecretFile)
	if err != nil {
		return "", err
	}
	key := string(content)
	return key, nil
}
