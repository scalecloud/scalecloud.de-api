package emailmanager

import (
	"testing"

	"go.uber.org/zap"
)

func setValidSMTPEnvironment(test *testing.T) {
	test.Helper()
	test.Setenv("SMTP_HOST", "smtp.example.com")
	test.Setenv("SMTP_PORT", "587")
	test.Setenv("SMTP_USERNAME", "smtp-user")
	test.Setenv("SMTP_PASSWORD", "smtp-password")
	test.Setenv("SMTP_FROM", "noreply@example.com")
}

func TestReadSMTPCredentials(t *testing.T) {
	setValidSMTPEnvironment(t)

	credentials, err := readSMTPCredentials(zap.NewNop())
	if err != nil {
		t.Fatalf("read valid SMTP credentials: %v", err)
	}
	if credentials.Host != "smtp.example.com" || credentials.Port != 587 || credentials.Username != "smtp-user" || credentials.Password != "smtp-password" || credentials.From != "noreply@example.com" {
		t.Fatalf("unexpected parsed SMTP credentials: %+v", credentials)
	}
}

func TestReadSMTPCredentialsRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		setInvalid func(*testing.T)
	}{
		{
			name: "invalid port",
			setInvalid: func(test *testing.T) {
				test.Setenv("SMTP_PORT", "not-a-port")
			},
		},
		{
			name: "missing host",
			setInvalid: func(test *testing.T) {
				test.Setenv("SMTP_HOST", " ")
			},
		},
		{
			name: "invalid sender address",
			setInvalid: func(test *testing.T) {
				test.Setenv("SMTP_FROM", "not-an-email")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(test *testing.T) {
			setValidSMTPEnvironment(test)
			testCase.setInvalid(test)
			if _, err := readSMTPCredentials(zap.NewNop()); err == nil {
				test.Fatal("expected invalid SMTP configuration to fail")
			}
		})
	}
}

func TestInitEMailConnectionBuildsClientsWithoutConnecting(t *testing.T) {
	setValidSMTPEnvironment(t)

	connection, err := InitEMailConnection(zap.NewNop())
	if err != nil {
		t.Fatalf("initialize SMTP clients: %v", err)
	}
	if connection.Client == nil || connection.healthClient == nil {
		t.Fatal("expected send and health clients to be initialized")
	}
	if connection.From != "noreply@example.com" {
		t.Fatalf("unexpected sender address: %q", connection.From)
	}
}
