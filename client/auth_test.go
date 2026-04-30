package client

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestBasicAuth tests the Basic authentication provider.
func TestBasicAuth(t *testing.T) {
	auth := NewBasicAuth("testuser", "testpass")

	token, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() failed: %v", err)
	}

	// Decode and verify token
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("Failed to decode token: %v", err)
	}

	expected := "testuser:testpass"
	if string(decoded) != expected {
		t.Errorf("Expected decoded token = %s, got %s", expected, string(decoded))
	}

	// Verify token is cached
	token2, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() second call failed: %v", err)
	}

	if token != token2 {
		t.Error("Token should be cached and return the same value")
	}
}

// TestBasicAuthRefresh tests that RefreshToken is a no-op for Basic Auth.
func TestBasicAuthRefresh(t *testing.T) {
	auth := NewBasicAuth("testuser", "testpass")

	err := auth.RefreshToken()
	if err != nil {
		t.Errorf("RefreshToken() should not return error for Basic Auth, got: %v", err)
	}
}

// TestOAuth2Auth tests the OAuth2 authentication provider.
func TestOAuth2Auth(t *testing.T) {
	// Create mock OAuth2 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("Expected path /oauth/token, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("Expected grant_type = client_credentials, got %s", r.Form.Get("grant_type"))
		}

		if r.Form.Get("client_id") != "test-client" {
			t.Errorf("Expected client_id = test-client, got %s", r.Form.Get("client_id"))
		}

		if r.Form.Get("client_secret") != "test-secret" {
			t.Errorf("Expected client_secret = test-secret, got %s", r.Form.Get("client_secret"))
		}

		// Return token response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}); err != nil {
			t.Fatalf("Failed to encode response body: %v", err)
		}
	}))
	defer server.Close()

	// Create OAuth2 auth provider
	auth := NewOAuth2Auth(server.URL, "test-client", "test-secret")

	// Get token
	token, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() failed: %v", err)
	}

	if token != "test-access-token" {
		t.Errorf("Expected token = test-access-token, got %s", token)
	}

	// Verify token is cached (should not make another request)
	token2, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() second call failed: %v", err)
	}

	if token != token2 {
		t.Error("Token should be cached and return the same value")
	}
}

// TestOAuth2AuthRefresh tests token refresh functionality.
func TestOAuth2AuthRefresh(t *testing.T) {
	callCount := 0

	// Create mock OAuth2 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}); err != nil {
			t.Fatalf("Failed to encode response body: %v", err)
		}
	}))
	defer server.Close()

	// Create OAuth2 auth provider
	auth := NewOAuth2Auth(server.URL, "test-client", "test-secret")

	// First refresh
	err := auth.RefreshToken()
	if err != nil {
		t.Fatalf("RefreshToken() failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call to token endpoint, got %d", callCount)
	}

	// Second refresh
	err = auth.RefreshToken()
	if err != nil {
		t.Fatalf("RefreshToken() second call failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls to token endpoint, got %d", callCount)
	}
}

// TestOAuth2AuthExpiry tests that expired tokens are refreshed automatically.
func TestOAuth2AuthExpiry(t *testing.T) {
	// Create mock OAuth2 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   1, // Expires in 1 second
		}); err != nil {
			t.Fatalf("Failed to encode response body: %v", err)
		}
	}))
	defer server.Close()

	// Create OAuth2 auth provider
	auth := NewOAuth2Auth(server.URL, "test-client", "test-secret")

	// Get initial token
	_, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() failed: %v", err)
	}

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Get token again - should trigger refresh
	_, err = auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() after expiry failed: %v", err)
	}
}
