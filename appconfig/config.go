package appconfig

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Environment string

const (
	Local      Environment = "local"
	Beta       Environment = "beta"
	Production Environment = "production"
)

type Config struct {
	Environment    Environment
	LogLevel       string
	FrontendURL    string
	APIDomain      string
	TrustedProxyIP string
}

func LoadFromEnv() (Config, error) {
	environment, err := LoadEnvironmentFromEnv()
	if err != nil {
		return Config{}, err
	}

	logLevel := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if logLevel == "" {
		logLevel = "info"
	}
	switch logLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, errors.New("LOG_LEVEL must be debug, info, warn, or error")
	}

	frontendURL, err := parseFrontendURL(os.Getenv("FRONTEND_URL"))
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Environment:    environment,
		LogLevel:       logLevel,
		FrontendURL:    frontendURL,
		APIDomain:      strings.TrimSpace(os.Getenv("API_DOMAIN")),
		TrustedProxyIP: strings.TrimSpace(os.Getenv("TRUSTED_PROXY_IP")),
	}

	if environment != Local {
		if !strings.HasPrefix(frontendURL, "https://") {
			return Config{}, errors.New("FRONTEND_URL must use HTTPS for beta and production")
		}
		if config.APIDomain == "" || strings.Contains(config.APIDomain, "/") || strings.Contains(config.APIDomain, "://") {
			return Config{}, errors.New("API_DOMAIN must be a hostname for beta and production")
		}
		if config.TrustedProxyIP == "" {
			return Config{}, errors.New("TRUSTED_PROXY_IP is required for beta and production")
		}
	}

	return config, nil
}

func LoadEnvironmentFromEnv() (Environment, error) {
	environment := Environment(strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))))
	switch environment {
	case Local, Beta, Production:
	default:
		return "", errors.New("APP_ENV must be local, beta, or production")
	}
	return environment, nil
}

func parseFrontendURL(rawURL string) (string, error) {
	value := strings.TrimSpace(rawURL)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("FRONTEND_URL must be an absolute origin URL without credentials, path, query, or fragment")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", errors.New("FRONTEND_URL must use HTTP or HTTPS")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("FRONTEND_URL must not include a path")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func (environment Environment) IsProductionLike() bool {
	return environment == Beta || environment == Production
}

func (environment Environment) String() string {
	return fmt.Sprint(string(environment))
}
