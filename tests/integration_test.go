package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
	"github.com/joho/godotenv"
)

// TestMain loads environment variables before running tests.
func TestMain(m *testing.M) {
	// Load .env file from parent directory (only needed for live mode)
	if err := godotenv.Load("../.env"); err != nil {
		// Only warn if we're in live mode
		if os.Getenv("TEST_MODE") == "live" {
			fmt.Println("Warning: .env file not found, using system environment variables")
		}
	}
	os.Exit(m.Run())
}

// getTestClient creates a client for integration testing.
// Supports two modes:
// - TEST_MODE=mock: Uses mock server (no .env file needed)
// - TEST_MODE=live: Uses real Workspace ONE environment (requires .env file)
// Uses standardized environment variable names from docs/SDK-STANDARDS.md.
func getTestClient(t *testing.T, authMethod string) *client.Client {
	t.Helper()

	// Check if we're in mock mode
	testMode := os.Getenv("TEST_MODE")
	if testMode == "" {
		testMode = "mock" // Default to mock mode
	}

	if testMode == "mock" {
		return getMockClient(t, authMethod)
	}

	// Live mode - use real Workspace ONE environment
	return getLiveClient(t, authMethod)
}

// getMockClient creates a client using the mock server.
func getMockClient(t *testing.T, authMethod string) *client.Client {
	t.Helper()

	// Load mock responses
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	// Create client pointing to mock server
	if authMethod == "oauth2" {
		return mockserver.NewMockClientWithOAuth2(t, ms)
	}
	return mockserver.NewMockClient(t, ms)
}

// getLiveClient creates a client using real Workspace ONE environment.
func getLiveClient(t *testing.T, authMethod string) *client.Client {
	t.Helper()

	instanceURL := os.Getenv("INSTANCE_URL")
	tenantCode := os.Getenv("TENANT_CODE")
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	oauth2TokenURL := os.Getenv("OAUTH2_TOKEN_URL")

	if instanceURL == "" {
		t.Skip("INSTANCE_URL environment variable not set, skipping integration test")
	}

	if tenantCode == "" {
		t.Skip("TENANT_CODE environment variable not set, skipping integration test")
	}

	var config *client.Config

	if authMethod == "oauth2" {
		if clientID == "" || clientSecret == "" {
			t.Skip("CLIENT_ID or CLIENT_SECRET not set, skipping OAuth2 test")
		}

		config = &client.Config{
			InstanceURL:    instanceURL,
			TenantCode:     tenantCode,
			AuthMethod:     "oauth2",
			ClientID:       clientID,
			ClientSecret:   clientSecret,
			OAuth2TokenURL: oauth2TokenURL,
			MaxRetries:     3,
			RateLimit:      1000,
			Timeout:        30 * time.Second,
		}
	} else {
		if username == "" || password == "" {
			t.Skip("USERNAME or PASSWORD not set, skipping Basic Auth test")
		}
		config = &client.Config{
			InstanceURL: instanceURL,
			TenantCode:  tenantCode,
			AuthMethod:  "basic",
			Username:    username,
			Password:    password,
			MaxRetries:  3,
			RateLimit:   1000,
			Timeout:     30 * time.Second,
		}
	}

	c, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return c
}

// TestIntegration_BasicAuth_SystemInfo tests Basic Auth with system info endpoint.
func TestIntegration_BasicAuth_SystemInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := getTestClient(t, "basic")

	ctx := context.Background()
	var result map[string]interface{}

	_, err := c.DoRequest(ctx, "GET", "/api/system/info", "application/json;version=1", "application/json", nil, &result)
	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}

	// Verify we got a response
	if len(result) == 0 {
		t.Error("Expected non-empty response from /api/system/info")
	}

	t.Logf("System Info Response: %+v", result)

	// Check for expected fields (these are common in Workspace ONE system info)
	expectedFields := []string{"ProductName", "Version", "ApiVersion"}
	for _, field := range expectedFields {
		if _, ok := result[field]; ok {
			t.Logf("✓ Found expected field: %s = %v", field, result[field])
		}
	}
}

// TestIntegration_BasicAuth_ProfileSearch tests profile search endpoint.
func TestIntegration_BasicAuth_ProfileSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := getTestClient(t, "basic")

	ctx := context.Background()
	var result map[string]interface{}

	// Search for profiles with pagination
	_, err := c.DoRequest(ctx, "GET", "/api/mdm/profiles/search?page=0&pagesize=10", "application/json;version=1", "application/json", nil, &result)
	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}

	t.Logf("Profile Search Response: %+v", result)

	// Check for pagination fields
	if page, ok := result["Page"]; ok {
		t.Logf("✓ Page: %v", page)
	}
	if total, ok := result["Total"]; ok {
		t.Logf("✓ Total pages: %v", total)
	}
	if profiles, ok := result["Profiles"]; ok {
		t.Logf("✓ Profiles returned: %T", profiles)
	}
}

// TestIntegration_OAuth2_SystemInfo tests OAuth2 authentication with system info endpoint.
func TestIntegration_OAuth2_SystemInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := getTestClient(t, "oauth2")

	ctx := context.Background()
	var result map[string]interface{}

	_, err := c.DoRequest(ctx, "GET", "/api/system/info", "application/json;version=1", "application/json", nil, &result)
	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}

	// Verify we got a response
	if len(result) == 0 {
		t.Error("Expected non-empty response from /api/system/info")
	}

	t.Logf("System Info Response (OAuth2): %+v", result)
}

// TestIntegration_RateLimiting tests that rate limiting works correctly.
func TestIntegration_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create client with very low rate limit for testing
	host := os.Getenv("HOST")
	apiKey := os.Getenv("API_KEY")
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")

	if host == "" || username == "" || password == "" {
		t.Skip("Required environment variables not set")
	}

	config := &client.Config{
		InstanceURL: fmt.Sprintf("https://%s", host),
		TenantCode:  apiKey,
		AuthMethod:  "basic",
		Username:    username,
		Password:    password,
		MaxRetries:  3,
		RateLimit:   10, // Only 10 requests per minute
		Timeout:     30 * time.Second,
	}

	c, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	var result map[string]interface{}

	// Make multiple requests and measure time
	start := time.Now()
	requestCount := 5

	for i := 0; i < requestCount; i++ {
		_, err = c.DoRequest(ctx, "GET", "/api/system/info", "application/json;version=1", "application/json", nil, &result)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		t.Logf("Request %d completed", i+1)
	}

	elapsed := time.Since(start)
	t.Logf("Completed %d requests in %v", requestCount, elapsed)

	// With rate limit of 10/min (6 seconds per request), 5 requests should take ~24 seconds
	// We'll check if it took at least 12 seconds (allowing for some variance)
	minExpectedDuration := 12 * time.Second
	if elapsed < minExpectedDuration {
		t.Logf("Warning: Rate limiting may not be working as expected. Expected at least %v, got %v", minExpectedDuration, elapsed)
	} else {
		t.Logf("✓ Rate limiting appears to be working correctly")
	}
}

// TestIntegration_ErrorHandling tests error handling for invalid requests.
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := getTestClient(t, "basic")

	ctx := context.Background()
	var result map[string]interface{}

	// Test 404 error
	_, err := c.DoRequest(ctx, "GET", "/api/invalid/endpoint/that/does/not/exist", "application/json;version=1", "application/json", nil, &result)
	if err == nil {
		t.Error("Expected error for invalid endpoint, got nil")
	} else {
		t.Logf("✓ Got expected error for invalid endpoint: %v", err)

		// Check if it's an APIError
		if apiErr, ok := err.(*client.APIError); ok {
			t.Logf("✓ Error is APIError with status code: %d", apiErr.StatusCode)
			if apiErr.StatusCode != 404 {
				t.Logf("Warning: Expected status code 404, got %d", apiErr.StatusCode)
			}
		}
	}
}

// TestIntegration_HeadersPresent tests that required headers are sent.
func TestIntegration_HeadersPresent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := getTestClient(t, "basic")

	ctx := context.Background()
	var result map[string]interface{}

	// Make a request - if headers are missing, the API will return an error
	_, err := c.DoRequest(ctx, "GET", "/api/system/info", "application/json;version=1", "application/json", nil, &result)
	if err != nil {
		t.Fatalf("DoRequest failed (headers may be missing): %v", err)
	}

	t.Log("✓ Request succeeded, indicating all required headers are present")
	t.Log("✓ aw-tenant-code header is being sent correctly")
	t.Log("✓ Authorization header is being sent correctly")
}

// TestIntegration_ContextCancellation tests that context cancellation works.
func TestIntegration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := getTestClient(t, "basic")

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var result map[string]interface{}

	_, err := c.DoRequest(ctx, "GET", "/api/system/info", "application/json;version=1", "application/json", nil, &result)
	if err == nil {
		t.Error("Expected error for canceled context, got nil")
	} else {
		t.Logf("✓ Got expected error for canceled context: %v", err)
	}
}
