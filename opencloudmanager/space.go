package opencloudmanager

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type SpaceQuota struct {
	Total     int64  `json:"total"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	State     string `json:"state"`
}

type Space struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	DriveType   string      `json:"driveType"`
	WebURL      string      `json:"webUrl"`
	Quota       *SpaceQuota `json:"quota"`
	Root        *struct {
		Deleted *struct {
			State string `json:"state"`
		} `json:"deleted"`
	} `json:"root"`
}

func (space Space) Disabled() bool {
	return space.Root != nil && space.Root.Deleted != nil && space.Root.Deleted.State == "trashed"
}

type SpaceUsage struct {
	Used      int64  `json:"used"`
	Total     int64  `json:"total"`
	Remaining int64  `json:"remaining"`
	State     string `json:"state"`
	Disabled  bool   `json:"disabled"`
}

func (client *Client) CreateSpace(ctx context.Context, name, description string, quotaBytes int64) (*Space, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("OpenCloud space name is required")
	}
	if quotaBytes <= 0 {
		return nil, ErrInvalidQuota
	}
	input := struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		DriveType   string `json:"driveType"`
		Quota       struct {
			Total int64 `json:"total"`
		} `json:"quota"`
	}{Name: name, Description: description, DriveType: "project"}
	input.Quota.Total = quotaBytes
	var space Space
	if err := client.request(ctx, http.MethodPost, "/graph/v1.0/drives", input, &space); err != nil {
		return nil, err
	}
	return &space, nil
}

func (client *Client) GetSpace(ctx context.Context, driveID string) (*Space, error) {
	if strings.TrimSpace(driveID) == "" {
		return nil, errors.New("OpenCloud drive ID is required")
	}
	var space Space
	path := "/graph/v1.0/drives/" + url.PathEscape(driveID)
	if err := client.request(ctx, http.MethodGet, path, nil, &space); err != nil {
		return nil, err
	}
	return &space, nil
}

func (client *Client) SetSpaceQuota(ctx context.Context, driveID string, quotaBytes int64) (*Space, error) {
	if quotaBytes <= 0 {
		return nil, ErrInvalidQuota
	}
	input := struct {
		Quota struct {
			Total int64 `json:"total"`
		} `json:"quota"`
	}{}
	input.Quota.Total = quotaBytes
	var space Space
	path := "/graph/v1.0/drives/" + url.PathEscape(driveID)
	if err := client.request(ctx, http.MethodPatch, path, input, &space); err != nil {
		return nil, err
	}
	return &space, nil
}

func (client *Client) GetSpaceUsage(ctx context.Context, driveID string) (*SpaceUsage, error) {
	space, err := client.GetSpace(ctx, driveID)
	if err != nil {
		return nil, err
	}
	usage := &SpaceUsage{Disabled: space.Disabled()}
	if space.Quota != nil {
		usage.Total = space.Quota.Total
		usage.Used = space.Quota.Used
		usage.Remaining = space.Quota.Remaining
		usage.State = space.Quota.State
	}
	return usage, nil
}

func (client *Client) DisableSpace(ctx context.Context, driveID string) error {
	if strings.TrimSpace(driveID) == "" {
		return errors.New("OpenCloud drive ID is required")
	}
	path := "/graph/v1.0/drives/" + url.PathEscape(driveID)
	return client.request(ctx, http.MethodDelete, path, nil, nil)
}

func (client *Client) RestoreSpace(ctx context.Context, driveID string) (*Space, error) {
	if strings.TrimSpace(driveID) == "" {
		return nil, errors.New("OpenCloud drive ID is required")
	}
	path := "/graph/v1.0/drives/" + url.PathEscape(driveID)
	var space Space
	if err := client.requestWithHeaders(ctx, http.MethodPatch, path, map[string]any{}, &space, map[string]string{"restore": "true"}); err != nil {
		return nil, err
	}
	return &space, nil
}

func (client *Client) FindProjectSpace(ctx context.Context, marker string) (*Space, error) {
	if strings.TrimSpace(marker) == "" {
		return nil, errors.New("OpenCloud space marker is required")
	}
	var response struct {
		Value []Space `json:"value"`
	}
	if err := client.request(ctx, http.MethodGet, "/graph/v1.0/drives", nil, &response); err != nil {
		return nil, err
	}
	for index := range response.Value {
		space := &response.Value[index]
		if space.DriveType == "project" && strings.Contains(space.Description, marker) {
			return space, nil
		}
	}
	return nil, ErrNotFound
}
