package authentikmanager

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

type AdminAPIClient struct {
	baseURL    *url.URL
	apiToken   string
	httpClient *http.Client
}

var ErrUserNotFound = errors.New("Authentik user not found")

type User struct {
	PK       int64  `json:"pk"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}

type Group struct {
	PK   int64  `json:"pk"`
	Name string `json:"name"`
}

type listResponse[T any] struct {
	Results []T `json:"results"`
}

func NewAdminAPIClientFromEnv() (*AdminAPIClient, error) {
	apiURL := strings.TrimRight(os.Getenv("AUTHENTIK_API_URL"), "/")
	apiToken := os.Getenv("AUTHENTIK_API_TOKEN")
	if apiURL == "" {
		return nil, errors.New("AUTHENTIK_API_URL is empty")
	}
	if apiToken == "" {
		return nil, errors.New("AUTHENTIK_API_TOKEN is empty")
	}

	parsedURL, err := url.Parse(apiURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("AUTHENTIK_API_URL must be an absolute HTTPS URL without credentials or query parameters")
	}

	return NewAdminAPIClient(parsedURL, apiToken, &http.Client{Timeout: 10 * time.Second}), nil
}

func NewAdminAPIClient(baseURL *url.URL, apiToken string, httpClient *http.Client) *AdminAPIClient {
	base := *baseURL
	base.Path = strings.TrimRight(base.Path, "/") + "/"
	return &AdminAPIClient{baseURL: &base, apiToken: apiToken, httpClient: httpClient}
}

func (client *AdminAPIClient) AddUserToSubscription(ctx context.Context, subscriptionID, email string) error {
	user, err := client.FindUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if !user.IsActive {
		return errors.New("Authentik user is inactive")
	}
	return client.AddUserPKToSubscription(ctx, subscriptionID, user.PK)
}

func (client *AdminAPIClient) FindUserByEmail(ctx context.Context, email string) (User, error) {
	return client.findUserByEmail(ctx, email)
}

func (client *AdminAPIClient) AddUserPKToSubscription(ctx context.Context, subscriptionID string, userPK int64) error {
	group, err := client.ensureSubscriptionGroup(ctx, subscriptionID)
	if err != nil {
		return err
	}
	return client.changeGroupMembership(ctx, group.PK, userPK, "add_user")
}

func (client *AdminAPIClient) RemoveUserFromSubscription(ctx context.Context, subscriptionID string, userPK int64) error {
	group, err := client.findSubscriptionGroup(ctx, subscriptionID)
	if err != nil {
		return err
	}
	if group == nil {
		return nil
	}
	return client.changeGroupMembership(ctx, group.PK, userPK, "remove_user")
}

func (client *AdminAPIClient) findUserByEmail(ctx context.Context, email string) (User, error) {
	query := url.Values{}
	query.Set("search", email)
	query.Set("page_size", "100")
	query.Set("include_groups", "false")
	query.Set("include_roles", "false")
	var response listResponse[User]
	if err := client.requestJSON(ctx, http.MethodGet, "core/users/?"+query.Encode(), nil, &response); err != nil {
		return User{}, err
	}

	normalizedEmail := normalizeEmail(email)
	var match *User
	for index := range response.Results {
		user := &response.Results[index]
		if normalizeEmail(user.Email) != normalizedEmail {
			continue
		}
		if match != nil {
			return User{}, errors.New("multiple Authentik users have the invited email")
		}
		match = user
	}
	if match == nil {
		return User{}, ErrUserNotFound
	}
	return *match, nil
}

func (client *AdminAPIClient) ensureSubscriptionGroup(ctx context.Context, subscriptionID string) (Group, error) {
	group, err := client.findSubscriptionGroup(ctx, subscriptionID)
	if err != nil {
		return Group{}, err
	}
	if group != nil {
		return *group, nil
	}

	name := subscriptionGroupName(subscriptionID)
	var created Group
	err = client.requestJSON(ctx, http.MethodPost, "core/groups/", map[string]string{"name": name}, &created)
	if err == nil {
		return created, nil
	}

	// A concurrent claim may have created the deterministic group first.
	if existing, lookupErr := client.findSubscriptionGroup(ctx, subscriptionID); lookupErr == nil && existing != nil {
		return *existing, nil
	}
	return Group{}, err
}

func (client *AdminAPIClient) findSubscriptionGroup(ctx context.Context, subscriptionID string) (*Group, error) {
	query := url.Values{}
	query.Set("search", subscriptionGroupName(subscriptionID))
	query.Set("page_size", "100")
	query.Set("include_users", "false")
	query.Set("include_children", "false")
	query.Set("include_parents", "false")
	var response listResponse[Group]
	if err := client.requestJSON(ctx, http.MethodGet, "core/groups/?"+query.Encode(), nil, &response); err != nil {
		return nil, err
	}
	name := subscriptionGroupName(subscriptionID)
	for index := range response.Results {
		if response.Results[index].Name == name {
			return &response.Results[index], nil
		}
	}
	return nil, nil
}

func (client *AdminAPIClient) changeGroupMembership(ctx context.Context, groupPK, userPK int64, action string) error {
	if action != "add_user" && action != "remove_user" {
		return errors.New("invalid group membership action")
	}
	path := fmt.Sprintf("core/groups/%d/%s/", groupPK, action)
	return client.requestJSON(ctx, http.MethodPost, path, map[string]int64{"pk": userPK}, nil)
}

func (client *AdminAPIClient) requestJSON(ctx context.Context, method, path string, requestBody, responseBody any) error {
	endpoint := *client.baseURL
	relative, err := url.Parse(path)
	if err != nil {
		return err
	}
	endpoint = *endpoint.ResolveReference(relative)

	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+client.apiToken)
	request.Header.Set("Accept", "application/json")
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("Authentik API request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Authentik API returned HTTP %d for %s", response.StatusCode, path)
	}
	if responseBody == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(responseBody); err != nil {
		return fmt.Errorf("decode Authentik API response: %w", err)
	}
	return nil
}

func subscriptionGroupName(subscriptionID string) string {
	return "scalecloud-subscription-" + subscriptionID
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
