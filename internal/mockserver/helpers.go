package mockserver

import (
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
)

// NewMockClient creates a client configured to use the mock server
// This is a convenience function for tests.
func NewMockClient(t *testing.T, mockServer *MockServer) *client.Client {
	t.Helper()

	config := &client.Config{
		InstanceURL: mockServer.URL(),
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}

	c, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create mock client: %v", err)
	}

	return c
}

// NewMockClientWithOAuth2 creates a client with OAuth2 authentication for the mock server.
func NewMockClientWithOAuth2(t *testing.T, mockServer *MockServer) *client.Client {
	t.Helper()

	config := &client.Config{
		InstanceURL:    mockServer.URL(),
		TenantCode:     "test-tenant",
		AuthMethod:     "oauth2",
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		OAuth2TokenURL: mockServer.URL() + "/api/v1/oauth/token",
	}

	c, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create mock OAuth2 client: %v", err)
	}

	return c
}

// LoadMockResponses is a convenience function that loads responses from testdata
// and creates a mock server.
func LoadMockResponses(t *testing.T, dir string) *MockServer {
	t.Helper()

	responses, err := LoadResponsesFromDir(dir)
	if err != nil {
		t.Fatalf("Failed to load mock responses from %s: %v", dir, err)
	}

	if len(responses) == 0 {
		t.Fatalf("No mock responses found in %s", dir)
	}

	return NewMockServer(responses)
}
