package client

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		header   string
		expected bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"Cookie", true},
		{"Set-Cookie", true},
		{"X-Api-Key", true},
		{"Aw-Tenant-Code", true},
		{"Content-Type", false},
		{"Accept", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := isSensitiveHeader(tt.header); got != tt.expected {
				t.Errorf("isSensitiveHeader(%q) = %v, want %v", tt.header, got, tt.expected)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"no query params", "http://example.com/api/profiles", "/api/profiles"},
		{"with query params", "http://example.com/api/profiles?token=secret&page=1", "/api/profiles?<redacted>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			if got := sanitizeURL(req); got != tt.expected {
				t.Errorf("sanitizeURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDebugTransport_RedactsSensitiveHeaders(t *testing.T) {
	// Capture debug output.
	var buf bytes.Buffer
	origWriter := logWriter
	logWriter = &buf
	defer func() { logWriter = origWriter }()

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc123")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &debugTransport{wrapped: http.DefaultTransport}
	req, _ := http.NewRequest("GET", server.URL+"/api/test", nil)
	req.Header.Set("Authorization", "Bearer longtoken12345")
	req.Header.Set("Cookie", "short")
	req.Header.Set("Content-Type", "application/json")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	_ = resp.Body.Close()

	output := buf.String()

	// Authorization should be fully redacted.
	if !strings.Contains(output, "Authorization: ***") {
		t.Errorf("Expected Authorization to be redacted, got:\n%s", output)
	}
	if strings.Contains(output, "Bearer") {
		t.Errorf("Expected no token prefix in output, got:\n%s", output)
	}

	// Cookie should be fully redacted.
	if !strings.Contains(output, "Cookie: ***") {
		t.Errorf("Expected Cookie to be redacted, got:\n%s", output)
	}

	// Set-Cookie in response should be redacted.
	if !strings.Contains(output, "Set-Cookie: ***") {
		t.Errorf("Expected Set-Cookie response header to be redacted, got:\n%s", output)
	}

	// Content-Type should appear in full.
	if !strings.Contains(output, "Content-Type: application/json") {
		t.Errorf("Expected Content-Type to appear unredacted, got:\n%s", output)
	}
}

func TestDebugTransport_DoesNotPanic(t *testing.T) {
	var buf bytes.Buffer
	origWriter := logWriter
	logWriter = &buf
	defer func() { logWriter = origWriter }()

	// Transport that returns an error.
	transport := &debugTransport{
		wrapped: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, http.ErrServerClosed
		}),
	}

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	_, err := transport.RoundTrip(req)
	if err == nil {
		t.Error("Expected error from RoundTrip")
	}

	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("Expected error logged, got:\n%s", buf.String())
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
