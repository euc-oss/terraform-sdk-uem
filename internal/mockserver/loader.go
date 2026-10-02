package mockserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResponsesDirEnv is the environment variable that overrides the mock-response
// fixture directory. When set to a non-blank value it takes precedence over the
// directory requested by the caller, so fixtures can live in a version-keyed
// location (e.g. co-located with the vendored swagger) without editing every
// test call site. See ResolveResponsesDir.
const ResponsesDirEnv = "MOCK_RESPONSES_DIR"

// mockResponsesDirName is the directory name every real call site's
// requested path is rooted at (e.g. "../testdata/mock-responses" or
// "../testdata/mock-responses/internal-apps-v1/apk"). ResolveResponsesDir
// uses it to preserve any subdirectory below that root when substituting
// the MOCK_RESPONSES_DIR override.
const mockResponsesDirName = "mock-responses"

// ResolveResponsesDir returns the directory mock responses should be loaded
// from. If the MOCK_RESPONSES_DIR environment variable is set to a non-blank
// value it wins; otherwise the requested directory is used unchanged.
//
// When the override applies, it does NOT simply replace requested wholesale:
// any path segment below a literal "mock-responses" directory in requested
// (e.g. "internal-apps-v1/apk") is preserved and re-joined onto the override
// root. This matters because several call sites request platform/service-
// scoped subdirectories that each define fixtures for the SAME endpoint+
// method with different bodies (e.g. internal-apps-v1/{apk,ipa,msi,pkg} all
// fixture POST /api/mam/apps/internal/application) — collapsing every
// requested dir onto the same override root would make those fixtures
// ambiguous. A root-level request (no segment after "mock-responses")
// naturally resolves to the plain override root, preserving the prior
// behavior for the common case.
func ResolveResponsesDir(requested string) string {
	override := strings.TrimSpace(os.Getenv(ResponsesDirEnv))
	if override == "" {
		return requested
	}

	cleaned := filepath.ToSlash(filepath.Clean(requested))
	segments := strings.Split(cleaned, "/")
	for i, seg := range segments {
		if seg == mockResponsesDirName {
			suffix := filepath.Join(segments[i+1:]...)
			return filepath.Join(override, suffix)
		}
	}

	// requested has no "mock-responses" path segment at all. None of the
	// real call sites hit this today, but rather than silently discarding
	// the override (or the requested path), fall back to the previous
	// verbatim-replace behavior.
	return override
}

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

	response.SourceFile = filepath.Base(cleanPath)
	return &response, nil
}
