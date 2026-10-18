package authentikmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAddUserToSubscriptionCreatesGroupAndAddsUser(t *testing.T) {
	groupCreated := false
	membershipAdded := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-token" {
			t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}

		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v3/core/users/"):
			if r.URL.Query().Get("search") != "invite@example.com" {
				t.Errorf("unexpected user search: %q", r.URL.Query().Get("search"))
			}
			json.NewEncoder(w).Encode(map[string]any{"results": []User{{PK: 42, Email: "invite@example.com", IsActive: true}}})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v3/core/groups/"):
			groups := []Group{}
			if groupCreated {
				groups = append(groups, Group{PK: 7, Name: "scalecloud-subscription-sub_test"})
			}
			json.NewEncoder(w).Encode(map[string]any{"results": groups})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/core/groups/":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			if request["name"] != "scalecloud-subscription-sub_test" {
				t.Errorf("unexpected group name: %q", request["name"])
			}
			groupCreated = true
			json.NewEncoder(w).Encode(Group{PK: 7, Name: request["name"]})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/core/groups/7/add_user/":
			var request map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			if request["pk"] != 42 {
				t.Errorf("unexpected user pk: %d", request["pk"])
			}
			membershipAdded = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/api/v3/")
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(baseURL, "test-api-token", server.Client())
	if err := client.AddUserToSubscription(context.Background(), "sub_test", "invite@example.com"); err != nil {
		t.Fatal(err)
	}
	if !groupCreated || !membershipAdded {
		t.Fatalf("expected group creation and membership add, got group=%t membership=%t", groupCreated, membershipAdded)
	}
}
