package bot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBot_Post_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := &Bot{httpClient: srv.Client()}
	err := b.Post(context.Background(), srv.URL, "req-1", RequestData{User: "alice", Roles: []string{"dev"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBot_Post_failsOn4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	b := &Bot{httpClient: srv.Client()}
	err := b.Post(context.Background(), srv.URL, "req-2", RequestData{User: "bob", Roles: []string{"admin"}})
	if err == nil {
		t.Fatal("expected error on 404")
	}
}
