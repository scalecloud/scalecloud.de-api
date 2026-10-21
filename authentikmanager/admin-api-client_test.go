package authentikmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFindUserByEmailReturnsEmptyUserWhenNoMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{"results": []User{}})
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/api/v3/")
	if err != nil {
		t.Fatal(err)
	}
	client := NewAdminAPIClient(baseURL, "test-api-token", server.Client())

	user, err := client.FindUserByEmail(context.Background(), "missing@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail() error = %v, want nil", err)
	}
	if user != (User{}) {
		t.Fatalf("FindUserByEmail() = %+v, want empty user", user)
	}
}

func TestAddUserToSubscriptionCreatesGroupAndAddsUser(t *testing.T) {
	fixture := &subscriptionTestFixture{groupPK: "a4c24298-111b-49a9-a72a-625f8eca49bf"}
	server := httptest.NewServer(fixture.handler(t))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/api/v3/")
	if err != nil {
		t.Fatal(err)
	}
	client := NewAdminAPIClient(baseURL, "test-api-token", server.Client())
	if err := client.AddUserToSubscription(context.Background(), "sub_test", "invite@example.com"); err != nil {
		t.Fatal(err)
	}
	if !fixture.groupCreated || !fixture.membershipAdded {
		t.Fatalf("expected group creation and membership add, got group=%t membership=%t", fixture.groupCreated, fixture.membershipAdded)
	}
}

func TestCheckHealthDecodesUUIDGroupPK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/core/users/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{"results": []User{}})
	})
	mux.HandleFunc("GET /api/v3/core/groups/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{"results": []Group{{PK: "a4c24298-111b-49a9-a72a-625f8eca49bf", Name: "scalecloud-subscription-sub_test"}}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/api/v3/")
	if err != nil {
		t.Fatal(err)
	}
	client := NewAdminAPIClient(baseURL, "test-api-token", server.Client())
	if err := client.CheckHealth(context.Background()); err != nil {
		t.Fatalf("CheckHealth() with UUID group PK: %v", err)
	}
}

type subscriptionTestFixture struct {
	groupPK         string
	groupCreated    bool
	membershipAdded bool
}

func (fixture *subscriptionTestFixture) handler(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/core/users/", fixture.findUser(t))
	mux.HandleFunc("GET /api/v3/core/groups/", fixture.findGroup(t))
	mux.HandleFunc("POST /api/v3/core/groups/", fixture.createGroup(t))
	mux.HandleFunc("POST /api/v3/core/groups/"+fixture.groupPK+"/add_user/", fixture.addUser(t))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-token" {
			t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		mux.ServeHTTP(w, r)
	})
}

func (fixture *subscriptionTestFixture) findUser(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "invite@example.com" {
			t.Errorf("unexpected user search: %q", r.URL.Query().Get("search"))
		}
		writeJSON(t, w, map[string]any{"results": []User{{PK: 42, Email: "invite@example.com", IsActive: true}}})
	}
}

func (fixture *subscriptionTestFixture) findGroup(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		groups := []Group{}
		if fixture.groupCreated {
			groups = append(groups, Group{PK: fixture.groupPK, Name: "scalecloud-subscription-sub_test"})
		}
		writeJSON(t, w, map[string]any{"results": groups})
	}
}

func (fixture *subscriptionTestFixture) createGroup(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode create-group request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request["name"] != "scalecloud-subscription-sub_test" {
			t.Errorf("unexpected group name: %q", request["name"])
		}
		fixture.groupCreated = true
		writeJSON(t, w, Group{PK: fixture.groupPK, Name: request["name"]})
	}
}

func (fixture *subscriptionTestFixture) addUser(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request map[string]int64
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode add-user request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request["pk"] != 42 {
			t.Errorf("unexpected user pk: %d", request["pk"])
		}
		fixture.membershipAdded = true
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response JSON: %v", err)
	}
}
