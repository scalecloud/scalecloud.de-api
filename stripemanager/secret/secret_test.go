package secret

import (
	"testing"

	"go.uber.org/zap"
)

func TestGetStripeKey(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "trims whitespace", value: "  sk_test_example  ", want: "sk_test_example"},
		{name: "empty value", value: "", wantErr: true},
		{name: "whitespace only", value: " \t\n ", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("STRIPE_SECRET_KEY", test.value)

			got, err := GetStripeKey()
			if (err != nil) != test.wantErr {
				t.Fatalf("GetStripeKey() error = %v, wantErr %t", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("GetStripeKey() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetStripeEndpointSecret(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "trims whitespace", value: "  whsec_example  ", want: "whsec_example"},
		{name: "empty value", value: "", wantErr: true},
		{name: "whitespace only", value: " \t\n ", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("STRIPE_ENDPOINT_SECRET", test.value)

			got, err := GetStripeEndpointSecret()
			if (err != nil) != test.wantErr {
				t.Fatalf("GetStripeEndpointSecret() error = %v, wantErr %t", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("GetStripeEndpointSecret() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCheckStripeConfig(t *testing.T) {
	tests := []struct {
		name           string
		secretKey      string
		endpointSecret string
		wantErr        string
	}{
		{name: "configured", secretKey: "sk_test_example", endpointSecret: "whsec_example"},
		{name: "missing API key", endpointSecret: "whsec_example", wantErr: "STRIPE_SECRET_KEY is empty"},
		{name: "blank API key", secretKey: " \t ", endpointSecret: "whsec_example", wantErr: "STRIPE_SECRET_KEY is empty"},
		{name: "missing endpoint secret", secretKey: "sk_test_example", wantErr: "STRIPE_ENDPOINT_SECRET is empty"},
		{name: "blank endpoint secret", secretKey: "sk_test_example", endpointSecret: " \t ", wantErr: "STRIPE_ENDPOINT_SECRET is empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("STRIPE_SECRET_KEY", test.secretKey)
			t.Setenv("STRIPE_ENDPOINT_SECRET", test.endpointSecret)

			err := CheckStripeConfig(zap.NewNop())
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("CheckStripeConfig() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErr {
				t.Fatalf("CheckStripeConfig() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}
