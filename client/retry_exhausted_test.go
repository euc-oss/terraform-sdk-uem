package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func newRetryTestClient(t *testing.T, url string, maxRetries int) *Client {
	t.Helper()
	c, err := NewClient(&Config{InstanceURL: url, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p", MaxRetries: maxRetries})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// A persistent 500 exhausts the retry policy. The final response must still
// reach handleResponse, so the caller gets an *APIError with the server's
// status, errorCode and message, plus the number of attempts made.
func TestRetryExhausted_Persistent500IsAPIError(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorCode":1000,"message":"Internal Server Error","activityId":"x"}`))
	}))
	t.Cleanup(srv.Close)
	c := newRetryTestClient(t, srv.URL, 1)

	hdr, err := c.DoRequest(context.Background(), "GET", "/api/mdm/scripts/x", "application/json;version=1", "application/json", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != 500 || apiErr.ErrorCode != "1000" || apiErr.Message != "Internal Server Error" {
		t.Errorf("got status=%d errorCode=%q message=%q, want 500/1000/Internal Server Error", apiErr.StatusCode, apiErr.ErrorCode, apiErr.Message)
	}
	if got := atomic.LoadInt32(&hits); got != 2 || apiErr.Attempts != 2 {
		t.Errorf("server hits=%d, Attempts=%d; want both 2 (1 try + 1 retry)", got, apiErr.Attempts)
	}
	if hdr.Get(attemptsHeader) != "" {
		t.Errorf("internal %s header leaked to the caller", attemptsHeader)
	}
	if IsNotFound(err) {
		t.Error("IsNotFound must be false for 500/errorCode 1000 (quirk 39)")
	}
}

// A transient 500 followed by a 200 still succeeds, with no error.
func TestRetryExhausted_Transient500ThenSuccess(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	c := newRetryTestClient(t, srv.URL, 2)

	var out struct{ OK bool }
	if _, err := c.DoRequest(context.Background(), "GET", "/x", "application/json;version=1", "application/json", nil, &out); err != nil {
		t.Fatalf("want success after one retry, got %v", err)
	}
	if !out.OK || atomic.LoadInt32(&hits) != 2 {
		t.Errorf("out=%+v hits=%d, want OK after 2 hits", out, hits)
	}
}

// A network error has no response to pass through: it must still fail
// cleanly with a plain error (not an *APIError), after the retries.
func TestRetryExhausted_NetworkErrorStillErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens on this address any more
	c := newRetryTestClient(t, url, 1)

	_, err := c.DoRequest(context.Background(), "GET", "/x", "application/json;version=1", "application/json", nil, nil)
	if err == nil {
		t.Fatal("want an error for an unreachable server")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Fatalf("a network error must not become an *APIError: %v", err)
	}
	if !strings.Contains(err.Error(), "giving up after 2 attempt(s)") {
		t.Errorf("error %q should report the attempt count", err)
	}
}

// A canceled context still ends the request with the context error.
func TestRetryExhausted_CanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := newRetryTestClient(t, srv.URL, 3)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.DoRequest(ctx, "GET", "/x", "application/json;version=1", "application/json", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
