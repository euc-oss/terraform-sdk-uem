package mockserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResponseFromFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid response file",
			content: `{
				"metadata": {
					"endpoint": "/api/test",
					"method": "GET",
					"description": "Test endpoint"
				},
				"request": {
					"headers": {"Accept": "application/json"}
				},
				"response": {
					"status_code": 200,
					"headers": {"Content-Type": "application/json"},
					"body": {"status": "ok"}
				}
			}`,
			expectError: false,
		},
		{
			name: "missing endpoint",
			content: `{
				"metadata": {
					"method": "GET"
				},
				"response": {
					"status_code": 200
				}
			}`,
			expectError: true,
			errorMsg:    "metadata.endpoint is required",
		},
		{
			name: "missing method",
			content: `{
				"metadata": {
					"endpoint": "/api/test"
				},
				"response": {
					"status_code": 200
				}
			}`,
			expectError: true,
			errorMsg:    "metadata.method is required",
		},
		{
			name: "missing status code",
			content: `{
				"metadata": {
					"endpoint": "/api/test",
					"method": "GET"
				},
				"response": {
					"headers": {}
				}
			}`,
			expectError: true,
			errorMsg:    "response.status_code is required",
		},
		{
			name:        "invalid JSON",
			content:     `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "mock-response-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() {
				if err := os.Remove(tmpFile.Name()); err != nil {
					t.Fatalf("Failed to remove temp file: %v", err)
				}
			}()

			// Write content
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			if err := tmpFile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			// Load response
			response, err := LoadResponseFromFile(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != "failed to parse JSON: "+tt.errorMsg && err.Error() != tt.errorMsg {
					// Check if error message contains expected text
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if response == nil {
					t.Errorf("Expected response but got nil")
				}
			}
		})
	}
}

func TestLoadResponsesFromDir(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mock-responses-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatalf("Failed to remove temp dir: %v", err)
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create valid response files
	validResponse := `{
		"metadata": {"endpoint": "/api/test", "method": "GET", "description": "Test"},
		"response": {"status_code": 200, "headers": {}, "body": {}}
	}`

	files := []string{
		filepath.Join(tmpDir, "response1.json"),
		filepath.Join(subDir, "response2.json"),
	}

	for _, file := range files {
		if err := os.WriteFile(file, []byte(validResponse), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", file, err)
		}
	}

	// Create non-JSON file (should be skipped)
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write readme: %v", err)
	}

	// Load responses
	responses, err := LoadResponsesFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load responses: %v", err)
	}

	// Verify count
	if len(responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(responses))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
