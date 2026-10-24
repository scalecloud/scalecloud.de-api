package opencloudmanager

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Mail        string `json:"mail"`
	Username    string `json:"onPremisesSamAccountName"`
}

func (client *Client) ListUsers(ctx context.Context) ([]User, error) {
	var response struct {
		Value []User `json:"value"`
	}
	if err := client.request(ctx, http.MethodGet, "/graph/v1.0/users", nil, &response); err != nil {
		return nil, err
	}
	return response.Value, nil
}

func (client *Client) GetUser(ctx context.Context, idOrUsername string) (*User, error) {
	if strings.TrimSpace(idOrUsername) == "" {
		return nil, errors.New("OpenCloud user ID or username is required")
	}
	var user User
	path := "/graph/v1.0/users/" + url.PathEscape(idOrUsername)
	if err := client.request(ctx, http.MethodGet, path, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}
