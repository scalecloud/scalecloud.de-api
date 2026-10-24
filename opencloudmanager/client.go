package opencloudmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrNotFound       = errors.New("opencloud resource not found")
	ErrUnauthorized   = errors.New("OpenCloud credentials rejected")
	ErrForbidden      = errors.New("OpenCloud permission denied")
	ErrNotMember      = errors.New("user is not a member of the OpenCloud space")
	ErrInvalidQuota   = errors.New("OpenCloud quota must be greater than zero")
	ErrMissingSetting = errors.New("OpenCloud URL, username, and app token are required")
)

type Config struct {
	BaseURL  string
	Username string
	AppToken string
	Roles    Roles
	HTTP     *http.Client
}

type Roles struct {
	Viewer  string
	Editor  string
	Manager string
}

type Client struct {
	baseURL  string
	username string
	appToken string
	roles    Roles
	http     *http.Client
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (err *APIError) Error() string {
	return fmt.Sprintf("OpenCloud API returned HTTP %d: %s", err.Status, err.Message)
}

func (err *APIError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return err.Status == http.StatusNotFound
	case ErrUnauthorized:
		return err.Status == http.StatusUnauthorized
	case ErrForbidden:
		return err.Status == http.StatusForbidden
	default:
		return false
	}
}

func NewClient(config Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(config.Username) == "" || config.AppToken == "" {
		return nil, ErrMissingSetting
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("OpenCloud base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	httpClient := config.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	if config.Roles.Viewer == "" {
		config.Roles.Viewer = "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
	}
	if config.Roles.Editor == "" {
		config.Roles.Editor = "58c63c02-1d89-4572-916a-870abc5a1b7d"
	}
	if config.Roles.Manager == "" {
		config.Roles.Manager = "312c0871-5ef7-4b3a-85b6-0e4074c64049"
	}
	return &Client{baseURL: baseURL, username: config.Username, appToken: config.AppToken, roles: config.Roles, http: httpClient}, nil
}

func NewClientFromEnv() (*Client, bool, error) {
	baseURL := strings.TrimSpace(os.Getenv("OPENCLOUD_BASE_URL"))
	username := strings.TrimSpace(os.Getenv("OPENCLOUD_USERNAME"))
	appToken := strings.TrimSpace(os.Getenv("OPENCLOUD_APP_TOKEN"))
	if baseURL == "" && username == "" && appToken == "" {
		return nil, false, nil
	}
	if baseURL == "" || username == "" || appToken == "" {
		return nil, false, ErrMissingSetting
	}
	client, err := NewClient(Config{
		BaseURL:  baseURL,
		Username: username,
		AppToken: appToken,
		Roles: Roles{
			Viewer:  os.Getenv("OPENCLOUD_ROLE_VIEWER"),
			Editor:  os.Getenv("OPENCLOUD_ROLE_EDITOR"),
			Manager: os.Getenv("OPENCLOUD_ROLE_MANAGER"),
		},
	})
	if err != nil {
		return nil, false, err
	}
	return client, true, nil
}

func (client *Client) CheckHealth(ctx context.Context) error {
	var user User
	return client.request(ctx, http.MethodGet, "/graph/v1.0/me", nil, &user)
}

func (client *Client) request(ctx context.Context, method, path string, input, output any) error {
	return client.requestWithHeaders(ctx, method, path, input, output, nil)
}

func (client *Client) requestWithHeaders(ctx context.Context, method, path string, input, output any, headers map[string]string) (err error) {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode OpenCloud request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create OpenCloud request: %w", err)
	}
	request.SetBasicAuth(client.username, client.appToken)
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("send OpenCloud request: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close OpenCloud response body: %w", closeErr)
		}
	}()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read OpenCloud response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var envelope struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(responseBody, &envelope)
		message := envelope.Error.Message
		if message == "" {
			message = strings.TrimSpace(string(responseBody))
		}
		return &APIError{Status: response.StatusCode, Code: envelope.Error.Code, Message: message}
	}
	if output != nil && len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, output); err != nil {
			return fmt.Errorf("decode OpenCloud response: %w", err)
		}
	}
	return nil
}
