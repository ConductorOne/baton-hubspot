package hubspot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// newTestClient spins up a server that always replies with body and returns a
// client pointed at it. Each case gets its own server so uhttp's response cache
// never serves one case's body to another.
func newTestClient(t *testing.T, body string) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient("token", server.Client(), server.URL+"/")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestGetUsersReturnsNextPage(t *testing.T) {
	client := newTestClient(t, `{"results":[{"id":"1","email":"a@example.com"}],"paging":{"next":{"after":"50"}}}`)

	users, nextPage, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 50})
	if err != nil {
		t.Fatalf("GetUsers: %v", err)
	}
	if len(users) != 1 || users[0].Id != "1" {
		t.Errorf("got users %+v, want a single user with id 1", users)
	}
	if nextPage != "50" {
		t.Errorf("got next page %q, want %q", nextPage, "50")
	}
}

// The last page still carries a paging object, just without a cursor.
func TestGetUsersLastPage(t *testing.T) {
	client := newTestClient(t, `{"results":[{"id":"1","email":"a@example.com"}],"paging":{}}`)

	users, nextPage, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 50})
	if err != nil {
		t.Fatalf("GetUsers: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("got %d users, want 1", len(users))
	}
	if nextPage != "" {
		t.Errorf("got next page %q, want empty", nextPage)
	}
}

// A 200 with results but no paging object means the API truncated the list
// without telling us. That must be an error, not a short sync.
func TestGetUsersMissingPagingErrors(t *testing.T) {
	client := newTestClient(t, `{"results":[{"id":"1","email":"a@example.com"}]}`)

	_, _, _, err := client.GetUsers(context.Background(), GetUsersVars{Limit: 50})
	if err == nil {
		t.Fatal("GetUsers: expected an error when the response omits paging")
	}
	if !errors.Is(err, uhttp.ErrMissingPaginationData) {
		t.Errorf("got error %v, want it to wrap ErrMissingPaginationData", err)
	}
}
