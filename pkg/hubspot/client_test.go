package hubspot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	betaUsersPath   = "/settings/users/2026-09-beta"
	stableUsersPath = "/settings/users/2026-03"
)

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient("test-token", http.DefaultClient, baseURL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func TestGetUsersFallsBackToStableEndpoint(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case betaUsersPath:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		case stableUsersPath:
			_, _ = w.Write([]byte(`{"results":[{"id":"1","email":"user@example.com"}]}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	users, nextPage, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 50})
	if err != nil {
		t.Fatalf("GetUsers: %v", err)
	}
	if len(users) != 1 || users[0].Email != "user@example.com" {
		t.Fatalf("got users %+v, want a single user@example.com", users)
	}
	if nextPage != "" {
		t.Errorf("got next page %q, want empty (stable endpoint has no paging)", nextPage)
	}
	if want := []string{betaUsersPath, stableUsersPath}; !equal(paths, want) {
		t.Fatalf("requested paths %v, want %v", paths, want)
	}

	// The fallback is sticky: the beta endpoint isn't probed again. A different
	// limit is used so the request isn't served from the uhttp response cache.
	if !client.listUsersFallback.Load() {
		t.Error("listUsersFallback not set after falling back")
	}
	paths = nil
	if _, _, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 25}); err != nil {
		t.Fatalf("GetUsers (second call): %v", err)
	}
	if want := []string{stableUsersPath}; !equal(paths, want) {
		t.Fatalf("requested paths %v, want %v", paths, want)
	}
}

func TestGetUsersUsesBetaEndpointWhenAvailable(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != betaUsersPath {
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"1","email":"user@example.com"}],"paging":{"next":{"after":"50"}}}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	users, nextPage, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 50})
	if err != nil {
		t.Fatalf("GetUsers: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	if nextPage != "50" {
		t.Errorf("got next page %q, want %q", nextPage, "50")
	}
	if want := []string{betaUsersPath}; !equal(paths, want) {
		t.Fatalf("requested paths %v, want %v", paths, want)
	}
}

func TestGetUsersDoesNotFallBackOnAuthError(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid token"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	_, _, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 50})
	if err == nil {
		t.Fatal("GetUsers: got nil error, want an error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("got error %v, want it to surface the 401", err)
	}
	if want := []string{betaUsersPath}; !equal(paths, want) {
		t.Fatalf("requested paths %v, want %v", paths, want)
	}
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
