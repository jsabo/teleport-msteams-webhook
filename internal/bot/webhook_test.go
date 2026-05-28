package bot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func testCard() teamsMessage {
	return BuildCard("req-test", RequestData{User: "alice", Roles: []string{"dev"}}, nil, "")
}

func TestPostCard_success(t *testing.T) {
	var received int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&received, 1)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := postCard(context.Background(), srv.Client(), srv.URL, testCard())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("server received %d requests, want 1", received)
	}
}

func TestPostCard_retryOn5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := postCard(context.Background(), srv.Client(), srv.URL, testCard())
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("server called %d times, want 2", calls)
	}
}

func TestPostCard_exhaustedRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	err := postCard(context.Background(), srv.Client(), srv.URL, testCard())
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
}

func TestPostCard_4xxNoRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := postCard(context.Background(), srv.Client(), srv.URL, testCard())
	if err == nil {
		t.Fatal("expected error on 4xx")
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("server called %d times, want 1 (no retry on 4xx)", calls)
	}
}

func TestPostCard_contextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := postCard(ctx, srv.Client(), srv.URL, testCard())
	if err == nil {
		t.Fatal("expected error when context cancelled")
	}
}
