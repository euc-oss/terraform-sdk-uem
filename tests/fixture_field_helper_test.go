package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// loadFixtureResponseBodyForTest reads a mock-response fixture's
// response.body at runtime and decodes it into target, instead of a test
// hardcoding a copy of one of its field values as a Go string literal. The
// sync scrubber rewrites UUID-shaped literals in shipped Go source, but the
// vendored fixture corpus is staged separately (`make mock-populate`, a git
// archive pull from the vendored orphan branch) and is never scrubbed --
// so a hardcoded literal copy of a real fixture's UUID silently drifts out
// of sync with the actual (unscrubbed) fixture content in the assembled
// public package. The relPath is relative to the mock-responses root (e.g.
// "sensors/get_sensor.json"), the same shape every LoadResponseFromFile
// caller already uses.
func loadFixtureResponseBodyForTest(t *testing.T, relPath string, target interface{}) {
	t.Helper()
	fullPath := filepath.Join(mockserver.ResolveResponsesDir("../testdata/mock-responses"), relPath)
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fullPath, err)
	}
	var decoded struct {
		Response struct {
			Body json.RawMessage `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("parse fixture %s: %v", fullPath, err)
	}
	if err := json.Unmarshal(decoded.Response.Body, target); err != nil {
		t.Fatalf("parse fixture %s response.body into %T: %v", fullPath, target, err)
	}
}
