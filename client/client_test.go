package client

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewClient tests the creation of a new client with various configurations.
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid OAuth2 config",
			config: &Config{
				InstanceURL:  "https://test.awmdm.com",
				TenantCode:   "test-tenant",
				AuthMethod:   "oauth2",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: false,
		},
		{
			name: "valid Basic Auth config",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "basic",
				Username:    "test-user",
				Password:    "test-pass",
			},
			wantErr: false,
		},
		{
			name: "missing instance URL",
			config: &Config{
				TenantCode:   "test-tenant",
				AuthMethod:   "oauth2",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: true,
		},
		{
			name: "missing tenant code",
			config: &Config{
				InstanceURL:  "https://test.awmdm.com",
				AuthMethod:   "oauth2",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: true,
		},
		{
			name: "missing OAuth2 credentials",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "oauth2",
			},
			wantErr: true,
		},
		{
			name: "missing Basic Auth credentials",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "basic",
			},
			wantErr: true,
		},
		{
			name: "invalid auth method",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

// TestClientDefaults tests that default values are set correctly.
func TestClientDefaults(t *testing.T) {
	config := &Config{
		InstanceURL: "https://test.awmdm.com",
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client.config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries = 3, got %d", client.config.MaxRetries)
	}

	if client.config.RateLimit != 1000 {
		t.Errorf("Expected RateLimit = 1000, got %d", client.config.RateLimit)
	}

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout = 30s, got %v", client.config.Timeout)
	}
}

// TestDoRequest tests the DoRequest method with a mock server.
func TestDoRequest(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("aw-tenant-code") != "test-tenant" {
			t.Errorf("Missing or incorrect aw-tenant-code header")
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("Missing Authorization header")
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
			t.Fatalf("Failed to encode response body: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	config := &Config{
		InstanceURL: server.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Execute request
	var result map[string]string
	_, err = client.DoRequest(context.Background(), "GET", "/test", "application/json;version=1", "application/json", nil, &result)
	if err != nil {
		t.Fatalf("DoRequest() failed: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status = success, got %s", result["status"])
	}
}

// TestDoRequestSetsHeadersFromParameters verifies that DoRequest sets Accept and Content-Type
// headers from the acceptHeader and contentType parameters rather than a global registry lookup.
func TestDoRequestSetsHeadersFromParameters(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json;version=2" {
			t.Errorf("Accept header = %q, want %q", got, "application/json;version=2")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	_, err = c.DoRequest(context.Background(), "GET", "/test",
		"application/json;version=2", "application/json", nil, &result)
	if err != nil {
		t.Fatal(err)
	}
}

// TestDoRequestRawByteBody verifies that DoRequest sends []byte bodies as raw bytes
// without JSON-marshaling, enabling binary blob uploads (application/octet-stream).
func TestDoRequestRawByteBody(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Value": 42}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, _ := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})

	rawBytes := []byte{0x50, 0x4B, 0x03, 0x04} // ZIP magic bytes
	var result map[string]interface{}
	_, err := c.DoRequest(context.Background(), "POST", "/upload",
		"application/json", "application/octet-stream", rawBytes, &result)
	if err != nil {
		t.Fatal(err)
	}

	// Verify raw bytes were sent, not JSON-marshaled
	if string(receivedBody) != string(rawBytes) {
		t.Errorf("body was JSON-marshaled instead of sent raw: got %q, want raw bytes", receivedBody)
	}
	if receivedContentType != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", receivedContentType)
	}
}

// TestDoRequestReturnsResponseHeaders verifies that DoRequest returns response headers.
func TestDoRequestReturnsResponseHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Api-Version", "v2")
		w.Header().Set("Location", "/api/resource/42")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	headers, err := c.DoRequest(context.Background(), "GET", "/test",
		"application/json;version=2", "application/json", nil, &result)
	if err != nil {
		t.Fatal(err)
	}
	if headers == nil {
		t.Fatal("expected non-nil response headers")
	}
	if got := headers.Get("X-Api-Version"); got != "v2" {
		t.Errorf("X-Api-Version header = %q, want %q", got, "v2")
	}
	if got := headers.Get("Location"); got != "/api/resource/42" {
		t.Errorf("Location header = %q, want %q", got, "/api/resource/42")
	}
}

// TestDoRequestXMLBody verifies that DoRequest marshals struct bodies as XML
// when the content type contains "xml".
func TestDoRequestXMLBody(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Value": 1}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, _ := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})

	type TestPayload struct {
		XMLName xml.Name `xml:"Payload"`
		Name    string   `xml:"Name"`
		Value   int      `xml:"Value"`
	}
	payload := TestPayload{Name: "test", Value: 42}

	var result map[string]interface{}
	_, err := c.DoRequest(context.Background(), "POST", "/upload",
		"application/xml", "application/xml", payload, &result)
	if err != nil {
		t.Fatal(err)
	}

	if receivedContentType != "application/xml" {
		t.Errorf("Content-Type = %q, want application/xml", receivedContentType)
	}
	bodyStr := string(receivedBody)
	if !strings.Contains(bodyStr, "<Name>test</Name>") {
		t.Errorf("body should contain XML element <Name>test</Name>, got: %s", bodyStr)
	}
	if strings.Contains(bodyStr, `"Name"`) {
		t.Errorf("body should not contain JSON key \"Name\", got: %s", bodyStr)
	}
}
