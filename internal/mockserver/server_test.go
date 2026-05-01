package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestMockServer_BasicOperation(t *testing.T) {
	// Create mock responses
	responses := []*MockResponse{
		{
			Metadata: ResponseMetadata{
				Endpoint:    "/api/test",
				Method:      "GET",
				Description: "Test endpoint",
			},
			Response: ResponseSpec{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]string{
					"status": "ok",
				},
			},
		},
	}

	// Create mock server
	server := NewMockServer(responses)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL() + "/api/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	// Verify status code
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify content type
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	// Verify body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status=ok, got %v", result)
	}
}

func TestMockServer_NoMatch(t *testing.T) {
	// Create mock server with no responses
	server := NewMockServer([]*MockResponse{})
	defer server.Close()

	// Make request to non-existent endpoint
	resp, err := http.Get(server.URL() + "/api/nonexistent")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	// Verify 404 status
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// Verify error message
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["error"] == "" {
		t.Errorf("Expected error message, got empty string")
	}
}

func TestMockServer_PathParameters(t *testing.T) {
	// Create mock response with path parameter
	responses := []*MockResponse{
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/profiles/{id}",
				Method:   "GET",
			},
			Response: ResponseSpec{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]interface{}{
					"id":   12345,
					"name": "Test Profile",
				},
			},
		},
	}

	server := NewMockServer(responses)
	defer server.Close()

	// Make request with ID parameter
	resp, err := http.Get(server.URL() + "/api/profiles/12345")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	// Verify status code
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["name"] != "Test Profile" {
		t.Errorf("Expected name=Test Profile, got %v", result["name"])
	}
}
