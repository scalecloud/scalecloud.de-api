package appconfig

import "testing"

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		environment  string
		frontendURL  string
		apiDomain    string
		proxyIP      string
		logLevel     string
		wantLogLevel string
		wantError    bool
	}{
		{
			name:         "local",
			environment:  "local",
			frontendURL:  "http://localhost:4200",
			wantLogLevel: "info",
		},
		{
			name:         "debug log level",
			environment:  "local",
			frontendURL:  "http://localhost:4200",
			logLevel:     "debug",
			wantLogLevel: "debug",
		},
		{
			name:         "beta",
			environment:  "beta",
			frontendURL:  "https://beta.scalecloud.de",
			apiDomain:    "api-beta.scalecloud.de",
			proxyIP:      "127.0.0.1",
			wantLogLevel: "info",
		},
		{
			name:        "invalid log level",
			environment: "local",
			frontendURL: "http://localhost:4200",
			logLevel:    "trace",
			wantError:   true,
		},
		{
			name:        "unknown environment",
			environment: "test",
			frontendURL: "http://localhost:4200",
			wantError:   true,
		},
		{
			name:        "beta requires a proxy",
			environment: "beta",
			frontendURL: "https://beta.scalecloud.de",
			apiDomain:   "api-beta.scalecloud.de",
			wantError:   true,
		},
		{
			name:        "production rejects HTTP",
			environment: "production",
			frontendURL: "http://www.scalecloud.de",
			apiDomain:   "api.scalecloud.de",
			proxyIP:     "127.0.0.1",
			wantError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("APP_ENV", test.environment)
			t.Setenv("FRONTEND_URL", test.frontendURL)
			t.Setenv("API_DOMAIN", test.apiDomain)
			t.Setenv("TRUSTED_PROXY_IP", test.proxyIP)
			t.Setenv("LOG_LEVEL", test.logLevel)

			config, err := LoadFromEnv()
			if (err != nil) != test.wantError {
				t.Fatalf("LoadFromEnv() error = %v, wantError %t", err, test.wantError)
			}
			if err == nil && config.Environment.String() != test.environment {
				t.Fatalf("environment = %q, want %q", config.Environment, test.environment)
			}
			if err == nil && config.LogLevel != test.wantLogLevel {
				t.Fatalf("log level = %q, want %q", config.LogLevel, test.wantLogLevel)
			}
		})
	}
}
