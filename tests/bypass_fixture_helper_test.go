package tests

import (
	"os"
	"regexp"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// newMockServerWithBypassFixtures builds a *mockserver.MockServer for tests
// that must load specific fixtures via a path that bypasses
// MOCK_RESPONSES_DIR (mockserver.LoadResponseFromFile is override-blind --
// see ResolveResponsesDir's doc comment), while still loading every other
// fixture the way mockserver.LoadMockResponses normally would (an
// override-aware recursive directory scan).
//
// This exists for fixtures that are intentionally NOT migrated to the
// vendored-SDK fixture set (e.g. testdata/mock-responses/scripts{,-assignments}/
// synthetic placeholders that have no live-tenant equivalent to capture): a
// test depending on one of those fixtures must not silently find nothing
// when MOCK_RESPONSES_DIR points at a vendored tree that never contains
// them. The bypassPaths are loaded directly via mockserver.LoadResponseFromFile
// (the same mechanism TestGeneratedScriptsV1BulkDeletePartial already uses
// for scripts/bulkdelete-zz-partial.json) and always win: any directory-scan
// fixture sharing the same (HTTP method, endpoint) tuple is dropped in favor
// of the bypass-loaded one, so a test never registers two competing
// responses for the same route.
func newMockServerWithBypassFixtures(t *testing.T, scanDir string, bypassPaths []string) *mockserver.MockServer { //nolint:unparam // scanDir stays a parameter so each call site names the corpus it scans
	t.Helper()

	resolvedScanDir := mockserver.ResolveResponsesDir(scanDir)

	// A scan directory that doesn't exist AT ALL (not just empty of the
	// specific fixtures a caller needs) is not an error here: this helper's
	// whole purpose is to keep working when repo-root testdata/ -- and thus
	// resolvedScanDir -- may be entirely absent (Phase 3's eventual `git rm
	// testdata/`, or this same condition simulated today via `mv testdata
	// ...`). Treat "directory doesn't exist" as zero scanned fixtures and
	// proceed on bypassPaths alone; any OTHER LoadResponsesFromDir error
	// (permissions, malformed JSON, etc.) still fails loudly as before.
	var scanned []*mockserver.MockResponse
	if _, statErr := os.Stat(resolvedScanDir); statErr == nil {
		var err error
		scanned, err = mockserver.LoadResponsesFromDir(resolvedScanDir)
		if err != nil {
			t.Fatalf("failed to load mock responses from %s: %v", resolvedScanDir, err)
		}
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("failed to stat scan dir %s: %v", resolvedScanDir, statErr)
	}

	bypassed := make([]*mockserver.MockResponse, 0, len(bypassPaths))
	exclude := make(map[string]bool, len(bypassPaths))
	for _, p := range bypassPaths {
		resp, err := mockserver.LoadResponseFromFile(p)
		if err != nil {
			t.Fatalf("failed to load bypass fixture %s: %v", p, err)
		}
		bypassed = append(bypassed, resp)
		exclude[resp.Metadata.Method+" "+resp.Metadata.Endpoint] = true
	}

	combined := make([]*mockserver.MockResponse, 0, len(scanned)+len(bypassed))
	for _, r := range scanned {
		if exclude[r.Metadata.Method+" "+r.Metadata.Endpoint] {
			continue
		}
		combined = append(combined, r)
	}
	combined = append(combined, bypassed...)

	if len(combined) == 0 {
		t.Fatalf("no mock responses found (scan dir=%s, bypass=%v)", resolvedScanDir, bypassPaths)
	}

	return mockserver.NewMockServer(combined)
}

// newMockServerFromBypassFixturesOnly builds a *mockserver.MockServer from
// EXACTLY the given bypass paths, with no directory scan at all.
//
// This exists for callers whose fixture set is entirely synthetic-placeholder
// (see internal/mockserver/testdata/synthetic-placeholders/) and whose normal
// scan directory is itself platform-specific and may not exist at all under a
// MOCK_RESPONSES_DIR override (e.g. runInternalAppsV1LifecycleBypass's
// "internal-apps-v1/zip" case: the vendored fixture tree has no zip
// subdirectory to fall back to, since live .zip capture is held pending an
// SFD feature-flag confirmation -- see internal_app_create_zip.json's own
// metadata.description). Unlike newMockServerWithBypassFixtures, this never
// calls mockserver.ResolveResponsesDir or mockserver.LoadResponsesFromDir, so
// it can never fail because a directory doesn't exist under the override.
func newMockServerFromBypassFixturesOnly(t *testing.T, bypassPaths []string) *mockserver.MockServer {
	t.Helper()

	responses := make([]*mockserver.MockResponse, 0, len(bypassPaths))
	for _, p := range bypassPaths {
		resp, err := mockserver.LoadResponseFromFile(p)
		if err != nil {
			t.Fatalf("failed to load bypass fixture %s: %v", p, err)
		}
		responses = append(responses, resp)
	}

	if len(responses) == 0 {
		t.Fatalf("no bypass fixtures given")
	}

	return mockserver.NewMockServer(responses)
}

// uuidShapePattern is a permissive 8-4-4-4-12 hex-with-dashes shape check
// (not RFC 4122 version/variant strict) used by tests that were rewritten to
// assert "a UUID-shaped string is present" rather than a specific captured
// UUID value, per the 2026-08-06 synthetic-placeholder fixture-honesty work
// covering population 2 (see internal/mockserver/testdata/synthetic-placeholders/).
var uuidShapePattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// looksLikeUUID reports whether s has the 8-4-4-4-12 hex-with-dashes shape of
// a UUID. It intentionally does not validate RFC 4122 version/variant bits --
// callers using it are checking "is this shaped like a UUID", not "is this a
// valid version-4 UUID".
func looksLikeUUID(s string) bool {
	return uuidShapePattern.MatchString(s)
}
