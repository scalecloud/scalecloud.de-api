package secret

import (
	"errors"
	"os"
	"strings"

	"go.uber.org/zap"
)

func CheckStripeConfig(log *zap.Logger) error {
	log.Info("Checking Stripe configuration")
	if strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")) == "" {
		return errors.New("STRIPE_SECRET_KEY is empty")
	}
	if strings.TrimSpace(os.Getenv("STRIPE_ENDPOINT_SECRET")) == "" {
		return errors.New("STRIPE_ENDPOINT_SECRET is empty")
	}
	return nil
}

func GetStripeKey() (string, error) {
	key := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	if key == "" {
		return "", errors.New("STRIPE_SECRET_KEY is empty")
	}
	return key, nil
}

func GetStripeEndpointSecret() (string, error) {
	key := strings.TrimSpace(os.Getenv("STRIPE_ENDPOINT_SECRET"))
	if key == "" {
		return "", errors.New("STRIPE_ENDPOINT_SECRET is empty")
	}
	return key, nil
}
