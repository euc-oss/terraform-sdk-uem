package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveResponsesDir(t *testing.T) {
	const envKey = "MOCK_RESPONSES_DIR"

	t.Run("returns requested dir when override unset", func(t *testing.T) {
		t.Setenv(envKey, "")
		got := ResolveResponsesDir("../testdata/mock-responses")
		if got != "../testdata/mock-responses" {
			t.Errorf("expected requested dir to be returned, got %q", got)
		}
	})

	t.Run("env override replaces requested dir", func(t *testing.T) {
		t.Setenv(envKey, "/captured/v2509/mock-responses")
		got := ResolveResponsesDir("../testdata/mock-responses")
		if got != "/captured/v2509/mock-responses" {
			t.Errorf("expected env override to win, got %q", got)
		}
	})

	t.Run("whitespace-only override is ignored", func(t *testing.T) {
		t.Setenv(envKey, "   ")
		got := ResolveResponsesDir("../testdata/mock-responses")
		if got != "../testdata/mock-responses" {
			t.Errorf("expected blank override to be ignored, got %q", got)
		}
	})

	t.Run("root-level request resolves to plain override root", func(t *testing.T) {
		t.Setenv(envKey, "/captured/v2509/mock-responses")
		got := ResolveResponsesDir("../testdata/mock-responses")
		want := filepath.Join("/captured/v2509/mock-responses")
		if got != want {
			t.Errorf("expected root-level request to resolve to plain override root, got %q want %q", got, want)
		}
	})

	t.Run("platform-scoped subdirectory request preserves suffix under override", func(t *testing.T) {
		t.Setenv(envKey, "/captured/v2509/mock-responses")
		got := ResolveResponsesDir("../testdata/mock-responses/internal-apps-v1/apk")
		want := filepath.Join("/captured/v2509/mock-responses", "internal-apps-v1", "apk")
		if got != want {
			t.Errorf("expected subdirectory suffix to be preserved under override, got %q want %q", got, want)
		}
	})

	t.Run("sibling platform subdirectories resolve to distinct dirs under the same override", func(t *testing.T) {
		t.Setenv(envKey, "/captured/v2509/mock-responses")
		apk := ResolveResponsesDir("../testdata/mock-responses/internal-apps-v1/apk")
		ipa := ResolveResponsesDir("../testdata/mock-responses/internal-apps-v1/ipa")
		msi := ResolveResponsesDir("../testdata/mock-responses/internal-apps-v1/msi")
		pkg := ResolveResponsesDir("../testdata/mock-responses/internal-apps-v1/pkg")
		seen := map[string]bool{}
		for _, dir := range []string{apk, ipa, msi, pkg} {
			if seen[dir] {
				t.Fatalf("platform-scoped dirs collided on %q — this would make the shared POST /api/mam/apps/internal/application fixture ambiguous across platforms", dir)
			}
			seen[dir] = true
		}
	})

	t.Run("request with no mock-responses segment falls back to verbatim override", func(t *testing.T) {
		t.Setenv(envKey, "/captured/v2509/mock-responses")
		got := ResolveResponsesDir("/some/other/fixtures/dir")
		if got != "/captured/v2509/mock-responses" {
			t.Errorf("expected verbatim fallback for a requested dir with no mock-responses segment, got %q", got)
		}
	})
}

func TestLoadMockResponsesHonorsEnvOverride(t *testing.T) {
	// A fixture dir the requested arg does NOT point at.
	overrideDir := t.TempDir()
	fixture := `{
		"metadata": {"endpoint": "/api/override/probe", "method": "GET", "description": "env override probe"},
		"response": {"status_code": 200, "headers": {}, "body": {"ok": true}}
	}`
	if err := os.WriteFile(filepath.Join(overrideDir, "probe.json"), []byte(fixture), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	t.Setenv(ResponsesDirEnv, overrideDir)

	// Requested dir is deliberately bogus; the override must win and load the probe.
	ms := LoadMockResponses(t, "/does/not/exist/mock-responses")
	t.Cleanup(ms.Close)

	if got := ms.URL(); got == "" {
		t.Fatal("expected a running mock server from the override dir")
	}
}

// TestLoadMockResponsesFromVendoredStagingLikeDir demonstrates, at the
// mechanism level, that a `make mock-populate`-shaped staging directory
// (vendored-mock-responses/internal-apps-v1/<platform>/*.json) is genuinely
// what gets resolved and loaded when MOCK_RESPONSES_DIR points at it — not
// testdata/. The real vendored orphan branch is a separate, currently-held
// deliverable, so this hand-builds a staging dir shaped like its real
// on-disk counterpart (testdata/mock-responses/internal-apps-v1/apk/
// get_internal_app_apk.json), with a distinctive marker value swapped into
// the body so a passing test can only mean the STAGING copy was served.
//
// It also stages a sibling msi/ dir with a conflicting body for a
// same-shaped GET, guarding against the exact platform-collision bug this
// task exists to prevent: if apk and msi ever resolved to the same
// directory, the wrong platform's body could leak through.
func TestLoadMockResponsesFromVendoredStagingLikeDir(t *testing.T) {
	stagingRoot := t.TempDir()

	const marker = "VENDORED-STAGING-MARKER-4f2c9"
	apkDir := filepath.Join(stagingRoot, "internal-apps-v1", "apk")
	if err := os.MkdirAll(apkDir, 0755); err != nil {
		t.Fatalf("failed to create staging apk dir: %v", err)
	}
	apkFixture := `{
		"metadata": {
			"endpoint": "/api/mam/apps/internal/67001",
			"method": "GET",
			"description": "staging-dir stand-in for the real vendored apk fixture"
		},
		"response": {
			"status_code": 200,
			"headers": {"Content-Type": "application/json; charset=utf-8"},
			"body": {"ApplicationName": "` + marker + `", "Platform": "Android", "id": 12347}
		}
	}`
	if err := os.WriteFile(filepath.Join(apkDir, "get_internal_app_apk.json"), []byte(apkFixture), 0644); err != nil {
		t.Fatalf("failed to write staging apk fixture: %v", err)
	}

	msiDir := filepath.Join(stagingRoot, "internal-apps-v1", "msi")
	if err := os.MkdirAll(msiDir, 0755); err != nil {
		t.Fatalf("failed to create staging msi dir: %v", err)
	}
	msiFixture := `{
		"metadata": {
			"endpoint": "/api/mam/apps/internal/67001",
			"method": "GET",
			"description": "staging-dir stand-in msi fixture with a CONFLICTING body for the same path+method"
		},
		"response": {
			"status_code": 200,
			"headers": {},
			"body": {"ApplicationName": "MSI-SHOULD-NOT-LEAK-INTO-APK", "Platform": "WinRT", "id": 99999}
		}
	}`
	if err := os.WriteFile(filepath.Join(msiDir, "get_internal_app_msi.json"), []byte(msiFixture), 0644); err != nil {
		t.Fatalf("failed to write staging msi fixture: %v", err)
	}

	t.Setenv(ResponsesDirEnv, stagingRoot)

	// The exact requested arg the real apk lifecycle call site uses (see
	// tests/generated_internalappsv1_lifecycle_test.go).
	ms := LoadMockResponses(t, "../testdata/mock-responses/internal-apps-v1/apk")
	t.Cleanup(ms.Close)

	resp, err := http.Get(ms.URL() + "/api/mam/apps/internal/67001") //nolint:noctx,gosec // test-only, fixed local mock server URL
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var body struct {
		ApplicationName string `json:"ApplicationName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.ApplicationName != marker {
		t.Fatalf("expected response served from the vendored STAGING dir (ApplicationName=%q), got %q — this fixture either fell back to testdata/ or leaked the sibling msi/ fixture", marker, body.ApplicationName)
	}
}

// TestLoadResponseFromFileRoundTripsLocationHeader is the standing proof for
// doctrine quirk 6 (create-endpoint-returns-201-empty-body-with-Location-header,
// confirmed for MacOsAppsV1Service.CreateMacOSApplication):
// a fixture's response.headers.Location must survive, byte-for-byte, from the
// JSON file on disk through LoadResponseFromFile (the real loader) and out the
// real http.ResponseWriter served by NewMockServer (the real server) — not a
// hand-constructed MockResponse struct.
//
// The JSON body below is byte-identical to the real vendored fixture
// mock-responses/mam-apps/macos_create.json on the
// chore/vendored-swagger-uem-26.2.0.0 orphan branch (confirmed via
// `git show origin/chore/vendored-swagger-uem-26.2.0.0:mock-responses/mam-apps/macos_create.json`
// while writing this test). It is embedded verbatim, following the same
// stand-in pattern as TestLoadMockResponsesFromVendoredStagingLikeDir above,
// so this test does not depend on a live git fetch of the orphan branch.
func TestLoadResponseFromFileRoundTripsLocationHeader(t *testing.T) {
	const wantLocation = "api/mam/apps/internal/424242"

	fixture := `{
		"metadata": {
			"endpoint": "/api/mam/groups/{id}/macos/apps",
			"method": "POST",
			"description": "Captured POST /api/mam/groups/{id}/macos/apps (Pattern B: 201 + Location header pointing at new app's canonical URL; empty body per canonical CreateMacOSApplication — .WithLocationHeader only, no .WithModel).",
			"version": "1"
		},
		"request": {
			"headers": {
				"Accept": "application/json;version=1",
				"Content-Type": "application/json"
			}
		},
		"response": {
			"status_code": 201,
			"headers": {
				"Content-Type": "application/json; charset=utf-8",
				"Location": "api/mam/apps/internal/424242"
			},
			"body": null
		}
	}`

	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "macos_create.json")
	if err := os.WriteFile(fixturePath, []byte(fixture), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Step 1: load through the real loader.
	response, err := LoadResponseFromFile(fixturePath)
	if err != nil {
		t.Fatalf("LoadResponseFromFile failed: %v", err)
	}
	if response.Response.Headers["Location"] != wantLocation {
		t.Fatalf("loaded fixture Location header = %q, want %q", response.Response.Headers["Location"], wantLocation)
	}

	// Step 2: serve through the real mock server.
	ms := NewMockServer([]*MockResponse{response})
	t.Cleanup(ms.Close)

	req, err := http.NewRequest(http.MethodPost, ms.URL()+"/api/mam/groups/10/macos/apps", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Accept", "application/json;version=1")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// Step 3: assert the Location header round-tripped byte-for-byte onto the
	// real http.ResponseWriter.
	if got := resp.Header.Get("Location"); got != wantLocation {
		t.Fatalf("response Location header = %q, want %q (byte-for-byte fixture value)", got, wantLocation)
	}

	// Confirm the empty-body half of the pattern too: JSON `null` in the
	// fixture unmarshals into a truly nil interface{} (confirmed empirically:
	// encoding/json sets an interface{} field to the untyped nil on a `null`
	// token), so server.go's `if response.Response.Body != nil` guard skips
	// encoding entirely and the wire body is genuinely empty — matching
	// quirk 6's "201 + empty body" shape, not a literal "null" payload.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if got := string(body); got != "" {
		t.Fatalf("expected empty body for quirk-6 empty-body create response, got %q", got)
	}
}

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
