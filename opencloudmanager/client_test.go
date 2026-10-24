package opencloudmanager

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckHealthUsesAppTokenBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, token, ok := request.BasicAuth()
		if !ok || username != "billing-service" || token != "app-secret" {
			t.Errorf("unexpected credentials: %q %q", username, token)
		}
		if request.Method != http.MethodGet || request.URL.Path != "/graph/v1.0/me" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckHealth(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRequestMapsAuthorizationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte(`{"error":{"code":"accessDenied","message":"not allowed"}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckHealth(context.Background()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestRequestReturnsBodyCloseError(t *testing.T) {
	closeErr := errors.New("response body close failed")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"id":"me"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret", HTTP: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		resp.Body = &closeTrackingReader{ReadCloser: resp.Body, closeErr: closeErr}
		return resp, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}

	err = client.CheckHealth(context.Background())
	if err == nil || !strings.Contains(err.Error(), "close OpenCloud response body") {
		t.Fatalf("CheckHealth() error = %v, want close response body error", err)
	}
	if !strings.Contains(err.Error(), closeErr.Error()) {
		t.Fatalf("CheckHealth() error = %v, want original close error %q", err, closeErr)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeTrackingReader struct {
	io.ReadCloser
	closeErr error
}

func (r *closeTrackingReader) Close() error {
	if err := r.ReadCloser.Close(); err != nil {
		return err
	}
	return r.closeErr
}

func TestGetSpaceUsagePreservesMissingQuotaValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"id":"drive-1","quota":{"state":"normal"}}`))
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}
	usage, err := client.GetSpaceUsage(context.Background(), "drive-1")
	if err != nil {
		t.Fatal(err)
	}
	if usage.Used != nil || usage.Total != nil || usage.Remaining != nil {
		t.Fatalf("missing quota values should remain unknown: %+v", usage)
	}
}

func TestGetUserDriveUsageUsesGraphPersonalDriveEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/graph/v1.0/users/user-1/drive" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"id":"personal-drive-1","driveType":"personal","quota":{"used":24,"total":100,"remaining":76,"state":"normal"}}`))
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}
	usage, err := client.GetUserDriveUsage(context.Background(), "user-1")
	if err != nil || usage.Used == nil || *usage.Used != 24 || usage.Total == nil || *usage.Total != 100 {
		t.Fatalf("GetUserDriveUsage() = %+v, %v", usage, err)
	}
}

func TestNewClientFromEnvIsOptionalButRejectsPartialSettings(t *testing.T) {
	t.Setenv("OPENCLOUD_BASE_URL", "")
	t.Setenv("OPENCLOUD_USERNAME", "")
	t.Setenv("OPENCLOUD_APP_TOKEN", "")
	client, configured, err := NewClientFromEnv()
	if err != nil || configured || client != nil {
		t.Fatalf("empty environment = (%v, %t, %v)", client, configured, err)
	}

	t.Setenv("OPENCLOUD_BASE_URL", "https://cloud.example")
	if _, _, err := NewClientFromEnv(); !errors.Is(err, ErrMissingSetting) {
		t.Fatalf("partial environment error = %v, want ErrMissingSetting", err)
	}
}

func TestCreateSpaceAndSetQuotaUseProjectDriveRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		switch request.Method {
		case http.MethodPost:
			if request.URL.Path != "/graph/v1.0/drives" || body["driveType"] != "project" {
				t.Errorf("unexpected create request: %s %s body=%v", request.Method, request.URL.Path, body)
			}
			quota := body["quota"].(map[string]any)
			if quota["total"] != float64(1000000000) {
				t.Errorf("create quota = %v", quota["total"])
			}
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"id":"drive-1","driveType":"project","quota":{"total":1000000000,"used":12,"remaining":999999988,"state":"normal"}}`))
		case http.MethodPatch:
			if request.URL.Path != "/graph/v1.0/drives/drive-1" {
				t.Errorf("unexpected quota path: %s", request.URL.Path)
			}
			quota := body["quota"].(map[string]any)
			if quota["total"] != float64(2000000000) {
				t.Errorf("patched quota = %v", quota["total"])
			}
			_, _ = writer.Write([]byte(`{"id":"drive-1","quota":{"total":2000000000,"used":12,"remaining":1999999988,"state":"normal"}}`))
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}
	space, err := client.CreateSpace(context.Background(), "Subscription 1", "subscription:sub_1", 1000000000)
	if err != nil || space.ID != "drive-1" || space.Quota == nil || space.Quota.Total == nil || *space.Quota.Total != 1000000000 {
		t.Fatalf("CreateSpace() = %+v, %v", space, err)
	}
	space, err = client.SetSpaceQuota(context.Background(), space.ID, 2000000000)
	if err != nil || space.Quota == nil || space.Quota.Total == nil || *space.Quota.Total != 2000000000 {
		t.Fatalf("SetSpaceQuota() = %+v, %v", space, err)
	}
}

func TestEnsureMemberInvitesOnlyWhenMissing(t *testing.T) {
	permissionsCalls := 0
	inviteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/root/permissions"):
			permissionsCalls++
			if permissionsCalls == 1 {
				_, _ = writer.Write([]byte(`{"value":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"value":[{"id":"permission-1","roles":["editor"],"grantedToV2":{"user":{"id":"user-1","displayName":"A"}}}]}`))
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/root/invite"):
			inviteCalls++
			var body struct {
				Recipients []struct {
					ObjectID string `json:"objectId"`
					Type     string `json:"@libre.graph.recipient.type"`
				} `json:"recipients"`
				Roles []string `json:"roles"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode invite body: %v", err)
			}
			if len(body.Recipients) != 1 || body.Recipients[0].ObjectID != "user-1" || body.Recipients[0].Type != "user" || len(body.Roles) != 1 || body.Roles[0] != "58c63c02-1d89-4572-916a-870abc5a1b7d" {
				t.Errorf("unexpected invite body: %+v", body)
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}
	added, err := client.EnsureMember(context.Background(), "drive-1", "user-1", SpaceRoleEditor)
	if err != nil || !added {
		t.Fatalf("first EnsureMember() = %t, %v", added, err)
	}
	added, err = client.EnsureMember(context.Background(), "drive-1", "user-1", SpaceRoleEditor)
	if err != nil || added || inviteCalls != 1 {
		t.Fatalf("second EnsureMember() = %t, %v; invite calls = %d", added, err, inviteCalls)
	}
}

func TestListProjectSpacesAndUsersUsesGraphCollectionEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/graph/v1.0/drives":
			if request.URL.Query().Get("$filter") != "driveType eq 'project'" {
				t.Fatalf("unexpected drives filter %q", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"value":[{"id":"drive-1","driveType":"project","description":"subscription:sub-1"},{"id":"drive-2","driveType":"project","description":"subscription:sub-2"}]}`))
		case "/graph/v1.0/users":
			_, _ = writer.Write([]byte(`{"value":[{"id":"user-1","displayName":"Alice","mail":"alice@example.com"},{"id":"user-2","displayName":"Bob","mail":"bob@example.com"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Username: "billing-service", AppToken: "app-secret"})
	if err != nil {
		t.Fatal(err)
	}

	spaces, err := client.ListProjectSpaces(context.Background())
	if err != nil {
		t.Fatalf("ListProjectSpaces() error = %v", err)
	}
	if len(spaces) != 2 || spaces[0].ID != "drive-1" || spaces[1].ID != "drive-2" {
		t.Fatalf("ListProjectSpaces() = %+v", spaces)
	}

	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 || users[0].ID != "user-1" || users[1].ID != "user-2" {
		t.Fatalf("ListUsers() = %+v", users)
	}
}
