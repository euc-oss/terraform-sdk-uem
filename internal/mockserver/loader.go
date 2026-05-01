package mockserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadResponsesFromDir loads all mock response JSON files from a directory recursively.
func LoadResponsesFromDir(dir string) ([]*MockResponse, error) {
	var responses []*MockResponse

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// Load the response file
		response, err := LoadResponseFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		responses = append(responses, response)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	return responses, nil
}

// LoadResponseFromFile loads a single mock response from a JSON file.
func LoadResponseFromFile(path string) (*MockResponse, error) {
	// Clean path to prevent directory traversal (gosec G304)
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var response MockResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if response.Metadata.Endpoint == "" {
		return nil, fmt.Errorf("metadata.endpoint is required")
	}
	if response.Metadata.Method == "" {
		return nil, fmt.Errorf("metadata.method is required")
	}
	if response.Response.StatusCode == 0 {
		return nil, fmt.Errorf("response.status_code is required")
	}

	return &response, nil
}
