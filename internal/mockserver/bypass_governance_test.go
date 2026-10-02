package mockserver_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// This file guards the set of fixtures loaded via a path that bypasses
// MOCK_RESPONSES_DIR / ResolveResponsesDir (see internal/mockserver/helpers.go).
// Three mechanisms bypass the override today:
//
//  1. mockserver.LoadResponseFromFile(literalPath) called directly from a
//     tests/*.go call site, including indirectly through the
//     decodeDeviceProfileFixtureBody(t, literalPath) wrapper in
//     tests/generated_deviceprofilev2_response_fields_test.go (it forwards its
//     argument straight into LoadResponseFromFile).
//  2. chunkFixturePath(name) in internal/mockserver/chunk_handler.go, which
//     builds an absolute path via runtime.Caller(0) relative to this package's
//     source file rather than going through ResolveResponsesDir.
//  3. A raw os.ReadFile(literalPath) in tests/*.go that never goes through
//     LoadResponseFromFile at all (e.g. a fixture-unmarshal regression test).
//
// Fixtures reached only through these paths are invisible to the
// MOCK_RESPONSES_DIR-aware staged/vendored suite: setting that env var can
// never redirect them, so a fixture intentionally never migrated to a vendored
// tree (a synthetic placeholder, e.g.) can still be relied on by a test, and
// conversely a fixture that WAS migrated still won't be picked up here even if
// it should be governed as vendored-verified.
//
// bypass-loaded-fixtures.json (tools/internal/capturetools/testdata/) is the
// committed, human-reviewed manifest of exactly which fixtures that is, one
// relative path per fixture (relative to testdata/mock-responses/), deduped
// by fixture path even though several are reached from more than one call
// site. TestBypassFixtureManifestMatchesSource below re-derives that same set
// directly from source (regex/grep-based, not a hardcoded copy of the
// manifest) and fails loudly if a new bypass call site appears without a
// matching manifest update -- catching silent drift instead of a human having
// to remember to keep the two in sync.
//
// # Governance ("verified must be on vendored") portability note
//
// The actually-vendored fixture tree lives on an orphan branch
// (chore/vendored-swagger-uem-<ver>, see docs/guides/vendored-swagger-sourcing.md)
// that this repo never checks out -- `make mock-populate` reads it via
// `git archive` at generate time. There is no committed, portable artifact in
// this repo that lists "which fixtures are currently on that branch", and any
// local checkout of it lives in a scratch directory outside this repo entirely
// -- not something a fresh checkout or CI runner will have. A hard filesystem
// check against a scratch path would pass on a machine that happens to have
// one staged and silently no-op (or worse, silently skip) everywhere else.
//
// So TestBypassFixtureGovernance does NOT attempt to check vendored-tree
// membership directly. Instead, for a bypass-loaded fixture whose ledger
// entry (tests/verified-endpoints.json) is in verified[], it asserts the
// entry's own record is well-formed and self-consistent (found via an exact
// match on its "fixture" field, with the identity fields that the
// support-matrix generator depends on all populated) -- the closest portable,
// committed proxy available for "this fixture is governed as having graduated
// past synthetic-placeholder status," given no committed vendored-fixture
// manifest exists to check against directly. This is a materially weaker check than "diff the fixture bytes
// against the vendored branch" and is flagged as such in the report; a
// stronger version would need either a committed vendored-fixture manifest or
// a CI step with real git access to the orphan branch.
//
// Roughly half of the 22 bypass-loaded fixtures have no ledger entry citing
// them by exact fixture path at all (e.g. profiles_apple_get_after_update.json
// is a secondary state-transition fixture inside a lifecycle test whose
// (service, method) is already governed via a different primary fixture
// citation). That is not treated as a governance failure -- the ledger tracks
// by (service, method), not 1:1 by fixture file -- but it is logged via
// t.Logf so the gap stays visible.

const (
	testsDirRelToPkg          = "../../tests"
	chunkHandlerRelToPkg      = "chunk_handler.go"
	manifestRelToPkg          = "../../tools/internal/capturetools/testdata/bypass-loaded-fixtures.json"
	verifiedEndpointsRelToPkg = "../../tests/verified-endpoints.json"
)

// loadManifest reads the committed bypass-fixture manifest as a sorted,
// deduped slice of paths relative to testdata/mock-responses/.
func loadManifest(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(manifestRelToPkg)
	if err != nil {
		t.Skipf("manifest %s is unreadable (%v) -- tools/ (including tools/internal/capturetools/testdata/) is never carried into the public package, so this private-capture-tooling governance check is structurally unavailable there", manifestRelToPkg, err)
	}
	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		t.Fatalf("parse manifest %s: %v", manifestRelToPkg, err)
	}
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		if seen[p] {
			t.Fatalf("manifest %s contains duplicate entry %q", manifestRelToPkg, p)
		}
		seen[p] = true
	}
	sort.Strings(paths)
	return paths
}

// normalizeMockResponsesLiteral strips the "../testdata/mock-responses/"
// prefix real tests/*.go call sites use (they're all rooted one directory
// below tests/), returning (relPath, true) on a match. Literals that don't
// carry that prefix aren't mock-response fixture references at all (e.g. an
// unrelated os.ReadFile in the same package) and are ignored by the caller.
func normalizeMockResponsesLiteral(raw string) (string, bool) {
	const prefix = "../testdata/mock-responses/"
	if !strings.HasPrefix(raw, prefix) {
		return "", false
	}
	return strings.TrimPrefix(raw, prefix), true
}

// scanMechanism1 derives the set of fixtures reached via
// mockserver.LoadResponseFromFile(literal) in tests/*.go, including via the
// decodeDeviceProfileFixtureBody(t, literal) wrapper.
//
// A 2026-08-06 Phase-3 pass wrapped several of these literals in
// mockserver.ResolveResponsesDir(literal) so they resolve correctly once
// MOCK_RESPONSES_DIR points at the vendored fixture set (loader.go's own
// override-aware resolution). That does NOT make them stop being
// "bypass-loaded" in the sense this file tracks: LoadResponseFromFile is
// still an exact single-file lookup, structurally distinct from the
// endpoint+method directory-scan/matching mockserver.LoadMockResponses does
// -- same reasoning scanMechanism2's doc comment already applies to
// chunkFixturePath's own MOCK_RESPONSES_DIR-aware fallback. So `direct` below
// matches BOTH the bare-literal form and the ResolveResponsesDir(...)-wrapped
// form; either shape still counts as this mechanism.
func scanMechanism1(t *testing.T) map[string]bool {
	t.Helper()
	// Bare-literal form: LoadResponseFromFile("literal").
	direct := regexp.MustCompile(`\bLoadResponseFromFile\(\s*"([^"]+)"\s*\)`)
	// ResolveResponsesDir(...)-wrapped form: LoadResponseFromFile(mockserver.ResolveResponsesDir("literal")).
	directResolved := regexp.MustCompile(`\bLoadResponseFromFile\(\s*(?:mockserver\.)?ResolveResponsesDir\(\s*"([^"]+)"\s*\)\s*\)`)
	wrapper := regexp.MustCompile(`\bdecodeDeviceProfileFixtureBody\(\s*t\s*,\s*"([^"]+)"\s*\)`)

	found := map[string]bool{}
	entries, err := os.ReadDir(testsDirRelToPkg)
	if err != nil {
		t.Fatalf("read dir %s: %v", testsDirRelToPkg, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(testsDirRelToPkg, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		src := string(data)
		for _, m := range direct.FindAllStringSubmatch(src, -1) {
			if rel, ok := normalizeMockResponsesLiteral(m[1]); ok {
				found[rel] = true
			}
		}
		for _, m := range directResolved.FindAllStringSubmatch(src, -1) {
			if rel, ok := normalizeMockResponsesLiteral(m[1]); ok {
				found[rel] = true
			}
		}
		for _, m := range wrapper.FindAllStringSubmatch(src, -1) {
			if rel, ok := normalizeMockResponsesLiteral(m[1]); ok {
				found[rel] = true
			}
		}
	}
	return found
}

// scanMechanism2 derives the set of fixtures reached via chunkFixturePath(literal)
// in internal/mockserver/chunk_handler.go. It also derives the fixed
// "internal-apps-v1" directory segment chunkFixturePath joins onto from that
// same function's own source, rather than hardcoding it here, so a future
// rename of that directory is caught rather than silently producing a wrong
// manifest comparison.
//
// Note: since the 2026-08-06 chunkFixturePath MOCK_RESPONSES_DIR-aware fix,
// this derived baseDir is chunkFixturePath's FALLBACK resolution (via
// ResolveResponsesDir) for names not present in
// chunkFixtureSyntheticPlaceholderOverrides -- it reflects the path a name
// resolves to when MOCK_RESPONSES_DIR is unset (or, for the apk/msi/ipa/pkg
// names, the testdata/-relative shape that MOCK_RESPONSES_DIR-aware
// resolution is modeled on), same as before this comment. It is NOT a
// MOCK_RESPONSES_DIR bypass for those names anymore -- they now DO respect
// that override at runtime -- but this manifest still tracks them under
// "bypass-loaded" because chunkFixturePath's exact-name lookup never
// participates in the endpoint+method directory-scan/matching the rest of
// the mock-response system uses, regardless of override-awareness. Names
// present in chunkFixtureSyntheticPlaceholderOverrides (currently zip create,
// begininstall-response.json, internal-application-response.json) are a
// genuine, unconditional bypass in both senses -- see scanChunkFixtureOverrideValues
// below, which derives those separately for the synthetic-placeholder
// manifest.
func scanMechanism2(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(chunkHandlerRelToPkg)
	if err != nil {
		t.Fatalf("read %s: %v", chunkHandlerRelToPkg, err)
	}
	src := string(data)

	// func chunkFixturePath(name string) string {
	//     ...
	//     dir := ResolveResponsesDir(filepath.Join(thisDir, "..", "..", "testdata", "mock-responses", "<segment>"))
	//     return filepath.Join(dir, name)
	// }
	segmentPattern := regexp.MustCompile(`"testdata",\s*"mock-responses",\s*"([^"]+)"\s*\)`)
	segMatch := segmentPattern.FindStringSubmatch(src)
	if segMatch == nil {
		t.Fatalf("could not derive chunkFixturePath's fixed directory segment from %s -- function signature/body changed", chunkHandlerRelToPkg)
	}
	baseDir := segMatch[1]

	found := map[string]bool{}

	plain := regexp.MustCompile(`\bchunkFixturePath\(\s*"([^"]+)"\s*\)`)
	for _, m := range plain.FindAllStringSubmatch(src, -1) {
		found[filepath.ToSlash(filepath.Join(baseDir, m[1]))] = true
	}

	joined := regexp.MustCompile(`\bchunkFixturePath\(\s*filepath\.Join\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)\s*\)`)
	for _, m := range joined.FindAllStringSubmatch(src, -1) {
		found[filepath.ToSlash(filepath.Join(baseDir, m[1], m[2]))] = true
	}

	if len(found) == 0 {
		t.Fatalf("scanMechanism2 found zero chunkFixturePath call sites in %s -- regexes are broken or the file was gutted", chunkHandlerRelToPkg)
	}
	return found
}

// scanMechanism3 derives the set of fixtures reached via a raw
// os.ReadFile(literal) in tests/*.go that is NOT routed through
// LoadResponseFromFile. Restricting the literal to ones containing
// "mock-responses" excludes the many other os.ReadFile call sites in tests/
// that read non-fixture testdata or take a variable path.
func scanMechanism3(t *testing.T) map[string]bool {
	t.Helper()
	pattern := regexp.MustCompile(`\bos\.ReadFile\(\s*"([^"]*mock-responses[^"]*)"\s*\)`)
	// ResolveResponsesDir(...)-wrapped form (see scanMechanism1's doc comment
	// for why this still counts as this mechanism, not a MOCK_RESPONSES_DIR
	// bypass exemption).
	patternResolved := regexp.MustCompile(`\bos\.ReadFile\(\s*(?:mockserver\.)?ResolveResponsesDir\(\s*"([^"]*mock-responses[^"]*)"\s*\)\s*\)`)

	found := map[string]bool{}
	entries, err := os.ReadDir(testsDirRelToPkg)
	if err != nil {
		t.Fatalf("read dir %s: %v", testsDirRelToPkg, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(testsDirRelToPkg, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, m := range pattern.FindAllStringSubmatch(string(data), -1) {
			if rel, ok := normalizeMockResponsesLiteral(m[1]); ok {
				found[rel] = true
			}
		}
		for _, m := range patternResolved.FindAllStringSubmatch(string(data), -1) {
			if rel, ok := normalizeMockResponsesLiteral(m[1]); ok {
				found[rel] = true
			}
		}
	}
	return found
}

// deriveBypassFixtureSet is the real, from-source derivation the manifest is
// checked against -- never a copy of the manifest itself.
func deriveBypassFixtureSet(t *testing.T) []string {
	t.Helper()
	union := map[string]bool{}
	for rel := range scanMechanism1(t) {
		union[rel] = true
	}
	for rel := range scanMechanism2(t) {
		union[rel] = true
	}
	for rel := range scanMechanism3(t) {
		union[rel] = true
	}
	out := make([]string, 0, len(union))
	for rel := range union {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

// TestBypassFixtureManifestMatchesSource fails loudly (naming the exact
// drift) if the statically-derived bypass-loaded fixture set and the
// committed manifest disagree in either direction: a fixture referenced by a
// new/changed bypass call site that isn't in the manifest, or a manifest
// entry no call site references anymore.
func TestBypassFixtureManifestMatchesSource(t *testing.T) {
	derived := deriveBypassFixtureSet(t)
	manifest := loadManifest(t)

	derivedSet := make(map[string]bool, len(derived))
	for _, p := range derived {
		derivedSet[p] = true
	}
	manifestSet := make(map[string]bool, len(manifest))
	for _, p := range manifest {
		manifestSet[p] = true
	}

	var missingFromManifest, staleInManifest []string
	for p := range derivedSet {
		if !manifestSet[p] {
			missingFromManifest = append(missingFromManifest, p)
		}
	}
	for p := range manifestSet {
		if !derivedSet[p] {
			staleInManifest = append(staleInManifest, p)
		}
	}
	sort.Strings(missingFromManifest)
	sort.Strings(staleInManifest)

	if len(missingFromManifest) > 0 || len(staleInManifest) > 0 {
		t.Errorf("bypass-loaded fixture set drifted from %s:\n  found in source but missing from manifest: %v\n  in manifest but no longer reachable from source: %v",
			manifestRelToPkg, missingFromManifest, staleInManifest)
	}
}

// --- tests/verified-endpoints.json ledger helpers ---

type ledgerEntry struct {
	Service      string `json:"service"`
	Method       string `json:"method"`
	HTTPMethod   string `json:"httpMethod"`
	Path         string `json:"path"`
	Fixture      string `json:"fixture,omitempty"`
	TestFunction string `json:"testFunction,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type ledger struct {
	Verified   []ledgerEntry `json:"verified"`
	Unverified []ledgerEntry `json:"unverified"`
}

func loadLedger(t *testing.T) ledger {
	t.Helper()
	data, err := os.ReadFile(verifiedEndpointsRelToPkg)
	if err != nil {
		t.Fatalf("read %s: %v", verifiedEndpointsRelToPkg, err)
	}
	var l ledger
	if err := json.Unmarshal(data, &l); err != nil {
		t.Fatalf("parse %s: %v", verifiedEndpointsRelToPkg, err)
	}
	return l
}

func findByFixture(entries []ledgerEntry, fixture string) *ledgerEntry {
	for i := range entries {
		if entries[i].Fixture == fixture {
			return &entries[i]
		}
	}
	return nil
}

// TestBypassFixtureGovernance walks the committed manifest (equal to the
// derived set per TestBypassFixtureManifestMatchesSource -- using the
// manifest here keeps this test's failures readable even if that other test
// is also failing) and checks each fixture's ledger governance state. See
// the package-level doc comment above for the "verified implies vendored"
// portability reasoning.
func TestBypassFixtureGovernance(t *testing.T) {
	manifest := loadManifest(t)
	l := loadLedger(t)

	for _, rel := range manifest {
		t.Run(rel, func(t *testing.T) {
			if v := findByFixture(l.Verified, rel); v != nil {
				if v.Service == "" || v.Method == "" || v.HTTPMethod == "" || v.Path == "" {
					t.Errorf("verified[] entry for fixture %q is missing identity fields: %+v", rel, v)
				}
				return
			}
			if u := findByFixture(l.Unverified, rel); u != nil {
				if strings.TrimSpace(u.Reason) == "" {
					t.Errorf("unverified[] entry for fixture %q has an empty reason", rel)
				}
				return
			}
			t.Logf("fixture %q has no ledger entry citing it by exact fixture path (expected for secondary/supplementary fixtures governed under a different fixture citation for the same service+method)", rel)
		})
	}
}

// --- Population 2/3 synthetic-placeholder fixture governance (2026-08-06) ---
//
// internal/mockserver/testdata/synthetic-placeholders/ (see that directory's
// own README.md) is a SEPARATE bypass location from testdata/mock-responses/,
// used by fixtures that must survive a future repo-root `git rm testdata/`
// (Phase 3 of the deterministic-capture migration). Fixtures there are
// always loaded via a literal path handed straight to
// mockserver.LoadResponseFromFile (directly, or through
// newMockServerWithBypassFixtures / newMockServerFromBypassFixturesOnly in
// tests/bypass_fixture_helper_test.go) -- never through a directory scan or
// ResolveResponsesDir, so MOCK_RESPONSES_DIR can never redirect or hide them.
//
// The two tests below are this location's counterpart to
// TestBypassFixtureManifestMatchesSource / TestBypassFixtureGovernance
// above: they guard that (a) every literal reference to this location in
// tests/*.go is captured by a committed manifest (drift in either direction
// fails loudly), and (b) every manifest entry exists on disk, is honestly
// labeled, and has SOME governance trail in tests/verified-endpoints.json.

const syntheticPlaceholderManifestRelToPkg = "../../tools/internal/capturetools/testdata/synthetic-placeholder-fixtures.json"
const syntheticPlaceholderDirRelToPkg = "testdata/synthetic-placeholders"

// loadSyntheticPlaceholderManifest reads the committed manifest as a sorted,
// deduped slice of paths relative to
// internal/mockserver/testdata/synthetic-placeholders/.
func loadSyntheticPlaceholderManifest(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(syntheticPlaceholderManifestRelToPkg)
	if err != nil {
		t.Skipf("manifest %s is unreadable (%v) -- tools/ (including tools/internal/capturetools/testdata/) is never carried into the public package, so this private-capture-tooling governance check is structurally unavailable there", syntheticPlaceholderManifestRelToPkg, err)
	}
	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		t.Fatalf("parse manifest %s: %v", syntheticPlaceholderManifestRelToPkg, err)
	}
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		if seen[p] {
			t.Fatalf("manifest %s contains duplicate entry %q", syntheticPlaceholderManifestRelToPkg, p)
		}
		seen[p] = true
	}
	sort.Strings(paths)
	return paths
}

// scanSyntheticPlaceholderLiterals derives the set of fixtures reached via a
// literal "../internal/mockserver/testdata/synthetic-placeholders/<path>"
// string anywhere in tests/*.go, regardless of which function the literal is
// passed to (newMockServerWithBypassFixtures, newMockServerFromBypassFixturesOnly,
// or any future caller) -- deliberately broader than scanMechanism1 above,
// which only matches specific call-site shapes, because these fixtures are
// typically passed inside a []string{...} slice literal rather than directly
// into LoadResponseFromFile(...).
func scanSyntheticPlaceholderLiterals(t *testing.T) map[string]bool {
	t.Helper()
	pattern := regexp.MustCompile(`"\.\./internal/mockserver/testdata/synthetic-placeholders/([^"]+)"`)

	found := map[string]bool{}
	entries, err := os.ReadDir(testsDirRelToPkg)
	if err != nil {
		t.Fatalf("read dir %s: %v", testsDirRelToPkg, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(testsDirRelToPkg, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, m := range pattern.FindAllStringSubmatch(string(data), -1) {
			found[m[1]] = true
		}
	}
	if len(found) == 0 {
		t.Fatalf("scanSyntheticPlaceholderLiterals found zero synthetic-placeholders literal references in %s -- regex is broken or every call site was removed", testsDirRelToPkg)
	}
	return found
}

// scanChunkFixtureOverrideValues derives the set of fixtures reached via
// internal/mockserver/chunk_handler.go's chunkFixtureSyntheticPlaceholderOverrides
// map -- a FOURTH mechanism (alongside scanSyntheticPlaceholderLiterals'
// tests/*.go literal scan) that reaches
// internal/mockserver/testdata/synthetic-placeholders/, added by the
// 2026-08-06 chunkFixturePath fix when begininstall-response.json and
// internal-application-response.json joined the pre-existing zip create
// entry in that table. The file chunk_handler.go lives in internal/mockserver/
// itself, not tests/, so scanSyntheticPlaceholderLiterals' scan of tests/*.go
// alone would never see these -- this function reads the map's VALUE side
// (the path relative to synthetic-placeholders/) directly out of
// chunk_handler.go's own source, the same "derive from source, never
// hardcode a copy" discipline scanMechanism2 above uses for its baseDir.
func scanChunkFixtureOverrideValues(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(chunkHandlerRelToPkg)
	if err != nil {
		t.Fatalf("read %s: %v", chunkHandlerRelToPkg, err)
	}
	src := string(data)

	const startMarker = "var chunkFixtureSyntheticPlaceholderOverrides = map[string]string{"
	start := strings.Index(src, startMarker)
	if start == -1 {
		t.Fatalf("could not find chunkFixtureSyntheticPlaceholderOverrides map literal in %s -- variable renamed or removed", chunkHandlerRelToPkg)
	}
	end := strings.Index(src[start:], "\n}")
	if end == -1 {
		t.Fatalf("could not find closing brace of chunkFixtureSyntheticPlaceholderOverrides map literal in %s", chunkHandlerRelToPkg)
	}
	block := src[start : start+end]

	// Each map entry is `<key>: filepath.Join(<quoted segments>),` -- match
	// the VALUE side only (the colon anchors past the key, whether the key
	// itself is a bare string literal or its own filepath.Join call).
	valuePattern := regexp.MustCompile(`:\s*filepath\.Join\(([^)]*)\)`)
	segmentPattern := regexp.MustCompile(`"([^"]+)"`)

	found := map[string]bool{}
	for _, m := range valuePattern.FindAllStringSubmatch(block, -1) {
		segs := segmentPattern.FindAllStringSubmatch(m[1], -1)
		if len(segs) == 0 {
			continue
		}
		parts := make([]string, 0, len(segs))
		for _, s := range segs {
			parts = append(parts, s[1])
		}
		found[strings.Join(parts, "/")] = true
	}
	if len(found) == 0 {
		t.Fatalf("scanChunkFixtureOverrideValues found zero map entries in %s -- regex is broken or the table was gutted", chunkHandlerRelToPkg)
	}
	return found
}

// TestSyntheticPlaceholderManifestMatchesSource fails loudly (naming the
// exact drift) if the statically-derived synthetic-placeholder fixture set
// and the committed manifest disagree in either direction. The derived set
// is the union of scanSyntheticPlaceholderLiterals (tests/*.go literals) and
// scanChunkFixtureOverrideValues (chunk_handler.go's override table) -- two
// independent mechanisms that both reach
// internal/mockserver/testdata/synthetic-placeholders/.
func TestSyntheticPlaceholderManifestMatchesSource(t *testing.T) {
	derivedSet := scanSyntheticPlaceholderLiterals(t)
	for rel := range scanChunkFixtureOverrideValues(t) {
		derivedSet[rel] = true
	}
	manifest := loadSyntheticPlaceholderManifest(t)

	manifestSet := make(map[string]bool, len(manifest))
	for _, p := range manifest {
		manifestSet[p] = true
	}

	var missingFromManifest, staleInManifest []string
	for p := range derivedSet {
		if !manifestSet[p] {
			missingFromManifest = append(missingFromManifest, p)
		}
	}
	for p := range manifestSet {
		if !derivedSet[p] {
			staleInManifest = append(staleInManifest, p)
		}
	}
	sort.Strings(missingFromManifest)
	sort.Strings(staleInManifest)

	if len(missingFromManifest) > 0 || len(staleInManifest) > 0 {
		t.Errorf("synthetic-placeholder fixture set drifted from %s:\n  found in source but missing from manifest: %v\n  in manifest but no longer reachable from source: %v",
			syntheticPlaceholderManifestRelToPkg, missingFromManifest, staleInManifest)
	}
}

// syntheticPlaceholderHonestyMarkers are the disclosure substrings every
// fixture under internal/mockserver/testdata/synthetic-placeholders/ must
// carry in its own metadata.description -- the same self-disclosure
// convention used throughout this directory (see its README.md and each
// fixture's own description).
var syntheticPlaceholderHonestyMarkers = []string{
	"SYNTHETIC",
	"REPRESENTATIVE PICK",
}

// syntheticPlaceholderHonestyMarkerExceptions carves out manifest entries
// whose metadata.description is honest by CONTEXT rather than by carrying a
// literal syntheticPlaceholderHonestyMarkers substring, so
// TestSyntheticPlaceholderFixtureGovernance's honesty-marker check doesn't
// flag them as false negatives. The value is the justification, surfaced via
// t.Logf so the carve-out stays visible rather than silently swallowed.
//
// The scripts/bulkdelete-zz-partial.json is NOT listed below (as of the fix
// disclosing it as SYNTHETIC -- see tests/verified-endpoints.json and this
// file's own metadata.description): its description now genuinely opens
// with "SYNTHETIC:", so it needs no exception here at all -- it passes the
// normal syntheticPlaceholderHonestyMarkers check directly. It used to be
// listed here (byte-for-byte preserved from its original, pre-disclosure
// wording, honest only by context rather than by a literal marker), but
// that justification is now stale and would be false if left in place.
//
// The 9 entries below (added 2026-09-22) are the
// fixtures whose ledger row was recaptured for real and moved to verified[],
// replacing this file's former fabricated/placeholder body with the genuine
// captured bytes (same relative path under
// internal/mockserver/testdata/synthetic-placeholders/, since that path is
// still the one the bypass mechanism in tests/*.go and
// chunk_handler.go's chunkFixtureSyntheticPlaceholderOverrides table
// hardcodes -- these fixtures are not migrated off that mechanism, just off
// FABRICATED content on it). A genuinely captured description legitimately
// opens with "Captured GET"/"Captured POST"/etc, which is exactly the phrase
// syntheticPlaceholderHonestyMarkers requires an override for (that phrase
// alone carries no SYNTHETIC/REPRESENTATIVE PICK marker, nor should it --
// claiming "SYNTHETIC" of genuinely captured content would itself be
// dishonest). See TestSyntheticPlaceholderFixtureGovernance's ledger check,
// which now accepts a verified[] citation for exactly this reason.
var syntheticPlaceholderHonestyMarkerExceptions = map[string]string{
	"scripts/samples-response.json":                "recaptured for real (vendored 4f0dd5fe5, 2026-09-24) and moved to verified[]; its description now genuinely opens with \"GENUINE: live-captured\", which is correct, not a missing marker.",
	"internal-apps-v1/begininstall-response.json":  "recaptured for real (26ece88ca, vendored branch) and moved to verified[] in tests/verified-endpoints.json; its own description already discloses genuine live capture + auto-anonymization, not a fabricated placeholder -- no SYNTHETIC marker needed or honest to add.",
	"scripts-assignments/bulkupdate-response.json": "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured POST\", which is accurate, not a stale false claim.",
	"scripts-assignments/create-response.json":     "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured POST\", which is accurate, not a stale false claim.",
	"scripts-assignments/get.json":                 "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured GET\", which is accurate, not a stale false claim.",
	"scripts-assignments/list.json":                "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured GET\", which is accurate, not a stale false claim.",
	"scripts/bulkdelete-success.json":              "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured POST\", which is accurate, not a stale false claim.",
	"scripts/create-response.json":                 "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured POST\", which is accurate, not a stale false claim.",
	"scripts/update-response.json":                 "recaptured for real (Phase 3, d14f6ad9b, vendored branch) and moved to verified[]; description now genuinely opens with \"Captured PUT\", which is accurate, not a stale false claim.",
	"system/info_success.json":                     "recaptured for real (Phase 1/3, b9c0556a3/d14f6ad9b, vendored branch) and moved to verified[]; body now carries the genuine seed-12345 gofakeit fingerprint (\"Test Rabbitthrow\"), not the hand-authored fabricated placeholder this file used to ship.",
	"auth/oauth2_token_success.json":               "genuine live capture (real OAuth2 client_credentials token response, shape/headers/status all real) moved to verified[]; access_token is deliberately replaced with an obvious literal placeholder, disclosed as such in the description -- claiming SYNTHETIC of a genuinely captured, only-narrowly-redacted response would itself be dishonest, the same reasoning already applied to the 9 fixtures above.",
}

// allDigitUUIDShape matches any all-digit, UUID-shaped (8-4-4-4-12) substring
// -- deliberately loose (RE2 has no backreferences, so the "each group is a
// single repeated digit" narrowing happens in isSequentialDigitPlaceholder
// below, not in this pattern).
var allDigitUUIDShape = regexp.MustCompile(`\b\d{8}-\d{4}-\d{4}-\d{4}-\d{12}\b`)

// buildUUIDForGovernanceTest assembles a UUID string from its 5
// hyphen-separated groups at RUNTIME, rather than as a single string
// literal. The public package's sync scrubber rewrites any whole-UUID-shaped
// literal it finds anywhere in shipped content -- Go source AND fixture
// JSON alike -- regardless of context, including planted test-control UUIDs
// that must stay byte-exact for this file's own tests to mean anything.
// None of the individual group arguments is itself UUID-shaped, so the
// scrubber has nothing to match; strings.Join re-assembles the same value
// this function's callers need at test-run time.
func buildUUIDForGovernanceTest(groups ...string) string {
	return strings.Join(groups, "-")
}

// guidEmpty is .NET's Guid.Empty ("00000000_0000_0000_0000_000000000000") --
// a common, entirely legitimate sentinel for an unset/default nullable GUID
// field on the wire (this SDK's canonical backend is C#). Every hyphen-group
// of Guid.Empty is trivially "one digit (0) repeated," so it structurally
// matches isSequentialDigitPlaceholder's placeholder definition even though
// it is a real value a live API can genuinely return -- it must never be
// flagged as a fabricated placeholder.
var guidEmpty = buildUUIDForGovernanceTest("00000000", "0000", "0000", "0000", "000000000000")

// isSequentialDigitPlaceholder reports whether every hyphen-delimited group
// of an all-digit-UUID-shaped match consists of one digit repeated for the
// group's full length, the hand-typed placeholder style aaf8134e9's overwrite
// retired ("11111111_2222_3333_4444_555555555555",
// "33333333_4444_5555_6666_777777777777",
// "55555555_6666_7777_8888_999999999999", ...). The gofakeit.UUID() (the genuine
// anonymizer's RNG) cannot produce this shape, so a match against the
// CURRENT on-disk body is proof of a reintroduced fabricated placeholder,
// independent of whatever the description field claims. The guidEmpty is
// excluded: it has the identical structural shape but is a legitimate .NET
// sentinel, not a fabricated placeholder.
func isSequentialDigitPlaceholder(uuid string) bool {
	if uuid == guidEmpty {
		return false
	}
	for _, group := range strings.Split(uuid, "-") {
		for i := 1; i < len(group); i++ {
			if group[i] != group[0] {
				return false
			}
		}
	}
	return true
}

// containsSequentialDigitPlaceholderUUID reports whether body contains any
// substring matching isSequentialDigitPlaceholder.
func containsSequentialDigitPlaceholderUUID(body string) bool {
	for _, m := range allDigitUUIDShape.FindAllString(body, -1) {
		if isSequentialDigitPlaceholder(m) {
			return true
		}
	}
	return false
}

// recapturedFixtureFabricationMarkers is TestSyntheticPlaceholderFixtureGovernance's
// content-level counterpart to the retired relocatedFalseProvenanceFixtures /
// staleFalseClaimSubstrings pair (see git show aaf8134e9 for the deleted
// code): those two guarded that the 9 fixtures' fabricated placeholder
// content, once overwritten with genuine recaptured bytes, never crept back
// in. Commit aaf8134e9 deleted both mechanisms as dead code but replaced them only
// with a ledger-identity check (verified[] row has non-empty
// service/method/fixture/testFunction) -- which says nothing about the
// FIXTURE FILE'S OWN CONTENT, so a plain revert of one of these 9 files to
// its pre-aaf8134e9 fabricated body (leaving the ledger row untouched) passes
// silently. This map restores that missing content check: distinctive
// substrings pulled verbatim from each fixture's OLD (pre-aaf8134e9,
// fabricated) body/description -- via `git show aaf8134e9^:internal/mockserver/
// testdata/synthetic-placeholders/<path>` -- that must never reappear in the
// CURRENT on-disk body. Every entry is checked in addition to (not instead
// of) containsSequentialDigitPlaceholderUUID above, which is a generic structural
// fingerprint rather than a per-fixture literal.
var recapturedFixtureFabricationMarkers = map[string][]string{
	"internal-apps-v1/begininstall-response.json": {
		"SYNTHETIC", "com.example.forklift", "Forklift Test App",
	},
	"scripts-assignments/bulkupdate-response.json": {"SYNTHETIC PLACEHOLDER"},
	"scripts-assignments/create-response.json":     {"SYNTHETIC PLACEHOLDER"},
	"scripts-assignments/get.json":                 {"SYNTHETIC PLACEHOLDER", "Sample assignment 1"},
	"scripts-assignments/list.json":                {"SYNTHETIC PLACEHOLDER", "Sample assignment 1"},
	"scripts/bulkdelete-success.json":              {"SYNTHETIC PLACEHOLDER"},
	"scripts/create-response.json":                 {"SYNTHETIC PLACEHOLDER"},
	"scripts/update-response.json":                 {"SYNTHETIC PLACEHOLDER"},
	"system/info_success.json": {
		"SYNTHETIC PLACEHOLDER", "SDK Synthetic Test Product", "0.0.0-synthetic",
	},
}

// recapturedFixtureFabricationProblems is checkNoRecapturedFixtureFabricationMarkers's
// t-free core: it returns one problem string per fabrication marker or
// sequential-digit-placeholder UUID found in raw, or nil if rel isn't one of
// the 9 recaptured fixtures or raw is clean. Splitting this out from the
// *testing.T-calling wrapper below lets TestSyntheticPlaceholderFixtureGovernance_FailsOnRevertedFixture
// assert the guard's failure path directly (a t.Errorf call can't be
// observed from outside the *testing.T it was invoked on), instead of only
// ever being exercised against fixtures that are already clean.
func recapturedFixtureFabricationProblems(rel string, raw []byte) []string {
	markers, ok := recapturedFixtureFabricationMarkers[rel]
	if !ok {
		return nil
	}
	var problems []string
	body := string(raw)
	for _, marker := range markers {
		if strings.Contains(body, marker) {
			problems = append(problems, fmt.Sprintf("fixture %q content contains fabrication marker %q -- this fixture was recaptured for real (aaf8134e9) and must never regress to its old fabricated placeholder body", rel, marker))
		}
	}
	if containsSequentialDigitPlaceholderUUID(body) {
		problems = append(problems, fmt.Sprintf("fixture %q content contains a sequential-digit placeholder UUID (e.g. \"11111111_2222_3333_4444_555555555555\"-style) -- this fixture was recaptured for real (aaf8134e9) and must never regress to its old fabricated placeholder body", rel))
	}
	return problems
}

// checkNoRecapturedFixtureFabricationMarkers asserts raw (the fixture's full
// on-disk file content, not just the parsed description) carries none of
// recapturedFixtureFabricationMarkers[rel] and no containsSequentialDigitPlaceholderUUID
// match -- a no-op for any rel not in the map (fixtures outside the 9
// recaptured-for-real set, which never made this specific fabricated-content
// claim in the first place).
func checkNoRecapturedFixtureFabricationMarkers(t *testing.T, rel string, raw []byte) {
	t.Helper()
	for _, problem := range recapturedFixtureFabricationProblems(rel, raw) {
		t.Errorf("%s", problem)
	}
}

// TestSyntheticPlaceholderFixtureGovernance walks the committed manifest and
// checks each fixture's on-disk honesty label and ledger governance state.
//
// Ledger matching prefers an exact tests/verified-endpoints.json verified[]
// or unverified[] entry citing this exact relative path via its "fixture"
// field (same findByFixture helper TestBypassFixtureGovernance uses above)
// -- true for 23 of the 24 manifest entries, since their relative sub-path
// was deliberately kept identical to the original testdata/mock-responses/…
// fixture name their ledger entry already cited. The one exception is
// profiles/search_v1_representative.json (the population-3 v1-search
// collision carve-out): its governing ledger entry legitimately cites the
// two vendored v1 search fixtures (search_android_success.json,
// search_empty.json) by design, not this representative-pick file, so an
// exact "fixture" field match is not expected for it. For any entry without
// an exact match, this test falls back to confirming SOME unverified[]
// reason mentions the fixture's relative path by name (a substring check),
// which is true for search_v1_representative.json (its governing entry's
// reason explicitly names it) -- so this fallback still catches a fixture
// that drops off the ledger's radar entirely, without requiring the
// population-3 carve-out to force an exact-fixture citation it deliberately
// doesn't have.
//
// 9 of the 24 manifest entries (2026-09-22) have since been
// recaptured for real and moved to verified[] -- their on-disk body/
// description at this manifest's path was overwritten with the genuine
// recaptured bytes at the same time, so this test's ledger check accepts a
// verified[] citation exactly like TestBypassFixtureGovernance already does,
// rather than requiring these 9 to keep a since-obsolete unverified[] row
// solely to satisfy this test.
//
// The honesty-marker check itself has carve-outs in
// syntheticPlaceholderHonestyMarkerExceptions (fixtures honest by context
// rather than by literal marker, including the 9 fixtures recaptured for
// real and moved to verified[]) -- see that var's doc comment for why.
//
// The ledger check accepts EITHER a verified[] or an unverified[] citation
// (mirroring TestBypassFixtureGovernance's own acceptance pattern above): a
// fixture reached via this bypass mechanism does not stop being tracked here
// once it graduates from fabricated to genuinely recaptured -- it just no
// longer needs an unverified[] reason, since verified[] is itself the
// governance record for real, ledger-verified content.
func TestSyntheticPlaceholderFixtureGovernance(t *testing.T) {
	manifest := loadSyntheticPlaceholderManifest(t)
	l := loadLedger(t)

	for _, rel := range manifest {
		t.Run(rel, func(t *testing.T) {
			fixturePath := filepath.Join(syntheticPlaceholderDirRelToPkg, filepath.FromSlash(rel))
			data, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("read fixture %s: %v", fixturePath, err)
			}
			var parsed struct {
				Metadata struct {
					Description string `json:"description"`
				} `json:"metadata"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("parse fixture %s: %v", fixturePath, err)
			}

			descUpper := strings.ToUpper(parsed.Metadata.Description)
			hasMarker := false
			for _, marker := range syntheticPlaceholderHonestyMarkers {
				if strings.Contains(descUpper, marker) {
					hasMarker = true
					break
				}
			}
			if !hasMarker {
				if justification, ok := syntheticPlaceholderHonestyMarkerExceptions[rel]; ok {
					t.Logf("fixture %s carries no literal honesty marker but is exempted: %s", rel, justification)
				} else {
					t.Errorf("fixture %s metadata.description carries none of the required honesty markers %v: %s", rel, syntheticPlaceholderHonestyMarkers, parsed.Metadata.Description)
				}
			}

			checkNoRecapturedFixtureFabricationMarkers(t, rel, data)

			if v := findByFixture(l.Verified, rel); v != nil {
				if v.Service == "" || v.Method == "" || v.HTTPMethod == "" || v.Path == "" {
					t.Errorf("verified[] entry for fixture %q is missing identity fields: %+v", rel, v)
				}
				return
			}

			if u := findByFixture(l.Unverified, rel); u != nil {
				if strings.TrimSpace(u.Reason) == "" {
					t.Errorf("unverified[] entry for fixture %q has an empty reason", rel)
				}
				return
			}

			for _, u := range l.Unverified {
				if strings.Contains(u.Reason, rel) {
					return
				}
			}
			t.Errorf("fixture %q has no verified[]/unverified[] entry citing it by exact fixture path, and no unverified[] reason mentions it by name -- ungoverned", rel)
		})
	}
}

// TestIsSequentialDigitPlaceholder_GuidEmptyExcluded is the positive/negative
// pair for the guidEmpty carve-out above: Guid.Empty must NOT be flagged
// (issue found in code-quality review round 2, item 3 -- every hyphen-group
// of "00000000_0000_0000_0000_000000000000" is structurally "one digit
// repeated," identical in shape to a genuine fabricated placeholder), while a
// real fabricated placeholder with a non-zero repeating digit must STILL be
// flagged -- proving the guidEmpty carve-out didn't quietly disable the
// detector altogether.
func TestIsSequentialDigitPlaceholder_GuidEmptyExcluded(t *testing.T) {
	if isSequentialDigitPlaceholder(guidEmpty) {
		t.Errorf("isSequentialDigitPlaceholder(%q) = true, want false -- Guid.Empty is a legitimate .NET sentinel, not a fabricated placeholder", guidEmpty)
	}
	if containsSequentialDigitPlaceholderUUID(fmt.Sprintf("prefix %s suffix", guidEmpty)) {
		t.Errorf("containsSequentialDigitPlaceholderUUID flagged a body containing only Guid.Empty, want no flag")
	}

	fabricated := buildUUIDForGovernanceTest("11111111", "2222", "3333", "4444", "555555555555")
	if !isSequentialDigitPlaceholder(fabricated) {
		t.Errorf("isSequentialDigitPlaceholder(%q) = false, want true -- regression control: the guidEmpty carve-out must not disable detection of genuine fabricated placeholders", fabricated)
	}
	if !containsSequentialDigitPlaceholderUUID(fmt.Sprintf("prefix %s suffix", fabricated)) {
		t.Errorf("containsSequentialDigitPlaceholderUUID did not flag a body containing the known fabricated placeholder %q", fabricated)
	}
}

// TestSyntheticPlaceholderFixtureGovernance_FailsOnRevertedFixture is the
// committed, automated form of the manual proof code-quality review round 2
// item 3 flagged as missing: someone manually reverted 2 of the 9 recaptured
// fixtures to their pre-aaf8134e9 fabricated bodies, ran the guard, watched
// it fail, then restored the real content and watched it pass -- a real
// verification step, but one that left no durable regression guard in the
// committed diff. This test makes that proof permanent.
//
// The scripts/create-response.json is the fixture under test: its pre-aaf8134e9
// body (`git show aaf8134e9^:internal/mockserver/testdata/synthetic-placeholders/scripts/create-response.json`)
// carries BOTH a literal "SYNTHETIC PLACEHOLDER" marker AND a sequential-digit
// placeholder UUID ("11111111_2222_3333_4444_555555555555") in its Location
// header, so reverting to it exercises both halves of
// recapturedFixtureFabricationProblems in one fixture.
func TestSyntheticPlaceholderFixtureGovernance_FailsOnRevertedFixture(t *testing.T) {
	const rel = "scripts/create-response.json"
	fixturePath := filepath.Join(syntheticPlaceholderDirRelToPkg, filepath.FromSlash(rel))

	current, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}

	// Positive control: the real, currently-genuine on-disk content must
	// pass clean. If this fails, the fixture itself regressed (a job for
	// TestSyntheticPlaceholderFixtureGovernance above), not this test.
	if problems := recapturedFixtureFabricationProblems(rel, current); len(problems) != 0 {
		t.Fatalf("positive control: current on-disk fixture %q unexpectedly flagged as fabricated: %v", rel, problems)
	}

	// The fixture's actual pre-aaf8134e9 fabricated body, captured verbatim
	// via `git show aaf8134e9^:...`, not reconstructed from memory. The
	// Location header's UUID is substituted in below (not a literal in this
	// template) via buildUUIDForGovernanceTest's scrubber-proof parts
	// assembly, same reasoning as guidEmpty/fabricated above.
	const revertedBodyTemplate = `{
  "metadata": {
    "endpoint": "/api/mdm/groups/{organizationGroupUuid}/scripts",
    "method": "POST",
    "description": "SYNTHETIC PLACEHOLDER -- NOT live-captured. The prior description's opening verb ('Captured', followed by the HTTP method) implied a real capture, but the Location UUID was a hand-typed sequential-digit placeholder ('11111111-1111-...') that the ledger's own 2026-08-05 audit (tests/verified-endpoints.json unverified[]) found was never touched by a genuine live run. POST /api/mdm/groups/{organizationGroupUuid}/scripts (Pattern B: 201 + Location header pointing at the new script's GET URL; empty body). Rewritten to an honest, self-evidently-fake placeholder UUID pending genuine live recapture.",
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
      "Location": "/api/mdm/scripts/%s"
    },
    "body": null
  }
}
`
	revertedBody := fmt.Sprintf(revertedBodyTemplate, buildUUIDForGovernanceTest("11111111", "2222", "3333", "4444", "555555555555"))

	// Write the reverted body to a temp file and read it back, mirroring
	// TestSyntheticPlaceholderFixtureGovernance's own os.ReadFile-then-check
	// flow, rather than mutating the real tracked fixture on disk.
	revertedPath := filepath.Join(t.TempDir(), "create-response.json")
	if err := os.WriteFile(revertedPath, []byte(revertedBody), 0o600); err != nil {
		t.Fatalf("write reverted fixture copy to temp dir: %v", err)
	}
	reverted, err := os.ReadFile(revertedPath)
	if err != nil {
		t.Fatalf("read reverted fixture copy: %v", err)
	}

	problems := recapturedFixtureFabricationProblems(rel, reverted)
	if len(problems) == 0 {
		t.Fatalf("expected reverting fixture %q to its old fabricated body to trigger the fabrication guard, got no problems -- the guard would silently accept a real regression", rel)
	}
	if len(problems) != 2 {
		t.Errorf("expected 2 problems (marker match + sequential-digit UUID match) for the planted revert, got %d: %v", len(problems), problems)
	}
}
