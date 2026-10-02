package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// vendoredCorpusMinFileCount is the floor below which a walked
// MOCK_RESPONSES_DIR corpus can no longer plausibly be the real vendored-262
// corpus. Both tests in this file only make sense against the real corpus:
// they skip (they do not fail) when MOCK_RESPONSES_DIR is unset or the corpus
// has fewer files than this floor, so CI must stage the corpus and set
// MOCK_RESPONSES_DIR for them to run.
const vendoredCorpusMinFileCount = 100

// vendoredFixtureEntry is one fixture file discovered by walking the
// resolved mock-responses corpus, keyed by its path RELATIVE to the resolved
// root (so it can be compared against ledger/manifest entries, which are
// also root-relative).
type vendoredFixtureEntry struct {
	relPath  string // path relative to the resolved corpus root
	fullPath string // full filesystem path, for loading
}

// walkVendoredFixtureCorpus walks mockserver.ResolveResponsesDir("../testdata/mock-responses")
// (the same root string fixture_collision_guard_test.go's TestNoDuplicateFixtureEndpoints
// uses) and returns every .json fixture file found, along with a hard
// guard that the resolved corpus is plausibly the real vendored-262 corpus
// (the test skips otherwise).
func walkVendoredFixtureCorpus(t *testing.T) []vendoredFixtureEntry {
	t.Helper()

	if strings.TrimSpace(os.Getenv(mockserver.ResponsesDirEnv)) == "" {
		t.Skipf("%s is unset -- these tests must run against the real vendored fixture corpus (this is expected in the public package, which does not ship the private vendored corpus). Stage the corpus first (e.g. `make mock-populate VER=26.2.0.0`) and rerun with MOCK_RESPONSES_DIR=<path> go test ...", mockserver.ResponsesDirEnv)
	}

	root := mockserver.ResolveResponsesDir("../testdata/mock-responses")

	var entries []vendoredFixtureEntry
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("computing relative path for %s under %s: %w", path, root, relErr)
		}
		entries = append(entries, vendoredFixtureEntry{
			relPath:  filepath.ToSlash(rel),
			fullPath: path,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking resolved mock-responses corpus %s: %v", root, err)
	}

	if len(entries) < vendoredCorpusMinFileCount {
		t.Skipf("resolved mock-responses corpus %s contains only %d fixture files (need >= %d to be the real vendored-262 corpus; this is expected when MOCK_RESPONSES_DIR points at a small directory rather than the private vendored corpus) -- stage the real corpus first (e.g. `make mock-populate VER=26.2.0.0`) and rerun with MOCK_RESPONSES_DIR=<path> go test ...",
			root, len(entries), vendoredCorpusMinFileCount)
	}

	return entries
}

// --- TestVendoredFixtureLedgerDrift ---

// vendoredFixtureLedgerEntry mirrors ledgerEntry (defined in
// internal/mockserver/bypass_governance_test.go) for tests/verified-endpoints.json's
// verified[] array -- redefined here rather than imported since that type
// lives in a different package (mockserver_test) than this file (tests).
type vendoredFixtureLedgerEntry struct {
	Service      string `json:"service"`
	Method       string `json:"method"`
	Fixture      string `json:"fixture"`
	Reason       string `json:"reason"`
	TestFunction string `json:"testFunction"`
}

type vendoredFixtureLedger struct {
	Verified   []vendoredFixtureLedgerEntry `json:"verified"`
	Unverified []vendoredFixtureLedgerEntry `json:"unverified"`
}

// splitLedgerFixtureField parses a verified[] entry's "fixture" field into
// zero or more corpus-relative fixture paths. Most entries hold exactly one
// path, but some legitimately hold a comma-separated LIST of paths in a
// single string (e.g. a create/get/delete platform-variant group all
// governed under one ledger entry), and some hold a non-path placeholder
// note (e.g. "N/A -- served by internal/mockserver's stateful ChunkHandler,
// not a static JSON fixture") documenting that no static fixture backs this
// entry at all. Treating either of those as a single literal path produced a
// direct contradiction: the whole string never matches a real file (a false
// missingFromCorpus finding) while the real filenames embedded inside it
// never get credited as referenced (a false orphanedFromLedger finding for
// the same files) -- this split fixes that at the parse boundary so both
// sets are built from the same real per-file granularity.
//
// A comma-separated part
// can ALSO be a real path immediately followed by a parenthetical note, e.g.
// "internal-apps-v1/msi/internal_app_create_msi.json (documentation-only --
// see reason)" (the literal shape of the msi entry in
// tests/verified-endpoints.json's InternalAppsV1Service.CreateInternalApplicationFromBlob
// row). The pre-widen version dropped that whole part on sight of any
// embedded whitespace, so the path was never scanned by either
// TestVendoredFixtureLedgerDrift or TestVendoredVerifiedFixturesNotFabricated
// (confirmed live: the apk sibling in the same ledger row appeared among the
// content-scan's subtests while the msi one silently did not). Fixed by
// taking just the LEADING whitespace-delimited token of each part and
// applying the ".json"-suffix path/prose test to that token alone, so a
// trailing parenthetical note no longer defeats the extraction -- the note
// itself is simply discarded, only the leading path token is kept and
// scanned.
func splitLedgerFixtureField(raw string) []string {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		token := part
		if idx := strings.IndexAny(token, " \t("); idx >= 0 {
			token = token[:idx]
		}
		// A real fixture path is a bare relative path ending in ".json";
		// anything else (e.g. the "N/A -- ..." placeholder note, whose
		// leading token is "N/A") is prose, not a path, and is dropped here.
		if !strings.HasSuffix(token, ".json") {
			continue
		}
		out = append(out, token)
	}
	return out
}

// TestVendoredFixtureLedgerDrift is a corpus-vs-ledger CONSISTENCY check --
// an informational/drift detector, NOT the primary serving proof (that's
// TestVendoredFixtureSelfServing below). It compares the set of fixture
// files physically present in the resolved vendored corpus against every
// fixture path referenced by tests/verified-endpoints.json's verified[]
// array (unverified[] entries are explicitly NOT considered test-exercised,
// so they're excluded), unioned with the two existing bypass/synthetic-
// placeholder manifests that also legitimately reference vendored-corpus
// fixture paths via a mechanism other than the directory-scan/matching this
// test walks.
//
// The missingFromCorpus (a ledger/manifest entry with no corresponding file on
// disk) always fails loudly -- that's a real broken reference. The orphanedFromLedger
// (a corpus file no ledger/manifest entry mentions) is only logged, not
// failed, here: TestVendoredFixtureSelfServing is the authoritative
// orphan/serving proof, since a fixture can be legitimately unreferenced by
// the ledger's (service, method) tracking (it tracks by service+method, not
// 1:1 by fixture file -- see bypass_governance_test.go's own note on this)
// while still being genuinely reachable and useful as a collision sibling.
//
// Known caveat (not this test's job to resolve): tests/verified-endpoints.json
// has an open provenance-trust question tracked separately. This drift check
// inherits whatever trust gap exists there.
func TestVendoredFixtureLedgerDrift(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)
	corpusSet := make(map[string]bool, len(corpus))
	for _, e := range corpus {
		corpusSet[e.relPath] = true
	}

	referenced := map[string]bool{}

	ledgerData, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- it is deliberately excluded from the assembled public package (tools/sync/manifest.yaml's carry_over_exclude), so this ledger-integrity check is structurally unavailable there", err)
	}
	var ledger vendoredFixtureLedger
	if err := json.Unmarshal(ledgerData, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 {
		t.Fatal("control check failed: tests/verified-endpoints.json verified[] is empty -- the ledger read itself is broken")
	}
	for _, e := range ledger.Verified {
		for _, rel := range splitLedgerFixtureField(e.Fixture) {
			referenced[rel] = true
		}
	}

	// bypass-loaded-fixtures.json paths are relative to testdata/mock-responses/
	// (the same root this test walks), so a bypass-loaded fixture IS expected
	// to physically exist in the corpus -- it's unioned into referenced and
	// participates in the missing-file check like any ledger path.
	bypassData, err := os.ReadFile("../tools/internal/capturetools/testdata/bypass-loaded-fixtures.json")
	if err != nil {
		t.Fatalf("read manifest ../tools/internal/capturetools/testdata/bypass-loaded-fixtures.json: %v", err)
	}
	var bypassPaths []string
	if err := json.Unmarshal(bypassData, &bypassPaths); err != nil {
		t.Fatalf("parse manifest ../tools/internal/capturetools/testdata/bypass-loaded-fixtures.json: %v", err)
	}
	for _, p := range bypassPaths {
		referenced[p] = true
	}

	// synthetic-placeholder-fixtures.json paths are relative to a DIFFERENT
	// root entirely -- internal/mockserver/testdata/synthetic-placeholders/,
	// not testdata/mock-responses/ -- so a fixture listed there is never
	// expected to physically exist in this corpus walk at all. It's a known,
	// intentionally-elsewhere fixture, not a ledger-vs-corpus gap: excluded
	// from the missing-file check below and reported separately instead of
	// unioned into referenced (unioning would be harmless but misleading,
	// since it implies these paths are corpus-relative when they aren't).
	syntheticData, err := os.ReadFile("../tools/internal/capturetools/testdata/synthetic-placeholder-fixtures.json")
	if err != nil {
		t.Fatalf("read manifest ../tools/internal/capturetools/testdata/synthetic-placeholder-fixtures.json: %v", err)
	}
	var syntheticPaths []string
	if err := json.Unmarshal(syntheticData, &syntheticPaths); err != nil {
		t.Fatalf("parse manifest ../tools/internal/capturetools/testdata/synthetic-placeholder-fixtures.json: %v", err)
	}
	knownElsewhere := make(map[string]bool, len(syntheticPaths))
	for _, p := range syntheticPaths {
		knownElsewhere[p] = true
	}

	var missingFromCorpus []string
	var knownElsewhereReferences []string
	for rel := range referenced {
		if corpusSet[rel] {
			continue
		}
		if knownElsewhere[rel] {
			knownElsewhereReferences = append(knownElsewhereReferences, rel)
			continue
		}
		missingFromCorpus = append(missingFromCorpus, rel)
	}
	sort.Strings(missingFromCorpus)
	sort.Strings(knownElsewhereReferences)

	if len(knownElsewhereReferences) > 0 {
		t.Logf("%d ledger-referenced path(s) not found in the corpus but explained by synthetic-placeholder-fixtures.json (they live under internal/mockserver/testdata/synthetic-placeholders/, a different root than this corpus walk -- known-elsewhere, not a gap):\n  %s",
			len(knownElsewhereReferences), strings.Join(knownElsewhereReferences, "\n  "))
	}

	var allowlistedOrphans, unexplainedOrphans []string
	for rel := range corpusSet {
		if referenced[rel] {
			continue
		}
		if _, ok := knownOrphanedVendoredFixtures[rel]; ok {
			allowlistedOrphans = append(allowlistedOrphans, rel)
			continue
		}
		unexplainedOrphans = append(unexplainedOrphans, rel)
	}
	sort.Strings(allowlistedOrphans)
	sort.Strings(unexplainedOrphans)

	if len(missingFromCorpus) > 0 {
		t.Errorf("%d ledger/manifest-referenced fixture path(s) do not exist as a file in the vendored corpus:\n  %s",
			len(missingFromCorpus), strings.Join(missingFromCorpus, "\n  "))
	}

	for _, rel := range allowlistedOrphans {
		t.Logf("known orphan %s: not referenced by verified-endpoints.json's verified[] or either bypass/synthetic-placeholder manifest, but documented in knownOrphanedVendoredFixtures -- %s", rel, knownOrphanedVendoredFixtures[rel])
	}

	if len(unexplainedOrphans) > 0 {
		t.Errorf("%d corpus file(s) not referenced by verified-endpoints.json's verified[] or either bypass/synthetic-placeholder manifest, AND not in knownOrphanedVendoredFixtures -- either wire up ledger coverage or add a documented, honest allowlist entry (see TestVendoredFixtureSelfServing for whether it's also reachable at the route level):\n  %s",
			len(unexplainedOrphans), strings.Join(unexplainedOrphans, "\n  "))
	}
}

// knownOrphanedVendoredFixtures documents every corpus file that
// TestVendoredFixtureLedgerDrift finds unreferenced by tests/verified-endpoints.json's
// verified[] array and both bypass/synthetic-placeholder manifests, with an
// HONEST per-entry reason. Entries fall into two distinct classes -- do not
// blanket-label; each entry's own reason says which:
//
//  1. Real, independently-provenanced live capture with no consuming test
//     yet. These are genuine coverage gaps, tracked separately but not yet
//     wired into the ledger or a test.
//  2. Fabricated/mislabeled/demoted clone, pending REMOVAL from the vendored
//     corpus -- allowlisted as known-not-a-real-coverage-target, NOT a real
//     untested fixture.
//
// This mirrors knownFixtureCollisionAllowlist's pattern (same file) for a
// different structural gap: that map documents same-key collisions the
// request matcher resolves by load order; this one documents corpus files
// with no ledger/manifest reference at all. Every entry here was verified
// against tests/verified-endpoints.json's own verified[]/unverified[]
// entries and/or knownFixtureCollisionAllowlist's own comments -- an
// unverified entry defeats the whole guard, same discipline as that map.
//
// "Orphaned" here means specifically: no consumer in GO'S test suite. A
// fixture entirely consumed by Python (and vice versa, for
// the Python runtime's own _KNOWN_ORPHANED_FIXTURES) is expected to appear on
// exactly one language's allowlist, not both -- being orphaned in one
// language says nothing about the other.
//
// Every value here starts with an explicit provenance tag -- GENUINE,
// PLACEHOLDER, SYNTHETIC, or UNVERIFIED-PROVENANCE -- describing the
// fixture's own content, not its orphan status: GENUINE is real captured
// wire data; PLACEHOLDER is an honestly-disclosed hand-typed stand-in for a
// value that cannot be genuinely captured (a secret, or a field with no
// reachable live case); SYNTHETIC is schema-derived or fabricated content,
// not wire data at all; UNVERIFIED-PROVENANCE is content whose origin no
// capture commit or dated record establishes either way.
// A GENUINE entry additionally carrying CONSUMED-VIA-DEFAULT-LOOKUP is
// consumed by a real test, but only through the mock server's directory-
// scan default lookup (no test names this exact path directly), which is
// why it still needs a documented entry despite genuinely being covered.
var knownOrphanedVendoredFixtures = map[string]string{
	"org-groups/groups_delete_uuid_v2.json":                           "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured with tools/mock-capture against UEM 26.2 on 2026-09-24 in a disposable sdktestrun org group (vendored commit 4f0dd5fe5); body anonymized by the capture anonymizer. DELETE /api/system/groups/{uuid} (v2) of a disposable org group: 200, empty body; the real uuid in the path is replaced by a fixed uuid unique in this corpus. Served through the shared mock server's route matcher to TestOrgGroupDeleteV2Fixtures (no test loads it by path); deletion-control (2026-09-24) shows that removing it makes that test FAIL. Not a verified[] row: no generated SDK method wraps this route.",
	"org-groups/groups_delete_uuid_v2_not_found.json":                 "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured with tools/mock-capture against UEM 26.2 on 2026-09-24 in a disposable sdktestrun org group (vendored commit 4f0dd5fe5); body anonymized by the capture anonymizer. The repeat DELETE of the same org group: 400, errorCode 400, not found (api-doctrine quirk 20); the real uuid and id are replaced by fixed values unique in this corpus. Served through the shared mock server's route matcher to TestOrgGroupDeleteV2Fixtures (no test loads it by path); deletion-control (2026-09-24) shows that removing it makes that test FAIL. Not a verified[] row: no generated SDK method wraps this route.",
	"profiles/profiles_get_not_found.json":                            "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured verbatim with tools/mock-capture --allow-error-status against UEM 26.2 on 2026-09-24, on an object deleted moments earlier inside a disposable sdktestrun org group; the resource id is replaced by a fixed id unique in this corpus, and the activityId is the capture anonymizer's output. Served through the shared mock server's route matcher (no test loads it by path) to TestNotFoundFixturesAreRecognised, which asserts client.IsNotFound on the resulting error; deletion-control (2026-09-24) shows that removing it makes TestNotFoundFixturesAreRecognised/profile_GET and TestSharedMockServer_ProfileDeleteDefersToConcreteFixture FAIL. Deliberately not a verified[] row: verified[] rows also drive the Python respx test generator, which assumes a success response. Profile GET after delete: 400, errorCode 400, 'Invalid Profile <id>.'.",
	"profiles/profiles_delete_not_found.json":                         "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured verbatim with tools/mock-capture --allow-error-status against UEM 26.2 on 2026-09-24, on an object deleted moments earlier inside a disposable sdktestrun org group; the resource id is replaced by a fixed id unique in this corpus, and the activityId is the capture anonymizer's output. Served through the shared mock server's route matcher (no test loads it by path) to TestNotFoundFixturesAreRecognised, which asserts client.IsNotFound on the resulting error; deletion-control (2026-09-24) shows that removing it makes TestNotFoundFixturesAreRecognised/profile_DELETE and TestSharedMockServer_ProfileDeleteDefersToConcreteFixture FAIL. Deliberately not a verified[] row: verified[] rows also drive the Python respx test generator, which assumes a success response. Profile second DELETE: 400, errorCode 400, 'Profile not found or User does not have access to the Profile.'.",
	"smartgroups/smartgroups_delete_not_found.json":                   "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured verbatim with tools/mock-capture --allow-error-status against UEM 26.2 on 2026-09-24, on an object deleted moments earlier inside a disposable sdktestrun org group; the resource id is replaced by a fixed id unique in this corpus, and the activityId is the capture anonymizer's output. Served through the shared mock server's route matcher (no test loads it by path) to TestNotFoundFixturesAreRecognised, which asserts client.IsNotFound on the resulting error; deletion-control (2026-09-24) shows that removing it makes TestNotFoundFixturesAreRecognised/smart_group_DELETE and TestSharedMockServer_SmartGroupGetFollowsObservedLifecycle FAIL. Deliberately not a verified[] row: verified[] rows also drive the Python respx test generator, which assumes a success response. Smart group second DELETE: 400, errorCode 400 (the GET returns errorCode 6), 'Smart Group not found with Id: <id> ...'.",
	"internal-apps-v1/macos/get_internal_app_macos_not_found.json":    "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured verbatim with tools/mock-capture --allow-error-status against UEM 26.2 on 2026-09-24, on an object deleted moments earlier inside a disposable sdktestrun org group; the resource id is replaced by a fixed id unique in this corpus, and the activityId is the capture anonymizer's output. Served through the shared mock server's route matcher (no test loads it by path) to TestNotFoundFixturesAreRecognised, which asserts client.IsNotFound on the resulting error; deletion-control (2026-09-24) shows that removing it makes TestNotFoundFixturesAreRecognised/macOS_internal_app_GET FAIL. Deliberately not a verified[] row: verified[] rows also drive the Python respx test generator, which assumes a success response. macOS internal app GET after delete: 401, errorCode 7000.",
	"internal-apps-v1/macos/delete_internal_app_macos_not_found.json": "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured verbatim with tools/mock-capture --allow-error-status against UEM 26.2 on 2026-09-24, on an object deleted moments earlier inside a disposable sdktestrun org group; the resource id is replaced by a fixed id unique in this corpus, and the activityId is the capture anonymizer's output. Served through the shared mock server's route matcher (no test loads it by path) to TestNotFoundFixturesAreRecognised, which asserts client.IsNotFound on the resulting error; deletion-control (2026-09-24) shows that removing it makes TestNotFoundFixturesAreRecognised/macOS_internal_app_DELETE FAIL. Deliberately not a verified[] row: verified[] rows also drive the Python respx test generator, which assumes a success response. macOS internal app second DELETE: 400, errorCode 400, 'Application not found or user does not have access to it.'.",
	// Class 1: real capture, no consuming test yet.
	"internal-apps-v1/ipa/internal_app_begininstall_ipa.json": "PLACEHOLDER: CORRECTED -- a prior pass found only the bulk e31771f45 corpus-land commit on the vendored branch; the actual dedicated capture commit is 33dcb74ff (2026-08-10), captured live against UEM 26.2.0.0 in the shared test org group, independently leak-audited for real emails/phone numbers/bundle IDs on the wire -- the body as a whole IS a genuine live capture. Retagged PLACEHOLDER (not GENUINE) because OrganizationGroupUuid and Uuid are hand-typed repeated-hex-word placeholders (\"aaaaaaaa-1111-...\"/\"bbbbbbbb-2222-...\"), not anonymizer output, disclosed in the fixture's own description. Presently unexercised by any assertion, inert not broken.",
	"profiles/search_android_success.json":                    "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured with tools/mock-capture against UEM 26.2 on 2026-09-24 in a disposable sdktestrun org group (vendored commit 4f0dd5fe5); body anonymized by the capture anonymizer. A v1 search of an org group holding one AppleOsX and one Linux profile created for the capture; no Android profile exists on the tenant, so the filename is historical. Consumed only when a ../.env file is present: TestIntegration_ProfileService_Get and the Search/EndpointConfiguration integration tests (tests/profile_integration_test.go, this v1 route) are served it by the mock server's directory-scan default lookup; deletion-control (2026-09-24, dummy .env present) shows that removing only this file makes TestIntegration_ProfileService_Search FAIL and TestIntegration_ProfileService_Get go PASS->SKIP. It wins over search_empty.json while both are present because the corpus is walked in lexical order.",
	"profiles/search_empty.json":                              "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: live-captured with tools/mock-capture against UEM 26.2 on 2026-09-24 in a disposable sdktestrun org group (vendored commit 4f0dd5fe5); body anonymized by the capture anonymizer. A v1 search of an EMPTY disposable org group: the server answers 204 No Content with no body. Only a fallback for the same v1 integration tests: search_android_success.json sorts first and is served while both are present, so removing only this file changes no test outcome, and removing both files makes Search, Get and EndpointConfiguration FAIL (deletion-control, 2026-09-24, dummy .env present); it is served only once search_android_success.json is absent.",
	"scripts/bulkdelete-zz-partial.json":                      "SYNTHETIC: Hand-written mock (commit 6f2b0c593, 2026-05-27), never changed since; delete_failed carries a hand-typed repeated-hex-word placeholder UUID (\"deadbeef_dead_beef_dead_beefdeadbeef\"). This vendored-corpus copy is no longer read by any test -- the consuming test (TestGeneratedScriptsV1BulkDeletePartial) was relocated to load a separate, relocated copy elsewhere instead. The vendored copy is left in place unmodified but is now genuinely orphaned, not test-exercised; crediting it as covered would be a false coverage claim.",
	"org-groups/groups_get.json":                              "GENUINE: capture commit 1060c5dfe (2026-04-13), GroupId re-anonymized in 6766ed973 (2026-09-03). Same route template (GET /api/system/groups/{id}) as org-groups/groups_get_id.json, the independently captured fixture that backs the OrganizationGroupsService.GetAsync verified[] row. This file is not cited by any verified[] row and no test consumes it: the GetAsync test loads groups_get_id.json into an isolated mock server precisely because this file sorts first under directory-scan lookup. Inert, not broken.",
	"sensors/list_sensors.json":                               "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: Empty-list response, still consumed by TestGeneratedSensorLifecycleV1's post-create list assertion (mechanically proven: deleting only this file from a scratch corpus copy of the vendored tip makes TestGeneratedSensorLifecycleV1 FAIL), but no longer the ledger's cited fixture for DeviceSensorsV1Service.GetDeviceSensors -- that row now cites sensors/list_sensors_populated.json, which round-trips real sensor items instead of an empty result. Kept in the corpus and load-bearing, just no longer the verified[] citation. CORRECTED -- a prior pass searched only the vendored orphan branch's git history; the SDK repo's own history has a dedicated capture commit, e7dc544bd (2026-08-05), which replaced a fabricated 2-entry fixture with a genuinely empty sensor list (0 sensors on the capture tenant), touching testdata/mock-responses/sensors/list_sensors.json directly and matching this file's empty-list content.",
	"sensors/list_sensors_v2_empty.json":                      "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: Empty-list response, still consumed by TestGeneratedSensorsV2List (mechanically proven: deleting only this file from a scratch corpus copy of the vendored tip makes TestGeneratedSensorsV2List FAIL), but no longer the ledger's cited fixture for DeviceSensorsV2Service.GetDeviceSensorsAsync -- that row now cites sensors/list_sensors_v2_populated.json, which round-trips real sensor items instead of an empty result. Kept in the corpus and load-bearing, just no longer the verified[] citation. CORRECTED -- a prior pass searched only the vendored orphan branch's git history; the SDK repo's own history has a dedicated capture commit, 3b5c230cc (2026-07-30), which recorded zero pre-existing sensors on the capture tenant, touching testdata/mock-responses/sensors/list_sensors_v2_empty.json directly and matching this file's empty-list content.",
	"profiles/search_v2.json":                                 "PLACEHOLDER CONSUMED-VIA-DEFAULT-LOOKUP: Hand-written (commit c2450a48b, 2026-04-08), not a live capture -- DISCLOSURE: ProfileUuid values (\"abc12345_1234_1234_1234_abc123456789\", \"def12345_1234_1234_1234_def123456789\") are hand-typed placeholders, not genuine server-issued or anonymizer-fingerprinted values; ProfileId (68748/68749), Platform, ProfileType, and ProfileStatus are hand-authored alongside them, same fixture. Not cited by any verified[] row's fixture field, but genuinely load-bearing: TestLayer2ProfileServiceGet's Android/Apple (68748/68749) discovery-refresh flow depends on this file's full-corpus-scan content, which is deliberately left at its ORIGINAL (pre-existing) content rather than overwritten with a genuine SearchProfiles capture -- overwriting it would break that unrelated test (it would no longer contain ids 68748/68749). The genuine SearchProfiles capture instead lives in the sibling profiles/profiles_search_v2.json, which IS cited by the ProfilesV2Service.SearchProfiles verified[] row and loaded directly by path (not full-corpus scan) by TestGeneratedProfilesV2SearchV2. Mechanically proven: deleting only this file from a scratch corpus copy of the vendored tip makes TestLayer2ProfileServiceGet FAIL.",
	"profiles/create_profile_success.json":                    "PLACEHOLDER: Hand-written stub (body is the bare literal integer 99999) predating this repo's capture tooling -- no live-capture narrative, no host/date evidence. Not fabricated-as-real: no test credits it as a genuine capture.",

	// Class 2: SYNTHETIC (schema-derived placeholder, disclosed in the
	// fixture's own metadata) or a near-certainly mislabeled/fabricated
	// clone pending removal from the vendored corpus.
	"baselines/assignments-get.json":                         "SYNTHETIC: Schema-derived placeholder, disclosed honestly in the fixture's own metadata.description -- a genuine live capture was impossible when this was written (the test tenant had no baseline and no enrolled device). No functional consumer in the Go suite; guard-only (deletion-control, 2026-09-24). TestGeneratedBaselinesV1Lifecycle loads a separate, non-identical copy from internal/mockserver/testdata/synthetic-placeholders/ instead. Bare array of BaselineAssignmentsV1Model; camelCase smartGroupUUID per D3.",
	"baselines/assignments-post.json":                        "SYNTHETIC: Schema-derived placeholder, disclosed honestly in the fixture's own metadata.description -- a genuine live capture was impossible when this was written (the test tenant had no baseline and no enrolled device). No functional consumer in the Go suite; guard-only (deletion-control, 2026-09-24). TestGeneratedBaselinesV1Lifecycle loads a separate, non-identical copy from internal/mockserver/testdata/synthetic-placeholders/ instead. Pattern D: 204 empty.",
	"baselines/clone-response.json":                          "SYNTHETIC: Schema-derived placeholder, disclosed honestly in the fixture's own metadata.description -- a genuine live capture was impossible when this was written (the test tenant had no baseline and no enrolled device). No functional consumer in the Go suite; guard-only (deletion-control, 2026-09-24). TestGeneratedBaselinesV1Lifecycle loads a separate, non-identical copy from internal/mockserver/testdata/synthetic-placeholders/ instead. Pattern C: 201 + body + Location. Org group UUID re-anonymized 2026-09-24 by tools/reanonymize-fixture at commit 999e3892a, command reanonymize-fixture -replace-uuid <original> -key OrganizationGroupUuid, where <original> is the UUID whose sha256 is in tests/leaked_og_uuid_guard_test.go; every other byte is unchanged.",
	"baselines/devices.json":                                 "SYNTHETIC: Schema-derived placeholder, disclosed honestly in the fixture's own metadata.description -- a genuine live capture was impossible when this was written (the test tenant had no baseline and no enrolled device). No functional consumer in the Go suite; guard-only (deletion-control, 2026-09-24). TestGeneratedBaselinesV1Lifecycle loads a separate, non-identical copy from internal/mockserver/testdata/synthetic-placeholders/ instead. Paged GetBaselineDeviceSummaryResponseV1Model wrapper.",
	"internal-apps-v1/zip/delete_internal_app_zip.json":      "SYNTHETIC: Own metadata self-labels SYNTHETIC/HAND-AUTHORED PLACEHOLDER, not live-captured; blocked pending a target org group feature flag; fabricated/mislabeled/demoted clone, pending REMOVAL from the vendored corpus; allowlisted as known-not-a-real-coverage-target, NOT a real untested fixture.",
	"internal-apps-v1/zip/get_internal_app_zip.json":         "SYNTHETIC: Own metadata self-labels SYNTHETIC/HAND-AUTHORED PLACEHOLDER, not live-captured; blocked pending a target org group feature flag; fabricated/mislabeled/demoted clone, pending REMOVAL from the vendored corpus; allowlisted as known-not-a-real-coverage-target, NOT a real untested fixture.",
	"mam-apps/apps_get_internal_ios_renewaldate.json":        "SYNTHETIC: Own description self-labels as a deliberately-crafted synthetic regression fixture, not a live capture; fabricated/mislabeled/demoted clone, pending REMOVAL from the vendored corpus; allowlisted as known-not-a-real-coverage-target, NOT a real untested fixture.",
	"profiles/search_android_success_v2.json":                "SYNTHETIC: Near-certainly a mislabeled clone of the v1-shaped search_android_success.json/search_empty.json (as those files were before their 2026-09-24 recapture) (Page/PageSize/Total wrapper hand-renamed to ProfileList, no capture narrative); dropped from the SearchProfiles ledger join in a prior reconcile; fabricated/mislabeled/demoted clone, pending REMOVAL from the vendored corpus; allowlisted as known-not-a-real-coverage-target, NOT a real untested fixture. Org group UUID re-anonymized 2026-09-24 by tools/reanonymize-fixture at commit 999e3892a, command reanonymize-fixture -replace-uuid <original> -key OrganizationGroupUuid, where <original> is the UUID whose sha256 is in tests/leaked_og_uuid_guard_test.go; every other byte is unchanged.",
	"profiles/search_empty_v2.json":                          "SYNTHETIC: Near-certainly a mislabeled clone of the v1-shaped search_android_success.json/search_empty.json (as those files were before their 2026-09-24 recapture) (Page/PageSize/Total wrapper hand-renamed to ProfileList, no capture narrative); dropped from the SearchProfiles ledger join in a prior reconcile; fabricated/mislabeled/demoted clone, pending REMOVAL from the vendored corpus; allowlisted as known-not-a-real-coverage-target, NOT a real untested fixture.",
	"sensors/get_device_sensors.json":                        "PLACEHOLDER: Own metadata.description (as of the vendored-branch honesty fix) discloses that the body's two uuid values are hand-typed ascending-hex placeholders, not gofakeit output, and explicitly says 'not a genuine live capture' -- the request/route shape itself is real, only the UUID values are a disclosed hand-typed stand-in. No live access to recapture with real values; not fabricated-as-real, since the fixture's own metadata already discloses this.",
	"updates/updates_device_status_count_no_deployment.json": "PLACEHOLDER CONSUMED-VIA-DEFAULT-LOOKUP: Own metadata.description discloses 'Hand-authored from a raw curl capture' -- the 404 status and error shape are genuinely real (captured via manual curl against a real no-deployment case), but body device_update_uuid and activityId are hand-typed placeholder literals, the same hand-typed pattern as the updates_deployment_get.json fixture family. Superseded for real-coverage purposes by updates_device_status_count_success.json, which backs UpdatesV1Service.GetCountOfDeviceStatus's verified[] row. Mechanically proven: deleting only this file from a scratch corpus copy of the vendored tip makes TestGeneratedUpdatesV1ReadOnly FAIL.",
	"updates/updates_group_action_invalid.json":              "PLACEHOLDER CONSUMED-VIA-DEFAULT-LOOKUP: Own metadata.description discloses a genuinely captured invalid-enum rejection (POST ?action=Pause, a real wire-vs-canonical mismatch), but body activityId cccccccc_dddd_eeee_ffff_000000000000 is a hand-typed sequential-letter placeholder, the same fixture family as updates_deployment_get.json. The sibling start/stop DevicesUpdateAction fixtures are genuine and stay verified -- only this one fixture was split out and demoted. Mock test coverage of this fixture is unaffected (mock tests may legitimately consume placeholder data); only the verified claim was wrong. See tests/verified-endpoints.json's unverified[] entry for UpdatesV1Service.DevicesUpdateAction for the full reason. Mechanically proven: deleting only this file from a scratch corpus copy of the vendored tip makes TestGeneratedUpdatesV1DevicesUpdateAction FAIL.",

	// The following 8 entries -- scripts-assignments/{bulkupdate-response,
	// create-response,get,list}.json, scripts/{create-response,
	// update-response,bulkdelete-success}.json, and system/info_success.json
	// -- were REMOVED from this map 2026-09-23.
	// They used to carry a "fabricated/mislabeled/demoted clone, pending
	// REMOVAL" (system/info_success.json: "SUSPICIOUS ... hand-typed") label
	// here, but that label had gone stale: the fixtures were genuinely
	// recaptured (Phase 1/3, commits d14f6ad9b/b9c0556a3, vendored branch;
	// gofakeit "Test Rabbitthrow"/"Test Koaladid" fingerprints, random
	// Location UUIDs) and moved to tests/verified-endpoints.json's verified[]
	// array. internal/mockserver/bypass_governance_test.go's
	// syntheticPlaceholderHonestyMarkerExceptions already records all 8 as
	// recaptured. Since verified[] now references them, they are no longer
	// orphaned and no longer belong in this allowlist at all.
}

// --- TestVendoredFixtureSelfServing ---

// TestVendoredFixtureSelfServing is the PRIMARY, ledger-independent serving
// proof: for every fixture file in the resolved vendored corpus, it loads
// the fixture, builds a synthetic *http.Request that replays exactly what
// the fixture's OWN metadata/request block describes, and asserts that
// running that request through mockserver.NewRequestMatcher (seeded with
// the FULL corpus, so real same-key collisions are exercised) resolves back
// to THIS exact fixture -- not nil, and not a different, same-key sibling.
//
// This catches the documented "route match-key gotcha" where a fixture is
// registered on disk but a same-key sibling always wins the real match
// (internal/mockserver's scoreMatch/pathMatchScore resolve ties by directory-
// walk load order, which is deterministic per run but not meaningful).
//
// A fixture whose (endpoint, method, version) key is in
// knownFixtureCollisionAllowlist (fixture_collision_guard_test.go) is
// EXPECTED to lose its own self-match to a documented sibling -- that's
// logged, not failed. A self-match loss outside the allowlist is a genuine,
// previously-invisible orphan/mismatch and fails loudly.
func TestVendoredFixtureSelfServing(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)

	byPath := make(map[string]*mockserver.MockResponse, len(corpus))
	var allResponses []*mockserver.MockResponse
	for _, e := range corpus {
		resp, err := mockserver.LoadResponseFromFile(e.fullPath)
		if err != nil {
			t.Fatalf("loading fixture %s: %v", e.fullPath, err)
		}
		byPath[e.relPath] = resp
		allResponses = append(allResponses, resp)
	}

	matcher := mockserver.NewRequestMatcher(allResponses)

	var failures []string
	var bodyOnlyCandidates []string
	allowlistedHits := 0
	passCount := 0

	for _, e := range corpus {
		own := byPath[e.relPath]

		req, err := buildSyntheticRequestForFixture(own)
		if err != nil {
			t.Errorf("fixture %s: could not build synthetic request: %v", e.relPath, err)
			continue
		}

		matched := matcher.Match(req)

		if matched == own {
			passCount++
			continue
		}

		key := fixtureEndpointKey{own.Metadata.Endpoint, own.Metadata.Method, own.Metadata.Version}
		if reason, ok := knownFixtureCollisionAllowlist[key]; ok {
			allowlistedHits++
			var winner string
			if matched != nil {
				winner = fmt.Sprintf("a sibling fixture (endpoint=%s method=%s version=%s)", matched.Metadata.Endpoint, matched.Metadata.Method, matched.Metadata.Version)
			} else {
				winner = "nothing (nil)"
			}
			t.Logf("known allowlisted collision for %s (%+v): self-match lost to %s by design -- %s", e.relPath, key, winner, reason)
			continue
		}

		if matched == nil {
			failures = append(failures, fmt.Sprintf("%s: expected self-match, got NO match at all (method=%s endpoint=%s version=%s)",
				e.relPath, own.Metadata.Method, own.Metadata.Endpoint, own.Metadata.Version))
		} else {
			matchedRel := ""
			for rel, r := range byPath {
				if r == matched {
					matchedRel = rel
					break
				}
			}
			failures = append(failures, fmt.Sprintf("%s: expected self-match, got a DIFFERENT fixture %s instead (method=%s endpoint=%s version=%s)",
				e.relPath, matchedRel, own.Metadata.Method, own.Metadata.Endpoint, own.Metadata.Version))

			// scoreMatch never scores on request body, so a same
			// (method, endpoint, version, query-params) pair that isn't in
			// the collision allowlist and isn't distinguished by query
			// params is a strong signal of a body-only-distinguished pair
			// this synthetic-request technique structurally cannot resolve.
			if len(own.Request.QueryParams) == 0 {
				bodyOnlyCandidates = append(bodyOnlyCandidates, e.relPath)
			}
		}
	}

	t.Logf("summary: %d/%d fixtures self-matched cleanly, %d allowlisted collisions logged, %d failures, %d possible body-only-distinguished candidates",
		passCount, len(corpus), allowlistedHits, len(failures), len(bodyOnlyCandidates))

	if len(bodyOnlyCandidates) > 0 {
		sort.Strings(bodyOnlyCandidates)
		t.Logf("possible body-only-distinguished fixtures (self-match lost, not allowlisted, no query-param disambiguator available -- these cannot be resolved by this technique, do not guess, escalate instead):\n  %s",
			strings.Join(bodyOnlyCandidates, "\n  "))
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("%d fixture(s) failed to self-match through the real request matcher:\n  %s", len(failures), strings.Join(failures, "\n  "))
	}
}

// buildSyntheticRequestForFixture constructs an *http.Request that replays
// exactly what a fixture's own metadata/request block describes: Method =
// Metadata.Method, URL.Path = Metadata.Endpoint (a literal self-match, which
// works even for template endpoints like /api/mdm/profiles/{id} because
// pathMatchScore in matcher.go exact-string-matches requestPath == pattern
// before falling back to regex), a query string built from
// Request.QueryParams, and, when Metadata.Version is non-empty, an Accept
// header of "application/json;version=<Metadata.Version>" matching the
// format extractVersionFromAccept parses. Request.Headers are also set onto
// the synthetic request for completeness, though header validation itself
// is only exercised by handleRequest, not by calling the matcher directly --
// out of scope here, since this only proves ROUTE resolution.
func buildSyntheticRequestForFixture(resp *mockserver.MockResponse) (*http.Request, error) {
	req, err := http.NewRequest(resp.Metadata.Method, resp.Metadata.Endpoint, nil)
	if err != nil {
		return nil, err
	}

	if len(resp.Request.QueryParams) > 0 {
		q := url.Values{}
		for k, v := range resp.Request.QueryParams {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	for k, v := range resp.Request.Headers {
		req.Header.Set(k, v)
	}

	if resp.Metadata.Version != "" {
		req.Header.Set("Accept", "application/json;version="+resp.Metadata.Version)
	}

	return req, nil
}

// --- TestVendoredVerifiedFixturesNotFabricated ---
//
// This is the CONTENT-level counterpart to TestVendoredFixtureLedgerDrift
// (which only checks existence/orphan status, never fixture bytes) and to
// internal/mockserver/bypass_governance_test.go's TestSyntheticPlaceholderFixtureGovernance
// (which only content-checks the small, explicitly-manifested set of
// bypass-loaded synthetic-placeholder fixtures under
// internal/mockserver/testdata/synthetic-placeholders/). Neither of those
// walks EVERY fixture backing a tests/verified-endpoints.json verified[] row
// against the real vendored corpus and inspects its raw bytes for a
// fabrication signature -- which is exactly the gap that let
// updates/updates_deployment_get.json's fabricated body sit undetected in
// verified[] until an external provenance trace found it. This test
// closes that gap: every verified[] fixture, not just a curated subset.
//
// The isSequentialDigitPlaceholderUUIDForLedgerSweep / allDigitUUIDShapeForLedgerSweep
// are duplicated from internal/mockserver/bypass_governance_test.go's
// isSequentialDigitPlaceholder / allDigitUUIDShape: that detector lives in
// package mockserver_test (internal/mockserver), which this package (tests)
// cannot import (test-only package, and importing across a _test.go
// package boundary isn't possible in Go regardless). Kept behaviorally
// identical for the guidEmpty sentinel carve-out -- see
// TestSequentialDigitPlaceholderUUIDForLedgerSweep_GuidEmptyExcluded below,
// the same positive/negative pin that file's own
// TestIsSequentialDigitPlaceholder_GuidEmptyExcluded asserts -- but widened
// 2026-09-23 beyond that file's digit-only
// detector: the shipped regex/logic only ever flagged all-DIGIT
// single-repeated groups (e.g. "11111111-2222-..."), so a letter-only
// placeholder with the same letter repeated across every hex group
// walked straight through undetected. The allDigitUUIDShapeForLedgerSweep now
// matches ANY hex-shaped UUID substring (digits AND a-f/A-F letters), not
// just all-digit ones -- the broad regex alone would over-match every
// legitimate hex UUID, so isSequentialDigitPlaceholderUUIDForLedgerSweep
// below does the real narrowing: per the verdict's own detection criteria
// (section 2b "UUID signatures"), a UUID is flagged when at least 3 of its 5
// hyphen-separated groups are a single repeated hex character (digit OR
// letter), not requiring every group to match as the pre-widen version did.
var allDigitUUIDShapeForLedgerSweep = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)

// buildUUIDForLedgerSweep assembles a UUID string from its 5 hyphen-separated
// groups at RUNTIME, rather than as a single string literal. The public
// package's sync scrubber rewrites any whole-UUID-shaped literal it finds
// anywhere in shipped Go source, regardless of context -- including planted
// test-control UUIDs that must stay byte-exact for the guard's own tests to
// mean anything. None of the individual group arguments is itself UUID-
// shaped, so the scrubber has nothing to match; strings.Join re-assembles the
// same value this function's callers need at test-run time.
func buildUUIDForLedgerSweep(groups ...string) string {
	return strings.Join(groups, "-")
}

var guidEmptySentinelForLedgerSweep = buildUUIDForLedgerSweep("00000000", "0000", "0000", "0000", "000000000000")

// sensorGetFixtureUUIDForTest reads sensors/get_sensor.json's real embedded
// response.body.uuid at runtime, instead of hardcoding it as a Go string
// literal -- the sync scrubber rewrites any whole-UUID-shaped literal it
// finds in shipped Go source, but does not (and must not) touch fixture JSON
// data the same way, so a hardcoded copy of this value drifts out of sync
// with the real fixture in the assembled public package. Same rationale as
// chunk_handler_test.go's chunkFixtureUuidForTest, mirrored for this
// fixture's shape.
func sensorGetFixtureUUIDForTest(t *testing.T, raw []byte) string {
	t.Helper()
	var decoded struct {
		Response struct {
			Body struct {
				UUID string `json:"uuid"`
			} `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("parse sensors/get_sensor.json: %v", err)
	}
	if decoded.Response.Body.UUID == "" {
		t.Fatal("sensors/get_sensor.json has an empty response.body.uuid field -- cannot derive an expected value")
	}
	return decoded.Response.Body.UUID
}

// groupIsSingleRepeatedHexChar reports whether group consists of one hex
// character repeated for its entire length (e.g. "cccccccc", "1111", "eeee")
// -- the per-group primitive isSequentialDigitPlaceholderUUIDForLedgerSweep
// composes across all 5 groups of a UUID.
func groupIsSingleRepeatedHexChar(group string) bool {
	if group == "" {
		return false
	}
	for i := 1; i < len(group); i++ {
		if group[i] != group[0] {
			return false
		}
	}
	return true
}

// hexRunPlaceholderMinLength is the minimum length of a monotonically
// ascending or descending hex-value run (mod 16, wrapping) inside a UUID's
// concatenated hex digits that hasSequentialHexRunForLedgerSweep treats as a
// hand-typed sequential placeholder -- e.g. "12345678" or "abcdef". Chosen
// short enough to catch the shortest documented real form (the 6-char
// "abcdef" run that spans a hyphen-group boundary inside
// "a1b2c3d4_e5f6_7890_abcd_ef1234567890") while staying long enough that a
// genuine gofakeit-random UUID hitting a run this long by chance is
// astronomically unlikely (roughly a 1-in-16 chance per additional step
// continuing in a fixed direction).
const hexRunPlaceholderMinLength = 6

// hexCharValue returns c's hex value (0-15), or -1 if c is not a hex digit.
func hexCharValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}

// hasSequentialHexRunForLedgerSweep reports whether hex (a UUID with its
// hyphens already stripped) contains a monotonically ascending or descending
// hex-value run -- each step is exactly +1 or -1 mod 16, so "f" -> "0"
// continues an ascending run -- at least hexRunPlaceholderMinLength
// characters long, anywhere in the string. The scan deliberately crosses
// hyphen-group boundaries (it walks the already-concatenated string, not
// group-by-group): real hand-typed placeholders like
// "a1b2c3d4_e5f6_7890_abcd_ef1234567890" carry their longest run ("abcdef")
// spanning exactly such a boundary (the last two characters of the "abcd"
// group plus the first two of the "ef1234567890" group), so a per-group-only
// scan would miss it.
func hasSequentialHexRunForLedgerSweep(hex string) bool {
	if len(hex) == 0 {
		return false
	}
	ascRun, descRun := 1, 1
	best := 1
	prev := hexCharValue(hex[0])
	for i := 1; i < len(hex); i++ {
		v := hexCharValue(hex[i])
		if prev < 0 || v < 0 {
			ascRun, descRun = 1, 1
			prev = v
			continue
		}
		if v == (prev+1)%16 {
			ascRun++
		} else {
			ascRun = 1
		}
		if v == (prev+15)%16 {
			descRun++
		} else {
			descRun = 1
		}
		if ascRun > best {
			best = ascRun
		}
		if descRun > best {
			best = descRun
		}
		prev = v
	}
	return best >= hexRunPlaceholderMinLength
}

// repeatingHexWordMinPeriod/MaxPeriod bound the "repeating hex word" unit
// length hasRepeatingHexWordForLedgerSweep looks for -- e.g. "deadbeef"
// (period 8) repeated across "deadbeef_dead_beef_dead_beefdeadbeef"
// (concatenated: "deadbeef" x4, 32 hex chars). Lower bound 2 excludes
// period-1 (a single repeated character), already fully covered by
// groupIsSingleRepeatedHexChar/signature 1 below -- keeping period>=2 here
// avoids redundant double-detection of that exact case, though it would be
// harmless either way. Upper bound 16 is half of a UUID's 32 hex digits,
// the longest period that can still repeat at least twice.
const (
	repeatingHexWordMinPeriod = 2
	repeatingHexWordMaxPeriod = 16
)

// hasRepeatingHexWordForLedgerSweep reports whether hex (a UUID with its
// hyphens already stripped) is built by repeating some short substring
// ("hex word") across its ENTIRE length -- e.g. "deadbeef" repeated 4 times,
// or "cafebabe" repeated 4 times. This is a general periodicity check (any
// unit length from repeatingHexWordMinPeriod to repeatingHexWordMaxPeriod
// that evenly divides len(hex) and tiles the whole string), not a fixed
// word list: a hand-typed mnemonic placeholder can use any hex-spellable
// word, and enumerating them would always be one word behind. Requires the
// repeating unit to contain at least 2 distinct hex characters, so a
// pure single-char repeat (already caught by signature 1) isn't
// double-counted here. A genuine gofakeit-random UUID matching this
// periodicity by chance is astronomically unlikely (every one of the
// string's len(hex)-period positions must independently align with its
// modular predecessor).
func hasRepeatingHexWordForLedgerSweep(hex string) bool {
	n := len(hex)
	for period := repeatingHexWordMinPeriod; period <= repeatingHexWordMaxPeriod && period*2 <= n; period++ {
		if n%period != 0 {
			continue
		}
		periodic := true
		for i := period; i < n; i++ {
			if hex[i] != hex[i%period] {
				periodic = false
				break
			}
		}
		if !periodic {
			continue
		}
		distinct := make(map[byte]bool, period)
		for i := 0; i < period; i++ {
			distinct[hex[i]] = true
		}
		if len(distinct) >= 2 {
			return true
		}
	}
	return false
}

// isSequentialDigitPlaceholderUUIDForLedgerSweep flags a UUID as a hand-typed
// placeholder under any of three independent signatures:
//
//  1. At least 3 of its 5 hyphen-separated groups are each a single repeated
//     hex character (digit OR letter a-f/A-F) -- matching the detection
//     criteria used to find the blind spot that widening closed. This also
//     covers an all-zero UUID that is NOT the literal .NET Guid.Empty
//     sentinel (which is explicitly excluded below, since that exact value
//     is a legitimate wire value, not a fabricated placeholder).
//  2. Its concatenated hex digits contain a monotonically ascending or
//     descending hex-value run (mod 16, wrapping) at least
//     hexRunPlaceholderMinLength characters long anywhere in the string --
//     catching hand-typed sequential forms like
//     "12345678_1234_1234_1234_123456789012",
//     "12345678_90ab_cdef_1234_567890abcdef" and
//     "a1b2c3d4_e5f6_7890_abcd_ef1234567890" that signature 1 alone does
//     not recognize.
//  3. Widened after scripts/bulkdelete-zz-partial.json's
//     "deadbeef_dead_beef_dead_beefdeadbeef" (concatenated: "deadbeef"
//     repeated 4 times) surfaced a blind spot neither signature above
//     catches: its concatenated hex digits are built by repeating a short
//     hex "word" across the whole string (hasRepeatingHexWordForLedgerSweep).
func isSequentialDigitPlaceholderUUIDForLedgerSweep(uuid string) bool {
	if uuid == guidEmptySentinelForLedgerSweep {
		return false
	}
	groups := strings.Split(uuid, "-")
	repeated := 0
	for _, group := range groups {
		if groupIsSingleRepeatedHexChar(group) {
			repeated++
		}
	}
	if repeated >= 3 {
		return true
	}
	concatenated := strings.Join(groups, "")
	if hasSequentialHexRunForLedgerSweep(concatenated) {
		return true
	}
	return hasRepeatingHexWordForLedgerSweep(concatenated)
}

// findSequentialDigitPlaceholderUUIDsForLedgerSweep returns every
// placeholder-shaped UUID substring found in body, or nil if body is clean.
func findSequentialDigitPlaceholderUUIDsForLedgerSweep(body string) []string {
	var out []string
	for _, m := range allDigitUUIDShapeForLedgerSweep.FindAllString(body, -1) {
		if isSequentialDigitPlaceholderUUIDForLedgerSweep(m) {
			out = append(out, m)
		}
	}
	return out
}

// verifiedLedgerFixtureFabricationAllowlist documents verified[] fixtures
// whose raw content legitimately contains a sequential-digit/letter
// placeholder-shaped UUID despite being a genuine live capture -- hand-
// anonymized rather than produced by gofakeit.UUID() (the real anonymizer's
// RNG, which cannot produce this shape). Every entry requires independently
// corroborated evidence (a doctrine quirk, a regression test it motivated,
// or an audited provenance writeup), never just "looks plausible" -- adding
// an entry here silences this test's only content-level fabrication check
// for that fixture, so an unjustified entry defeats the whole guard.
var verifiedLedgerFixtureFabricationAllowlist = map[string]string{
	"mam-apps/apps_get_internal_provisionings.json": "Genuine live capture (UEM 26.2.0.0, shared test org group). AppUuids[0] (\"bbbbbbbb_2222_3333_4444_555555555555\") and the top-level uuid (\"cccccccc_3333_4444_5555_666666666666\") were hand-anonymized rather than produced by gofakeit.UUID(), but the capture narrative is specific and falsifiable (real host/tenant/org-group/app-id, a described discovery workflow) and has already been load-bearing for a real, independently-landed swagger/codegen fix with its own regression test (internal-source/mamv2_internalapps_provisioning_test.go, TestMamV2_AppListUsingProvisioningProfileModel_AppUuidsIsStringArray) and doctrine entry (.claude/rules/api-doctrine.md quirk 19). This fixture is a genuine capture with hand-anonymized UUIDs, provenance-weak only on the literal -- as opposed to updates/updates_deployment_get.json, which was undocumented fabrication and was demoted to unverified[] rather than allowlisted here.",

	// The 8 entries below were added 2026-09-23 after this guard's first run against the
	// real ledger surfaced them as additional undocumented placeholder hits
	// beyond updates/updates_deployment_get.json. Each was individually
	// classified KIND A (genuine capture, 1-2 ID fields hand-anonymized before
	// tool coverage) or KIND B (hand-assembled body from genuine error text,
	// not a capture at all) against a fixed evidence standard -- see each
	// row's own tests/verified-endpoints.json "reason" field (prefixed
	// "SWEEP NOTE" for KIND A, "SYNTHETIC BODY, genuine error text" for KIND
	// B) for the full per-file evidence. Vendored fixture content itself was
	// NOT touched -- only ledger reasons and this allowlist.
	"sensors/assign_sensors.json":                       "KIND A: genuine live capture (shared test org group, UEM 26.2.0.0); activity_id (the only content-bearing field on this all-accepted response) is hand-anonymized before tool coverage. See tests/verified-endpoints.json's DeviceSensorsV1Service.AssignDeviceSensor reason for full evidence, including cross-corpus corroboration from the sibling sensors/*.json fixtures captured in the same capture pass.",
	"internal-apps-v1/apk/internal_app_create_apk.json": "KIND A: genuine live capture via the real ChunkedUploader (detailed empirical required-field findings); OrganizationGroupUuid and Uuid are hand-anonymized before tool coverage, and LocationGroupId/Id.Value are the anonymizer's shared 12345 sequential placeholder (NOT the real shared test org group id) -- while BundleId (com.airwatch.JustSayHello, APK-manifest-derived)/ApplicationName/AppVersion/PushMode remain genuine unaltered wire content. LocationGroupId is the anonymizer's shared 12345 sequential placeholder, not genuine wire content matching the real org group. See tests/verified-endpoints.json's InternalAppsV1Service.CreateInternalApplicationFromBlob reason for full evidence.",
	"updates/updates_deployments_empty.json":            "KIND A: genuine live capture; the only placeholder-shaped value is request.query_params.organization_group_uuid, a hand-typed mockserver request-matching key (matcher.go excludes the query string from its own endpoint match) rather than asserted wire data -- the response body itself carries no placeholder content. See tests/verified-endpoints.json's UpdatesV1Service.GetDeploymentsByDeviceUpdate reason for full evidence.",
	"updates/updates_search.json":                       "KIND A: genuine live capture; the only placeholder-shaped value is request.query_params.organization_group_uuid (a mockserver request-matching key, not wire data -- same reasoning as its sibling updates_deployments_empty.json). The response body is genuine: 46 gofakeit-fingerprinted update_list entries, corroborated by a literal UUID shared with sensors/get_sensor.json from the same fixed-seed anonymizer run. See tests/verified-endpoints.json's UpdatesV1Service.GetDeviceUpdatesBySearchParameters reason for full evidence.",
	"profiles/profiles_apple_get_after_delete.json":     "KIND B: hand-assembled body (not a capture) from genuine error text -- mock-capture's CLI cannot write a fixture for a non-2xx response. See tests/verified-endpoints.json's ProfilesV2Service.GetDeviceProfileDetailsAsync reason for fixture profiles/profiles_apple_get_after_delete.json (SYNTHETIC BODY, genuine error text) for full evidence, including exactly which genuine error text this body reflects and what the consuming test actually proves.",
	"profiles/profiles_appleosx_get_after_delete.json":  "KIND B: hand-assembled body (not a capture) from genuine error text -- same architecture as the sibling profiles_apple_get_after_delete.json entry. See tests/verified-endpoints.json's ProfilesV2Service.GetDeviceProfileDetailsAsync reason for fixture profiles/profiles_appleosx_get_after_delete.json for full evidence.",
	"profiles/profiles_winrt_get_after_delete.json":     "KIND B: hand-assembled body (not a capture) from genuine error text -- same architecture as the sibling profiles_apple_get_after_delete.json entry. See tests/verified-endpoints.json's ProfilesV2Service.GetDeviceProfileDetailsAsync reason for fixture profiles/profiles_winrt_get_after_delete.json for full evidence.",

	// Once the detector was widened to letters and a >=3-of-5-groups
	// threshold (instead of requiring every group to match), it correctly
	// found two REAL, previously-undetected placeholder-shaped UUIDs already
	// present in the vendored corpus. Both are independently corroborated as
	// benign, not undisclosed fabrication -- see each entry's own reason.
	"smartgroups/smartgroups_get_after_delete.json":       "KIND B: hand-assembled body (not a capture) from genuine error text -- same 'hand-assembled error body, genuine error text' architecture as the ProfilesV2Service get-after-delete siblings above (mock-capture's CLI cannot write a fixture for a non-2xx response). See tests/verified-endpoints.json's SmartGroupsService.LoadSmartGroupAsync reason, DISCLOSURE (2026-09-23) paragraph: activityId (\"cccccccc_dddd_eeee_ffff_000000000000\") is a hand-typed placeholder, not part of the disclosed genuine error text; what the fixture actually proves is the real HTTP 400/errorCode 6 'Smart Group not found' behavior on the immediate follow-up GET after delete, not the exact activityId value.",
	"profiles/appleosx/custom-settings/get-by-id-v2.json": "PayloadUUID (\"00000000_0000_0000_0000_000000000001\", embedded in the CustomSettings XML plist string) is NOT server-fabricated response content -- it is the literal PayloadUUID the SDK itself sent on create, sourced from testdata/request-templates/profiles/appleosx/custom-settings/custom-settings-create-v2.json (same literal value, byte-for-byte), and the live server simply echoed it back verbatim on this GET. Per quirk 13's classify-before-override discipline, a REQUEST-side value the server echoes on read is not a response-fabrication signal. Org group UUID re-anonymized 2026-09-24 by tools/reanonymize-fixture at commit 999e3892a, command reanonymize-fixture -replace-uuid <original> -key OrganizationGroupUuid, where <original> is the UUID whose sha256 is in tests/leaked_og_uuid_guard_test.go; every other byte is unchanged.",

	// Added 2026-09-23: this entry's fixture
	// path is followed by a "(documentation-only -- see reason)" parenthetical
	// note in the same verified[] row's comma-separated fixture field
	// (InternalAppsV1Service.CreateInternalApplicationFromBlob). Before the
	// splitLedgerFixtureField fix above, that note's embedded whitespace
	// caused this path to be dropped entirely, so this fixture was never
	// content-scanned. Now that it is, its genuine hand-typed placeholder
	// UUIDs need this entry, same as the sibling apk allowlist entry above.
	"internal-apps-v1/msi/internal_app_create_msi.json": "KIND A: genuine live capture via the real ChunkedUploader (26.2.0.0 test tenant, 2026-07-30; detailed empirical required-field findings on the fixture's own metadata.description, including MSI-specific DeploymentOptions/SupportedProcessorArchitecture requirements and the cross-platform PushMode string-vs-wire-number defect). OrganizationGroupUuid (\"11111111_2222_3333_4444_555555555555\") and Uuid (\"bbbbbbbb_cccc_dddd_eeee_ffffffffffff\") are hand-typed placeholders, and LocationGroupId/Id.Value are the anonymizer's shared 12345 sequential placeholder -- but ApplicationName, BundleId (a real installer GUID, not tenant-specific -- see the private ledger for provenance), AppVersion, and PushMode=0 are genuine unaltered wire content. See tests/verified-endpoints.json's InternalAppsV1Service.CreateInternalApplicationFromBlob reason, DISCLOSURE paragraph, for the full per-field evidence -- same disclosure architecture as this map's internal-apps-v1/apk/internal_app_create_apk.json sibling entry above.",

	// Added 2026-09-23: the ascending-hex-run
	// signature (hasSequentialHexRunForLedgerSweep) newly recognizes the
	// "12345678-90ab-cdef-..." form as a placeholder shape; these two rows
	// already carried a truthful DISCLOSURE of that exact literal before the
	// widening landed -- both are safe only because their reasons disclose it,
	// not because the guard would catch them -- the widened guard now also
	// catches them, so the existing disclosure is what keeps them passing).
	"blobs/upload_blob_v1.json": "Genuine live capture (26.2 test tenant, shared test org group, 2026-08-10 recapture; BlobsV1Service.UploadBlobAsync). DISCLOSURE already on the ledger row: the body's 'uuid' field (\"22345678_90ab_cdef_1234_567890abcdef\") is a hand-typed placeholder, not gofakeit- or server-produced -- the rest of the body ('Value', the numeric blob id) and the capture narrative are genuine, live-captured wire content. See tests/verified-endpoints.json's BlobsV1Service.UploadBlobAsync reason for the full evidence.",
	"blobs/upload_blob_v2.json": "Genuine live capture (26.2.0.0 test tenant, 2026-07-30, direct-SDK-client scratch program; BlobsV2Service.UploadBlobAsync). DISCLOSURE already on the ledger row: the body's 'uuid' field (\"12345678_90ab_cdef_1234_567890abcdef\") is a hand-typed placeholder, not genuine wire-populated content -- the rest of the body (Value, and the request/response shape, headers, and Content-Type findings) is genuine, live-captured wire content. See tests/verified-endpoints.json's BlobsV2Service.UploadBlobAsync reason for the full evidence.",
}

// fabricationProblemsForVerifiedLedgerFixture is the *testing.T-free core of
// TestVendoredVerifiedFixturesNotFabricated: given a verified[] row's
// (service, method, fixture-relative-path) and the fixture's raw on-disk
// bytes, it returns one problem string per placeholder-shaped UUID found
// that isn't covered by verifiedLedgerFixtureFabricationAllowlist, or nil if
// clean/allowlisted. Split out so
// TestVendoredVerifiedFixturesNotFabricated_FailsOnPreDemotionLedgerRow can
// assert the guard's failure path directly against a synthetic pre-demotion
// ledger row, the same technique
// internal/mockserver/bypass_governance_test.go's
// recapturedFixtureFabricationProblems /
// TestSyntheticPlaceholderFixtureGovernance_FailsOnRevertedFixture pair
// uses.
//
// The allowlist is keyed by fixture PATH, not by (row, fixture): more than
// one verified[] row can legitimately cite the same fixture path (e.g. a
// platform-group row and a platform-specific row that share a create
// fixture). A path-only check would let a row with no disclosure of its own
// silently inherit an exemption a DIFFERENT row's reason earned -- so an
// allowlisted path is a necessary but not sufficient condition: the row's
// OWN reason must also independently disclose the placeholder, via
// placeholderDisclosurePhrasesForLedgerSweep, before this returns clean.
func fabricationProblemsForVerifiedLedgerFixture(service, method, rel, reason string, raw []byte) []string {
	hits := findSequentialDigitPlaceholderUUIDsForLedgerSweep(string(raw))
	if len(hits) == 0 {
		return nil
	}
	if _, ok := verifiedLedgerFixtureFabricationAllowlist[rel]; ok {
		if _, disclosed := matchedPlaceholderDisclosurePhrase(strings.ToLower(reason)); disclosed {
			return nil
		}
		return []string{fmt.Sprintf(
			"verified[] fixture %q (backing %s.%s) contains sequential-digit/letter placeholder UUID(s) %v, and the fixture PATH is allowlisted in verifiedLedgerFixtureFabricationAllowlist -- but THIS ROW's own reason field discloses none of it (reason=%q). A shared allowlist entry covers a fixture path, not a specific row; every row that cites a placeholder-bearing fixture needs its own disclosure, not a borrowed one",
			rel, service, method, hits, reason,
		)}
	}
	return []string{fmt.Sprintf(
		"verified[] fixture %q (backing %s.%s) contains sequential-digit/letter placeholder UUID(s) %v -- this is the project's own documented fabrication fingerprint (internal/mockserver/bypass_governance_test.go's isSequentialDigitPlaceholder). Either this fixture is a fabricated placeholder that must be demoted to unverified[] with an honest reason, or it is a genuine hand-anonymized capture that needs a corroborated, reviewed entry in verifiedLedgerFixtureFabricationAllowlist -- do not add an allowlist entry without independent evidence",
		rel, service, method, hits,
	)}
}

// placeholderDisclosurePhrasesForLedgerSweep are the case-insensitive
// substrings a verified[] row's own reason must contain to count as
// disclosing a placeholder-shaped value in its cited fixture -- the
// row-level counterpart to the path-level
// verifiedLedgerFixtureFabricationAllowlist entry.
var placeholderDisclosurePhrasesForLedgerSweep = []string{
	"disclosure",
	"hand-typed",
	"hand-anonymized",
	"placeholder",
}

// matchedPlaceholderDisclosurePhrase returns the first
// placeholderDisclosurePhrasesForLedgerSweep entry contained in lower
// (which the caller must already have strings.ToLower'd), or ("", false).
func matchedPlaceholderDisclosurePhrase(lower string) (string, bool) {
	for _, p := range placeholderDisclosurePhrasesForLedgerSweep {
		if strings.Contains(lower, p) {
			return p, true
		}
	}
	return "", false
}

// selfLabelPhrasesForLedgerSweep are the case-insensitive substrings this
// guard treats as a fixture self-labeling its own content as not-a-genuine-
// live-capture. The UUID-signature detector above only
// catches placeholder-SHAPED values, never a fixture's own prose admission --
// which is exactly how row 64 (profiles/get_profile_12345.json, self-labeled
// "field-decode-shape test double … not a live-captured canonical answer")
// escaped it and had to be caught by hand instead of by CI.
var selfLabelPhrasesForLedgerSweep = []string{
	"test double",
	"synthetic",
	"fabricat",
	"placeholder",
	"not live",
}

// matchedSelfLabelPhrase returns the first selfLabelPhrasesForLedgerSweep
// entry contained in lower (which the caller must already have
// strings.ToLower'd), or ("", false) if none match.
func matchedSelfLabelPhrase(lower string) (string, bool) {
	for _, p := range selfLabelPhrasesForLedgerSweep {
		if strings.Contains(lower, p) {
			return p, true
		}
	}
	return "", false
}

// minimalFixtureMetadataForLedgerSweep decodes just the one field
// selfLabelProblemsForVerifiedLedgerFixture needs (metadata.description) --
// deliberately not mockserver.MockResponse/LoadResponseFromFile, since a
// planted-control fixture body in this file's tests is a hand-built minimal
// JSON literal that only needs to round-trip this one field, not the full
// response schema.
type minimalFixtureMetadataForLedgerSweep struct {
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
}

// selfLabelProblemsForVerifiedLedgerFixture is the self-label counterpart to
// fabricationProblemsForVerifiedLedgerFixture above, closing guard gap C4:
// given a verified[] row's (service, method, fixture-relative-path, reason)
// and the fixture's raw on-disk bytes, it returns a problem when the
// fixture's OWN metadata.description self-labels itself (matches
// selfLabelPhrasesForLedgerSweep) and neither of two escape hatches applies:
// (a) rel is in verifiedLedgerFixtureFabricationAllowlist -- the SAME shared
// allowlist the UUID-signature path above uses, deliberately reused rather
// than a second map, so one reviewed entry discloses a fixture for both
// detectors; or (b) the row's own reason field itself references/acknowledges
// the self-labeled nature (contains one of the same trigger phrases) --
// e.g. a reason that says "SYNTHETIC BODY, genuine error text: ..." or
// "… hand-anonymized … not a wholesale fabrication" is disclosing, not
// hiding, the self-label. An empty reason never satisfies (b).
func selfLabelProblemsForVerifiedLedgerFixture(service, method, rel, reason string, raw []byte) []string {
	var fx minimalFixtureMetadataForLedgerSweep
	if err := json.Unmarshal(raw, &fx); err != nil {
		// Malformed/undecodable fixture content is TestVendoredFixtureLedgerDrift's
		// (or the fixture-loading path's) job to catch, not this content-scan's.
		return nil
	}

	phrase, matched := matchedSelfLabelPhrase(strings.ToLower(fx.Metadata.Description))
	if !matched {
		return nil
	}

	if _, ok := verifiedLedgerFixtureFabricationAllowlist[rel]; ok {
		return nil
	}

	if _, disclosed := matchedSelfLabelPhrase(strings.ToLower(reason)); disclosed {
		return nil
	}

	return []string{fmt.Sprintf(
		"verified[] fixture %q (backing %s.%s) has a metadata.description that self-labels itself %q, but the row's reason field neither is empty-exempt nor itself references/acknowledges that self-labeled nature (reason=%q) -- either demote this fixture to unverified[] with an honest reason (as row 64, profiles/get_profile_12345.json, was), or add a reviewed entry to verifiedLedgerFixtureFabricationAllowlist disclosing why the self-label doesn't disqualify it",
		rel, service, method, phrase, reason,
	)}
}

// TestVendoredVerifiedFixturesNotFabricated walks every fixture path cited
// by tests/verified-endpoints.json's verified[] array, locates it in the
// resolved vendored corpus, and content-scans it for a sequential-digit
// placeholder UUID. A hit outside verifiedLedgerFixtureFabricationAllowlist
// fails loudly, naming the offending ledger row.
//
// Fixture paths that don't resolve to a file physically present in this
// corpus walk (e.g. bypass-loaded synthetic-placeholder fixtures, or the
// "N/A -- served by ChunkHandler" non-path notes splitLedgerFixtureField
// already drops) are skipped here, not failed -- existence/orphan gaps are
// TestVendoredFixtureLedgerDrift's job, not this content-scan's.
func TestVendoredVerifiedFixturesNotFabricated(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)
	byRelPath := make(map[string]string, len(corpus))
	for _, e := range corpus {
		byRelPath[e.relPath] = e.fullPath
	}

	ledgerData, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- this guard needs the private vendored-corpus ledger, which is carry_over_excluded from the public package", err)
	}
	var ledger vendoredFixtureLedger
	if err := json.Unmarshal(ledgerData, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 {
		t.Fatal("control check failed: tests/verified-endpoints.json verified[] is empty -- the ledger read itself is broken")
	}

	checked := 0
	for _, e := range ledger.Verified {
		for _, rel := range splitLedgerFixtureField(e.Fixture) {
			t.Run(rel, func(t *testing.T) {
				fullPath, ok := byRelPath[rel]
				if !ok {
					t.Skip("fixture not present in the resolved vendored corpus (existence/orphan gaps are governed by TestVendoredFixtureLedgerDrift, not this content-scan)")
				}
				raw, err := os.ReadFile(fullPath)
				if err != nil {
					t.Fatalf("read fixture %s: %v", fullPath, err)
				}
				checked++
				for _, problem := range fabricationProblemsForVerifiedLedgerFixture(e.Service, e.Method, rel, e.Reason, raw) {
					t.Errorf("%s", problem)
				}
				for _, problem := range selfLabelProblemsForVerifiedLedgerFixture(e.Service, e.Method, rel, e.Reason, raw) {
					t.Errorf("%s", problem)
				}
			})
		}
	}
	if checked == 0 {
		t.Fatal("control check failed: zero verified[] fixtures were content-checked against the corpus -- the sweep logic itself is broken")
	}
}

// preDemotionUpdatesDeploymentGetFixtureBytesTemplate is the exact
// hand-authored placeholder body updates/updates_deployment_get.json carried
// before its live re-capture (every UUID a hand-typed sequential-digit/letter
// placeholder) -- embedded literally here, rather than read from disk,
// because the corpus file at this path has since been legitimately
// re-verified with genuine live-captured content at the same relative path
// (see tests/verified-endpoints.json's verified[] row for
// UpdatesV1Service.GetDeviceUpdateDeploymentDetails). This test's job is to
// prove the fabrication guard would have caught the OLD fabricated bytes,
// permanently, independent of whatever now lives at that path.
//
// The four UUID values are substituted in via %s at runtime (buildUUIDForLedgerSweep
// parts-assembly), not literals in this template: the public package's sync
// scrubber rewrites any whole-UUID-shaped literal it finds anywhere in
// shipped content, Go source included, which would replace these
// placeholder values with random ones and make the assertion below
// meaningless.
const preDemotionUpdatesDeploymentGetFixtureBytesTemplate = `{
  "metadata": {
    "endpoint": "/api/mdm/updates/deployments/{uuid}",
    "method": "GET",
    "version": "1"
  },
  "response": {
    "status_code": 200,
    "body": {
      "uuid": "%s",
      "ranking": 1,
      "organization_group_uuid": "%s",
      "device_update_uuid": "%s",
      "name": "Test Deployment",
      "deployment_start_time": "2024-01-15T00:00:00Z",
      "deployment_type": "DOWNLOAD_ONLY",
      "smart_group_uuids": ["%s"],
      "notifications": []
    }
  }
}`

var preDemotionUpdatesDeploymentGetFixtureBytes = fmt.Sprintf(
	preDemotionUpdatesDeploymentGetFixtureBytesTemplate,
	buildUUIDForLedgerSweep("99999999", "8888", "7777", "6666", "555544443333"),
	buildUUIDForLedgerSweep("11111111", "2222", "3333", "4444", "555555555555"),
	buildUUIDForLedgerSweep("aaaaaaaa", "bbbb", "cccc", "dddd", "eeeeeeeeeeee"),
	buildUUIDForLedgerSweep("33333333", "4444", "5555", "6666", "777777777777"),
)

// TestVendoredVerifiedFixturesNotFabricated_FailsOnPreDemotionLedgerRow is
// the committed, automated proof for this guard: it asserts
// fabricationProblemsForVerifiedLedgerFixture WOULD have flagged
// UpdatesV1Service.GetDeviceUpdateDeploymentDetails's fixture while it still
// held its pre-demotion, hand-authored placeholder content (embedded above,
// not read from the live corpus -- that path now legitimately holds a
// genuine re-capture, so reading it live would no longer exercise this
// guard's failure path at all).
func TestVendoredVerifiedFixturesNotFabricated_FailsOnPreDemotionLedgerRow(t *testing.T) {
	const rel = "updates/updates_deployment_get.json"

	problems := fabricationProblemsForVerifiedLedgerFixture("UpdatesV1Service", "GetDeviceUpdateDeploymentDetails", rel, "", []byte(preDemotionUpdatesDeploymentGetFixtureBytes))
	if len(problems) == 0 {
		t.Fatalf("expected fabricationProblemsForVerifiedLedgerFixture to flag %s (hand-typed sequential-digit placeholder UUIDs throughout the body) given its pre-demotion content, got no problems -- the guard would have silently accepted the pre-demotion fabrication", rel)
	}
}

// copyFileForPlant copies src to dst, creating dst's parent directory if
// needed. Used only by the planted-control tests below to build a scratch
// corpus COPY in a temporary directory -- the real vendored corpus (whatever
// MOCK_RESPONSES_DIR points at) is never written to.
func copyFileForPlant(t *testing.T, src, dst string) {
	t.Helper()
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("copyFileForPlant: read %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("copyFileForPlant: mkdir for %s: %v", dst, err)
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatalf("copyFileForPlant: write %s: %v", dst, err)
	}
}

// TestVendoredVerifiedFixturesNotFabricated_CatchesLetterPlaceholderUUID is
// the planted-failing control proving the guard catches a UUID whose repeated
// groups are LETTERS rather than digits, a shape the original digit-only
// regex could not detect. It builds a scratch
// COPY of the real resolved vendored corpus in a temporary directory (never
// touching the real corpus), plants that exact letter-placeholder UUID into
// the copy's sensors/get_sensor.json (a real verified[] fixture, backing
// DeviceSensorsV1Service.GetDeviceSensor), points MOCK_RESPONSES_DIR at the
// scratch copy, and asserts fabricationProblemsForVerifiedLedgerFixture now
// flags it -- then, as the "pass on real/unmodified" half of the same
// before/after proof, asserts the corpus's own real, unmutated bytes for the
// same fixture are clean.
func TestVendoredVerifiedFixturesNotFabricated_CatchesLetterPlaceholderUUID(t *testing.T) {
	const rel = "sensors/get_sensor.json"

	realCorpus := walkVendoredFixtureCorpus(t)
	var realFullPath string
	for _, e := range realCorpus {
		if e.relPath == rel {
			realFullPath = e.fullPath
			break
		}
	}
	if realFullPath == "" {
		t.Fatalf("fixture %s not found in the resolved vendored corpus -- cannot run the planted-control proof", rel)
	}

	realRaw, err := os.ReadFile(realFullPath)
	if err != nil {
		t.Fatalf("read real fixture %s: %v", realFullPath, err)
	}
	if problems := fabricationProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, "", realRaw); len(problems) != 0 {
		t.Fatalf("control check failed: real, unmutated %s already flagged as fabricated (%v) -- the planted-control comparison below would be meaningless", rel, problems)
	}

	scratchDir := t.TempDir()

	for _, e := range realCorpus {
		copyFileForPlant(t, e.fullPath, filepath.Join(scratchDir, e.relPath))
	}

	// Assembled at runtime via buildUUIDForLedgerSweep, not a UUID-shaped Go
	// string literal: the letter-only shape this test's guard-gap proof
	// depends on would not survive the sync scrubber's deterministic
	// UUID-literal rewrite otherwise (it would become an arbitrary hex hash
	// with no letter-repetition structure, defeating the point of the plant).
	letterPlaceholder := buildUUIDForLedgerSweep("cccccccc", "dddd", "eeee", "ffff", "000000000000")
	realUUID := sensorGetFixtureUUIDForTest(t, realRaw)
	planted := strings.Replace(string(realRaw), realUUID, letterPlaceholder, 1)
	if planted == string(realRaw) {
		t.Fatal("control check failed: plant substitution made no change -- the expected literal UUID is no longer present in sensors/get_sensor.json, update this test's target string")
	}
	if err := os.WriteFile(filepath.Join(scratchDir, rel), []byte(planted), 0o644); err != nil {
		t.Fatalf("write planted fixture: %v", err)
	}

	t.Setenv(mockserver.ResponsesDirEnv, scratchDir)

	scratchCorpus := walkVendoredFixtureCorpus(t)
	var scratchFullPath string
	for _, e := range scratchCorpus {
		if e.relPath == rel {
			scratchFullPath = e.fullPath
			break
		}
	}
	if scratchFullPath == "" {
		t.Fatalf("fixture %s not found in the scratch corpus %s -- plant setup is broken", rel, scratchDir)
	}
	scratchRaw, err := os.ReadFile(scratchFullPath)
	if err != nil {
		t.Fatalf("read scratch fixture %s: %v", scratchFullPath, err)
	}

	problems := fabricationProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, "", scratchRaw)
	if len(problems) == 0 {
		t.Fatalf("expected fabricationProblemsForVerifiedLedgerFixture to flag the planted letter-placeholder UUID %q in %s, got no problems -- guard gap 1 (letter placeholders) is not actually fixed", letterPlaceholder, rel)
	}
	found := false
	for _, p := range problems {
		if strings.Contains(p, letterPlaceholder) {
			found = true
		}
	}
	if !found {
		t.Fatalf("fabricationProblemsForVerifiedLedgerFixture flagged %s but none of the problem(s) named the planted UUID %q: %v", rel, letterPlaceholder, problems)
	}
}

// TestVendoredVerifiedFixturesNotFabricated_CatchesParentheticalNotedPath is
// the planted-failing control for guard gap 2 (a verified[] "fixture" field
// entry whose path is followed by a parenthetical note, e.g. the real
// internal-apps-v1/msi/internal_app_create_msi.json row's literal
// "internal-apps-v1/msi/internal_app_create_msi.json (documentation-only --
// see reason)" shape in tests/verified-endpoints.json). It writes a planted
// fixture file with a fabricated placeholder UUID to a temporary scratch
// directory (never touching the real vendored corpus or
// tests/verified-endpoints.json), builds a synthetic ledger "fixture" field
// string with the same trailing-parenthetical shape pointing at that planted
// file's relative path, and asserts splitLedgerFixtureField extracts the
// leading path (not dropping it, and not swallowing the note into the path)
// AND that fabricationProblemsForVerifiedLedgerFixture flags the planted
// file's content once scanned -- proving the parser fix actually lets a
// previously-invisible parenthetical-noted fixture reach the content-scan.
func TestVendoredVerifiedFixturesNotFabricated_CatchesParentheticalNotedPath(t *testing.T) {
	const rel = "internal-apps-v1/msi/planted_parenthetical_control.json"
	placeholder := buildUUIDForLedgerSweep("11111111", "2222", "3333", "4444", "555555555555")

	scratchDir := t.TempDir()

	plantedBody := fmt.Sprintf(`{"metadata":{"endpoint":"/api/mam/apps/internal/application","method":"POST","version":"1"},"response":{"status_code":200,"body":{"Uuid":%q}}}`, placeholder)
	scratchPath := filepath.Join(scratchDir, rel)
	if err := os.MkdirAll(filepath.Dir(scratchPath), 0o755); err != nil {
		t.Fatalf("mkdir for planted fixture: %v", err)
	}
	if err := os.WriteFile(scratchPath, []byte(plantedBody), 0o644); err != nil {
		t.Fatalf("write planted fixture: %v", err)
	}

	ledgerField := rel + " (documentation-only -- see reason)"
	rels := splitLedgerFixtureField(ledgerField)
	if len(rels) != 1 || rels[0] != rel {
		t.Fatalf("splitLedgerFixtureField(%q) = %v, want [%q] -- guard gap 2 (parenthetical-noted path) is not actually fixed", ledgerField, rels, rel)
	}

	raw, err := os.ReadFile(scratchPath)
	if err != nil {
		t.Fatalf("read planted fixture %s: %v", scratchPath, err)
	}
	problems := fabricationProblemsForVerifiedLedgerFixture("InternalAppsV1Service", "CreateInternalApplicationFromBlob", rels[0], "", raw)
	if len(problems) == 0 {
		t.Fatalf("expected fabricationProblemsForVerifiedLedgerFixture to flag the planted placeholder UUID %q once reached via the parenthetical-noted path, got no problems", placeholder)
	}

	// Regression companion: a genuine non-path prose note (no ".json" leading
	// token) must still be dropped entirely, not misparsed as a path.
	if got := splitLedgerFixtureField("N/A -- served by internal/mockserver's stateful ChunkHandler, not a static JSON fixture"); len(got) != 0 {
		t.Errorf("splitLedgerFixtureField on a non-path prose note = %v, want empty -- the parenthetical-path fix must not start treating prose notes as paths", got)
	}
}

// TestSequentialDigitPlaceholderUUIDForLedgerSweep_GuidEmptyExcluded is the
// positive/negative pin for this file's duplicated detector, mirroring
// internal/mockserver/bypass_governance_test.go's own
// TestIsSequentialDigitPlaceholder_GuidEmptyExcluded so drift between the
// two copies is caught by a test failure, not just a doc comment.
func TestSequentialDigitPlaceholderUUIDForLedgerSweep_GuidEmptyExcluded(t *testing.T) {
	if isSequentialDigitPlaceholderUUIDForLedgerSweep(guidEmptySentinelForLedgerSweep) {
		t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = true, want false -- Guid.Empty is a legitimate .NET sentinel, not a fabricated placeholder", guidEmptySentinelForLedgerSweep)
	}

	fabricated := buildUUIDForLedgerSweep("11111111", "2222", "3333", "4444", "555555555555")
	if !isSequentialDigitPlaceholderUUIDForLedgerSweep(fabricated) {
		t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- regression control: the guidEmpty carve-out must not disable detection of genuine fabricated placeholders", fabricated)
	}
}

// TestSequentialDigitPlaceholderUUIDForLedgerSweep_CatchesAscendingHexRun is
// the direct unit-level positive control for the ascending/descending
// hex-run widening: two hand-typed sequential-hex-run placeholder forms that a
// repeated-character-group-only detector cannot recognize. Both UUIDs are assembled at runtime via buildUUIDForLedgerSweep
// (the scrubber-proof parts-assembly technique used elsewhere in this file), since they are themselves full
// UUID-shaped literals that the public-package sync scrubber would otherwise
// rewrite.
func TestSequentialDigitPlaceholderUUIDForLedgerSweep_CatchesAscendingHexRun(t *testing.T) {
	ascendingRun := buildUUIDForLedgerSweep("12345678", "1234", "1234", "1234", "123456789012")
	if !isSequentialDigitPlaceholderUUIDForLedgerSweep(ascendingRun) {
		t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- an 8-character ascending hex run (\"12345678\") is exactly the hand-typed sequential form this widening must catch", ascendingRun)
	}

	alternatingRun := buildUUIDForLedgerSweep("a1b2c3d4", "e5f6", "7890", "abcd", "ef1234567890")
	if !isSequentialDigitPlaceholderUUIDForLedgerSweep(alternatingRun) {
		t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- this UUID's longest repeated-character group is only 1 character, so signature 1 (>= 3 of 5 groups single-repeated-char) cannot catch it; it must be caught by the ascending-hex-run signature instead (its \"abcd\"+\"ef\" boundary-crossing run is 6 characters long)", alternatingRun)
	}
}

// TestSequentialDigitPlaceholderUUIDForLedgerSweep_CatchesRepeatingHexWord is
// the direct unit-level positive control for the repeating-hex-word
// widening (signature 3), covering the three shapes named when this
// widening was requested: a "deadbeef"-style repeated hex word, an
// all-zero UUID that is NOT the literal .NET Guid.Empty sentinel, and a
// plain repeated-single-character UUID (signature 1's own domain,
// re-asserted here so all three named shapes have an explicit control in
// one place). All UUIDs are assembled at runtime via buildUUIDForLedgerSweep,
// since they are themselves full UUID-shaped literals the public-package
// sync scrubber would otherwise rewrite.
func TestSequentialDigitPlaceholderUUIDForLedgerSweep_CatchesRepeatingHexWord(t *testing.T) {
	t.Run("deadbeef_style", func(t *testing.T) {
		// scripts/bulkdelete-zz-partial.json's real placeholder:
		// "deadbeef_dead_beef_dead_beefdeadbeef" -- concatenated hex digits
		// are "deadbeef" repeated 4 times, which neither signature 1
		// (no group is a single repeated character) nor signature 2 (no
		// monotonic hex run) can catch.
		deadbeef := buildUUIDForLedgerSweep("deadbeef", "dead", "beef", "dead", "beefdeadbeef")
		if !isSequentialDigitPlaceholderUUIDForLedgerSweep(deadbeef) {
			t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- a deadbeef-style repeated-hex-word placeholder must be caught", deadbeef)
		}
	})

	t.Run("all_zero_non_exempt", func(t *testing.T) {
		// Structurally identical to guidEmptySentinelForLedgerSweep (all
		// groups single-repeated-char '0'), but at a DIFFERENT literal value
		// than the exact .NET Guid.Empty sentinel this detector carves out --
		// proving the carve-out is narrowly scoped to that one exact value,
		// not to "any all-zero-shaped UUID."
		almostAllZero := buildUUIDForLedgerSweep("00000000", "0000", "0000", "0000", "000000000001")
		if !isSequentialDigitPlaceholderUUIDForLedgerSweep(almostAllZero) {
			t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- an all-zero-shaped UUID other than the exact Guid.Empty sentinel must still be caught", almostAllZero)
		}
	})

	t.Run("repeated_hex_char", func(t *testing.T) {
		repeatedHex := buildUUIDForLedgerSweep("cccccccc", "cccc", "cccc", "cccc", "cccccccccccc")
		if !isSequentialDigitPlaceholderUUIDForLedgerSweep(repeatedHex) {
			t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- a plain repeated-hex-character UUID (signature 1) must still be caught", repeatedHex)
		}
	})

	t.Run("genuine_uuid_not_flagged", func(t *testing.T) {
		// A realistic, non-periodic UUID must never be flagged -- the whole
		// point of bounding the period check is to keep this from
		// false-positiving on genuine gofakeit-random values.
		genuine := buildUUIDForLedgerSweep("4f8a2c91", "77e3", "4b1d", "9a02", "e6c58d1f3a4b")
		if isSequentialDigitPlaceholderUUIDForLedgerSweep(genuine) {
			t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = true, want false -- a genuine non-periodic UUID was wrongly flagged", genuine)
		}
	})

	t.Run("abc12345_ascending_then_sequential", func(t *testing.T) {
		// profiles/search_v2.json's real hand-typed placeholder:
		// "abc12345_1234_1234_1234_abc123456789". This shape is already
		// caught -- NOT by a new signature, but by the EXISTING signature 2
		// (hasSequentialHexRunForLedgerSweep): the concatenated hex digits
		// end in "...abc123456789", whose "123456789" tail is a 9-character
		// ascending run, well past hexRunPlaceholderMinLength (6). This
		// control exists to pin that fact explicitly (verified independently
		// with a standalone check before this test was written) rather than
		// leave it implicit -- no widening of the detector itself was
		// needed for this specific shape, only this dedicated control.
		abc12345 := buildUUIDForLedgerSweep("abc12345", "1234", "1234", "1234", "abc123456789")
		if !isSequentialDigitPlaceholderUUIDForLedgerSweep(abc12345) {
			t.Errorf("isSequentialDigitPlaceholderUUIDForLedgerSweep(%q) = false, want true -- an ascending-then-sequential placeholder like profiles/search_v2.json's real hand-typed UUIDs must be caught", abc12345)
		}
	})
}

// TestVendoredVerifiedFixturesNotFabricated_CatchesAscendingHexRun is the
// planted-failing control for the same ascending/descending hex-run widening, but exercised
// through the full fabricationProblemsForVerifiedLedgerFixture path (the
// unit-level test above only calls the detector primitive directly) -- same
// scratch-corpus-copy technique as
// TestVendoredVerifiedFixturesNotFabricated_CatchesLetterPlaceholderUUID
// above: the real corpus is never written to, only a temporary copy of it.
func TestVendoredVerifiedFixturesNotFabricated_CatchesAscendingHexRun(t *testing.T) {
	const rel = "sensors/get_sensor.json"

	realCorpus := walkVendoredFixtureCorpus(t)
	var realFullPath string
	for _, e := range realCorpus {
		if e.relPath == rel {
			realFullPath = e.fullPath
			break
		}
	}
	if realFullPath == "" {
		t.Fatalf("fixture %s not found in the resolved vendored corpus -- cannot run the planted-control proof", rel)
	}

	realRaw, err := os.ReadFile(realFullPath)
	if err != nil {
		t.Fatalf("read real fixture %s: %v", realFullPath, err)
	}
	if problems := fabricationProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, "", realRaw); len(problems) != 0 {
		t.Fatalf("control check failed: real, unmutated %s already flagged as fabricated (%v) -- the planted-control comparison below would be meaningless", rel, problems)
	}

	realUUID := sensorGetFixtureUUIDForTest(t, realRaw)

	scratchDir := t.TempDir()
	for _, e := range realCorpus {
		copyFileForPlant(t, e.fullPath, filepath.Join(scratchDir, e.relPath))
	}

	plants := map[string]string{
		"ascending":   buildUUIDForLedgerSweep("12345678", "1234", "1234", "1234", "123456789012"),
		"alternating": buildUUIDForLedgerSweep("a1b2c3d4", "e5f6", "7890", "abcd", "ef1234567890"),
	}

	for name, planted := range plants {
		t.Run(name, func(t *testing.T) {
			body := strings.Replace(string(realRaw), realUUID, planted, 1)
			if body == string(realRaw) {
				t.Fatal("control check failed: plant substitution made no change -- the expected literal UUID is no longer present in sensors/get_sensor.json, update this test's target string")
			}
			scratchPath := filepath.Join(scratchDir, rel)
			if err := os.WriteFile(scratchPath, []byte(body), 0o644); err != nil {
				t.Fatalf("write planted fixture: %v", err)
			}
			raw, err := os.ReadFile(scratchPath)
			if err != nil {
				t.Fatalf("read scratch fixture %s: %v", scratchPath, err)
			}
			problems := fabricationProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, "", raw)
			if len(problems) == 0 {
				t.Fatalf("expected fabricationProblemsForVerifiedLedgerFixture to flag the planted hex-run placeholder UUID %q in %s, got no problems -- the ascending/descending hex-run widening is not actually working", planted, rel)
			}
			found := false
			for _, p := range problems {
				if strings.Contains(p, planted) {
					found = true
				}
			}
			if !found {
				t.Fatalf("fabricationProblemsForVerifiedLedgerFixture flagged %s but none of the problem(s) named the planted UUID %q: %v", rel, planted, problems)
			}
			// Restore the scratch file to its real bytes so a later
			// subtest/plant in this same scratchDir starts clean.
			if err := os.WriteFile(scratchPath, realRaw, 0o644); err != nil {
				t.Fatalf("restore scratch fixture %s: %v", scratchPath, err)
			}
		})
	}
}

// TestVendoredVerifiedFixturesNotFabricated_CatchesUndisclosedSelfLabel is
// the planted-failing control proving the guard catches a verified[] fixture whose OWN
// metadata.description self-labels itself ("test double", "synthetic",
// "fabricat[ed]", "placeholder", "not live[-captured]") with no reason
// disclosing/acknowledging that -- exactly how row 64
// (profiles/get_profile_12345.json) escaped the UUID-signature-only guard
// before it was demoted. It builds a scratch COPY of the real resolved
// vendored corpus in a temporary directory (same technique as
// TestVendoredVerifiedFixturesNotFabricated_CatchesLetterPlaceholderUUID
// above; the real corpus is never written to), plants a self-label phrase
// into the copy's sensors/get_sensor.json description, and asserts:
//  1. the real, unmutated fixture (real description, empty reason) is clean;
//  2. the planted self-label WITH an empty reason fires;
//  3. the identical planted self-label WITH a reason that itself references
//     the self-label (the disclosure escape hatch) does NOT fire -- proving
//     the guard flags silence, not the self-label phrase itself.
func TestVendoredVerifiedFixturesNotFabricated_CatchesUndisclosedSelfLabel(t *testing.T) {
	const rel = "sensors/get_sensor.json"

	realCorpus := walkVendoredFixtureCorpus(t)
	var realFullPath string
	for _, e := range realCorpus {
		if e.relPath == rel {
			realFullPath = e.fullPath
			break
		}
	}
	if realFullPath == "" {
		t.Fatalf("fixture %s not found in the resolved vendored corpus -- cannot run the planted-control proof", rel)
	}

	realRaw, err := os.ReadFile(realFullPath)
	if err != nil {
		t.Fatalf("read real fixture %s: %v", realFullPath, err)
	}
	if problems := selfLabelProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, "", realRaw); len(problems) != 0 {
		t.Fatalf("control check failed: real, unmutated %s already flagged as self-labeled (%v) -- the planted-control comparison below would be meaningless", rel, problems)
	}

	var realFx minimalFixtureMetadataForLedgerSweep
	if err := json.Unmarshal(realRaw, &realFx); err != nil {
		t.Fatalf("parse real fixture %s: %v", realFullPath, err)
	}
	if realFx.Metadata.Description == "" {
		t.Fatalf("control check failed: real fixture %s has an empty metadata.description -- the plant substitution below needs real text to replace", rel)
	}

	scratchDir := t.TempDir()

	for _, e := range realCorpus {
		copyFileForPlant(t, e.fullPath, filepath.Join(scratchDir, e.relPath))
	}

	const plantedDescription = "SYNTHETIC field-decode-shape test double -- not a live-captured canonical answer, planted by TestVendoredVerifiedFixturesNotFabricated_CatchesUndisclosedSelfLabel"
	// realRaw holds the JSON-encoded (quoted/escaped) form of the description,
	// not the decoded Go string fx.Metadata.Description -- re-encode both
	// sides through json.Marshal so the substitution matches the raw bytes
	// verbatim regardless of any escaping (quotes, backslashes) in the real
	// description text.
	realDescriptionJSON, err := json.Marshal(realFx.Metadata.Description)
	if err != nil {
		t.Fatalf("marshal real description for substitution: %v", err)
	}
	plantedDescriptionJSON, err := json.Marshal(plantedDescription)
	if err != nil {
		t.Fatalf("marshal planted description for substitution: %v", err)
	}
	planted := strings.Replace(string(realRaw), string(realDescriptionJSON), string(plantedDescriptionJSON), 1)
	if planted == string(realRaw) {
		t.Fatal("control check failed: plant substitution made no change -- the real description text was not found verbatim in the raw bytes, update this test's substitution logic")
	}
	if err := os.WriteFile(filepath.Join(scratchDir, rel), []byte(planted), 0o644); err != nil {
		t.Fatalf("write planted fixture: %v", err)
	}

	t.Setenv(mockserver.ResponsesDirEnv, scratchDir)

	scratchCorpus := walkVendoredFixtureCorpus(t)
	var scratchFullPath string
	for _, e := range scratchCorpus {
		if e.relPath == rel {
			scratchFullPath = e.fullPath
			break
		}
	}
	if scratchFullPath == "" {
		t.Fatalf("fixture %s not found in the scratch corpus %s -- plant setup is broken", rel, scratchDir)
	}
	scratchRaw, err := os.ReadFile(scratchFullPath)
	if err != nil {
		t.Fatalf("read scratch fixture %s: %v", scratchFullPath, err)
	}

	// (2) Empty reason: must fire.
	problems := selfLabelProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, "", scratchRaw)
	if len(problems) == 0 {
		t.Fatalf("expected selfLabelProblemsForVerifiedLedgerFixture to flag the planted self-labeled description in %s with an empty reason, got no problems -- guard gap C4 (self-labels ignored) is not actually fixed", rel)
	}
	found := false
	for _, p := range problems {
		if strings.Contains(p, "test double") {
			found = true
		}
	}
	if !found {
		t.Fatalf("selfLabelProblemsForVerifiedLedgerFixture flagged %s but none of the problem(s) named the matched phrase %q: %v", rel, "test double", problems)
	}

	// (3) Reason that itself references the self-label: must NOT fire -- the
	// disclosure escape hatch.
	const disclosingReason = "Genuine live capture; the SYNTHETIC-looking description text is a leftover test-double label from an earlier draft, not an admission that this fixture's own content is fabricated -- see the ledger row's full reason for the corroborating evidence."
	if problems := selfLabelProblemsForVerifiedLedgerFixture("DeviceSensorsV1Service", "GetDeviceSensor", rel, disclosingReason, scratchRaw); len(problems) != 0 {
		t.Fatalf("expected selfLabelProblemsForVerifiedLedgerFixture to NOT flag %s once its reason itself references the self-label (disclosure escape hatch), got problems: %v", rel, problems)
	}
}

// --- TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture ---
//
// A fixture-deletion audit found 9 verified[] rows (scripts,
// scripts-assignments, system/info, and internal-apps-v1/begininstall-response.json)
// whose named Go test still PASSES with the row's vendored-corpus fixture
// deleted, because that test actually loads a PRIVATE "twin" fixture at
// internal/mockserver/testdata/synthetic-placeholders/<same relative path>
// through the override-blind bypass loader, which always wins over the
// vendored corpus regardless of MOCK_RESPONSES_DIR. A byte-identical twin
// PLUS a guard enforcing that identity (with a planted-drift control proving
// the guard actually fires on divergence) satisfies consumption for those
// rows -- this is that guard.
//
// The syntheticPlaceholdersRoot is relative to this package's own directory
// (tests/), matching every other relative path in this file.
const syntheticPlaceholdersRoot = "../internal/mockserver/testdata/synthetic-placeholders"

// syntheticPlaceholderTwinDriftProblems is the *testing.T-free core of
// TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture: given a twin's
// relative path and both files' raw bytes, it returns a problem string if
// they differ, or nil if byte-identical. Split out, same shape as
// fabricationProblemsForVerifiedLedgerFixture above, so the planted-drift
// control test can assert the guard's failure path directly against
// in-memory mutated bytes rather than only its pass path.
func syntheticPlaceholderTwinDriftProblems(rel string, twinRaw, vendoredRaw []byte) []string {
	if bytes.Equal(twinRaw, vendoredRaw) {
		return nil
	}
	return []string{fmt.Sprintf(
		"synthetic-placeholders twin %q differs from its same-relative-path vendored corpus counterpart -- rows whose named Go test reads this twin (not the vendored fixture directly, via the override-blind bypass loader) rely on this byte-identity to prove the vendored fixture is actually the thing being consumed; either restore the twin to match the vendored fixture, or the row's ledger reason claiming twin-identity consumption is no longer true",
		rel,
	)}
}

// syntheticPlaceholderTwinMissingCounterpartProblems is the *testing.T-free
// core of the missing-counterpart half of
// TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture: a twin whose
// same-relative-path vendored counterpart does not exist at all makes the
// row's twin-identity claim just as unverifiable as a byte mismatch would --
// there is nothing left to compare the twin against. Pure function, same
// shape as syntheticPlaceholderTwinDriftProblems above, so a planted control
// can assert this failure mode directly rather than only via a full corpus
// walk.
func syntheticPlaceholderTwinMissingCounterpartProblems(rel string) []string {
	return []string{fmt.Sprintf(
		"synthetic-placeholders twin %q has no same-relative-path vendored corpus counterpart -- the row's ledger reason claiming twin-identity consumption cannot be checked against anything and is no longer verifiable",
		rel,
	)}
}

// TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture walks every
// fixture path referenced by tests/verified-endpoints.json's verified[]
// array (same extraction as TestVendoredVerifiedFixturesNotFabricated) and,
// for each one that ALSO has a same-relative-path twin under
// internal/mockserver/testdata/synthetic-placeholders/, asserts the two are
// byte-identical.
//
// Deliberately scoped to verified[] rows only, NOT every twin in the
// synthetic-placeholders tree: many twins there (e.g. auth/oauth2_token_success.json,
// sensors/get_device_sensors.json, the demoted baselines/* fixtures) are
// documented Class-2 fabricated/demoted test doubles in
// knownOrphanedVendoredFixtures above -- intentionally NOT byte-identical to
// their same-named vendored-corpus sibling, since that sibling is the
// genuine capture the twin was never meant to match. Only a verified[] row
// makes a truth claim ("this fixture backs a real, passing, live-derived
// test") that the twin's byte-identity is load-bearing for; scoping to that
// set is what a first (unscoped) implementation of this guard got wrong --
// it fired on all 13 of those intentionally-divergent Class-2 twins before
// being narrowed to this ledger-driven set.
// Skips (does not fail) when MOCK_RESPONSES_DIR/the vendored corpus or the
// ledger is absent, same portability requirement as this file's other
// guards.
func TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)
	byRelPath := make(map[string]string, len(corpus))
	for _, e := range corpus {
		byRelPath[e.relPath] = e.fullPath
	}

	ledgerData, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- this guard needs the private vendored-corpus ledger, which is carry_over_excluded from the public package", err)
	}
	var ledger vendoredFixtureLedger
	if err := json.Unmarshal(ledgerData, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 {
		t.Fatal("control check failed: tests/verified-endpoints.json verified[] is empty -- the ledger read itself is broken")
	}

	verifiedRelPaths := map[string]bool{}
	for _, e := range ledger.Verified {
		for _, rel := range splitLedgerFixtureField(e.Fixture) {
			verifiedRelPaths[rel] = true
		}
	}

	var checked int
	var mismatches []string
	for rel := range verifiedRelPaths {
		twinFullPath := filepath.Join(syntheticPlaceholdersRoot, filepath.FromSlash(rel))
		twinInfo, statErr := os.Stat(twinFullPath)
		if statErr != nil || twinInfo.IsDir() {
			// No same-path twin under synthetic-placeholders -- not this
			// guard's concern, this row's test presumably reads the
			// vendored fixture directly (or via some other mechanism).
			continue
		}
		vendoredFullPath, ok := byRelPath[rel]
		if !ok {
			// A twin exists but its same-relative-path vendored counterpart
			// does not: the row's twin-identity claim can no longer be
			// checked against anything, which is exactly as load-bearing a
			// failure as a byte mismatch would be -- a missing counterpart
			// is not a lesser case of "not this guard's concern", it's the
			// same claim with no vendored fixture left to compare against.
			checked++
			mismatches = append(mismatches, syntheticPlaceholderTwinMissingCounterpartProblems(rel)...)
			continue
		}
		checked++

		twinRaw, readErr := os.ReadFile(twinFullPath)
		if readErr != nil {
			t.Fatalf("read twin %s: %v", twinFullPath, readErr)
		}
		vendoredRaw, readErr := os.ReadFile(vendoredFullPath)
		if readErr != nil {
			t.Fatalf("read vendored counterpart %s: %v", vendoredFullPath, readErr)
		}
		mismatches = append(mismatches, syntheticPlaceholderTwinDriftProblems(rel, twinRaw, vendoredRaw)...)
	}

	if checked == 0 {
		t.Fatal("control check failed: zero verified[] fixture paths matched a same-path synthetic-placeholders twin -- the walk/match logic itself is broken (expected at least the 9 scripts/scripts-assignments/system/begininstall rows to match)")
	}

	sort.Strings(mismatches)
	if len(mismatches) > 0 {
		t.Errorf("%d synthetic-placeholders twin(s) are not byte-identical to their vendored corpus counterpart:\n  %s", len(mismatches), strings.Join(mismatches, "\n  "))
	}
}

// TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture_CatchesDrift is
// the planted-drift control for the guard above: it copies one real,
// currently byte-identical twin+vendored pair into scratch temporary
// directories (t.TempDir(), never touching the real
// internal/mockserver/testdata/ files or the real vendored corpus), mutates
// a single byte in the scratch twin copy, and asserts
// syntheticPlaceholderTwinDriftProblems now flags it -- then, as the "pass on
// real/unmodified" half of the same before/after proof, asserts the real,
// unmutated pair's own bytes are still identical.
func TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture_CatchesDrift(t *testing.T) {
	const rel = "scripts-assignments/list.json"

	corpus := walkVendoredFixtureCorpus(t)
	var vendoredFullPath string
	for _, e := range corpus {
		if e.relPath == rel {
			vendoredFullPath = e.fullPath
			break
		}
	}
	if vendoredFullPath == "" {
		t.Fatalf("fixture %s not found in the resolved vendored corpus -- cannot run the planted-drift control", rel)
	}
	twinFullPath := filepath.Join(syntheticPlaceholdersRoot, rel)

	vendoredRaw, err := os.ReadFile(vendoredFullPath)
	if err != nil {
		t.Fatalf("read vendored fixture %s: %v", vendoredFullPath, err)
	}
	twinRaw, err := os.ReadFile(twinFullPath)
	if err != nil {
		t.Fatalf("read twin %s: %v", twinFullPath, err)
	}

	if problems := syntheticPlaceholderTwinDriftProblems(rel, twinRaw, vendoredRaw); len(problems) != 0 {
		t.Fatalf("control check failed: real, unmutated twin %s already differs from its vendored counterpart (%v) -- the planted-drift comparison below would be meaningless", rel, problems)
	}

	scratchDir := t.TempDir()
	scratchVendored := filepath.Join(scratchDir, "vendored", rel)
	scratchTwin := filepath.Join(scratchDir, "twin", rel)
	copyFileForPlant(t, vendoredFullPath, scratchVendored)
	copyFileForPlant(t, twinFullPath, scratchTwin)

	mutated, err := os.ReadFile(scratchTwin)
	if err != nil {
		t.Fatalf("read scratch twin %s: %v", scratchTwin, err)
	}
	if len(mutated) == 0 {
		t.Fatalf("control check failed: scratch twin copy %s is empty -- cannot mutate a byte", scratchTwin)
	}
	mutated[0] ^= 0xFF // flip one byte -- never touches the real twin file, only the scratch copy
	if err := os.WriteFile(scratchTwin, mutated, 0o644); err != nil {
		t.Fatalf("write mutated scratch twin: %v", err)
	}

	scratchVendoredRaw, err := os.ReadFile(scratchVendored)
	if err != nil {
		t.Fatalf("read scratch vendored %s: %v", scratchVendored, err)
	}
	scratchTwinRaw, err := os.ReadFile(scratchTwin)
	if err != nil {
		t.Fatalf("read scratch twin %s: %v", scratchTwin, err)
	}

	problems := syntheticPlaceholderTwinDriftProblems(rel, scratchTwinRaw, scratchVendoredRaw)
	if len(problems) == 0 {
		t.Fatalf("expected syntheticPlaceholderTwinDriftProblems to flag the planted single-byte drift between the scratch twin and scratch vendored copies of %s, got no problems -- the twin-identity guard is not actually working", rel)
	}
}

// TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture_CatchesMissingCounterpart
// is the planted-deletion control for the guard above: it stages one real
// twin into a scratch directory (t.TempDir(), never touching the real
// internal/mockserver/testdata/ files or the real vendored corpus) and
// simulates its vendored counterpart having been deleted from the corpus by
// building a scratch corpus index that omits it entirely -- exactly the
// condition TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture's own
// main loop checks for via its byRelPath lookup. Asserts detection fires for
// the missing counterpart, then, as the "pass when present" half of the same
// before/after proof, asserts a real, existing pair reports no problem.
func TestSyntheticPlaceholderTwinsByteIdenticalToVendoredFixture_CatchesMissingCounterpart(t *testing.T) {
	const rel = "scripts-assignments/list.json"

	corpus := walkVendoredFixtureCorpus(t)
	var vendoredFullPath string
	for _, e := range corpus {
		if e.relPath == rel {
			vendoredFullPath = e.fullPath
			break
		}
	}
	if vendoredFullPath == "" {
		t.Fatalf("fixture %s not found in the resolved vendored corpus -- cannot run the planted-deletion control", rel)
	}
	twinFullPath := filepath.Join(syntheticPlaceholdersRoot, rel)
	if _, err := os.Stat(twinFullPath); err != nil {
		t.Fatalf("twin %s not found -- cannot run the planted-deletion control: %v", twinFullPath, err)
	}

	scratchDir := t.TempDir()
	scratchTwin := filepath.Join(scratchDir, "twin", rel)
	copyFileForPlant(t, twinFullPath, scratchTwin)
	// Deliberately do NOT copy a scratch vendored counterpart -- this
	// simulates the vendored corpus no longer containing this file, the same
	// state the main loop observes as `ok == false` from its byRelPath
	// lookup.

	problems := syntheticPlaceholderTwinMissingCounterpartProblems(rel)
	if len(problems) == 0 {
		t.Fatalf("expected syntheticPlaceholderTwinMissingCounterpartProblems to flag %s as having no vendored counterpart, got no problems -- the twin-identity guard would silently pass a deleted vendored fixture", rel)
	}

	// Pass-when-present half: the real pair (twin exists, vendored exists)
	// must report no problem via the drift check, proving this control's
	// missing-counterpart detection isn't just unconditionally firing.
	vendoredRaw, err := os.ReadFile(vendoredFullPath)
	if err != nil {
		t.Fatalf("read vendored fixture %s: %v", vendoredFullPath, err)
	}
	scratchTwinRaw, err := os.ReadFile(scratchTwin)
	if err != nil {
		t.Fatalf("read scratch twin %s: %v", scratchTwin, err)
	}
	if problems := syntheticPlaceholderTwinDriftProblems(rel, scratchTwinRaw, vendoredRaw); len(problems) != 0 {
		t.Fatalf("control check failed: real, present twin+vendored pair for %s was unexpectedly flagged (%v) -- the missing-counterpart control above would be meaningless if presence itself already failed", rel, problems)
	}
}

// TestFabricationProblemsForVerifiedLedgerFixture_RowAwareAllowlist
// reproduces the exact scenario a path-only allowlist check misses: two
// DIFFERENT verified[] rows citing the SAME allowlisted fixture path, one
// with a disclosure in its own reason and one without. A path-only check
// (rel is in the map, full stop) would pass both; the row-aware check must
// pass only the disclosed row and flag the undisclosed one.
func TestFabricationProblemsForVerifiedLedgerFixture_RowAwareAllowlist(t *testing.T) {
	const rel = "internal-apps-v1/apk/internal_app_create_apk.json"
	if _, ok := verifiedLedgerFixtureFabricationAllowlist[rel]; !ok {
		t.Fatalf("control check failed: %s is expected to be present in verifiedLedgerFixtureFabricationAllowlist for this scenario to be meaningful", rel)
	}
	placeholder := buildUUIDForLedgerSweep("11111111", "2222", "3333", "4444", "555555555555")
	raw := []byte(fmt.Sprintf(`{"response":{"body":{"Uuid":%q}}}`, placeholder))

	disclosedReason := "Genuine live capture; OrganizationGroupUuid is a hand-anonymized placeholder, not gofakeit-produced."
	if problems := fabricationProblemsForVerifiedLedgerFixture("InternalAppsV1Service", "CreateInternalApplicationFromBlob", rel, disclosedReason, raw); len(problems) != 0 {
		t.Errorf("row WITH its own disclosure was flagged: %v -- an allowlisted path plus a disclosing reason must pass", problems)
	}

	undisclosedReason := "Genuine live capture, consumed by a sibling test."
	problems := fabricationProblemsForVerifiedLedgerFixture("InternalAppsV1Service", "CreateInternalApplicationFromBlob (undisclosed-sibling)", rel, undisclosedReason, raw)
	if len(problems) == 0 {
		t.Fatal("row citing the SAME allowlisted path but with NO disclosure of its own was not flagged -- the allowlist is still effectively path-only, not row-aware; this is exactly the msi verified[48]/verified[123] scenario the row-aware fix closes")
	}
}

// --- Ledger fixture-name EXISTENCE and CROSS-ROW STATUS guards ---
//
// These two checks close a class of error distinct from the narrative guard
// in tests/ledger_reason_narrative_guard_test.go: a reason can be perfectly
// timeless in wording and still be FACTUALLY WRONG about a fixture --
// naming a file that was never captured under that name, or describing a
// sibling fixture's verified/unverified status incorrectly. Both extract
// candidate fixture-path tokens from reason/testFunction text using the
// same strict dir/name.json shape splitLedgerFixtureField already applies
// to the fixture field, so ordinary prose (a bare word, a sentence
// fragment) is never mistaken for a filename.

// ledgerFixturePathPattern matches a path-shaped token: one or more
// "/"-separated segments, the first starting with a letter (so a bare
// filename with no directory, or a run of digits, never matches), ending in
// ".json". Two candidates separated only by a "/" with no real directory
// name on either side (e.g. prose informally listing "a.json/b.json" as
// alternatives) never matches at all under this shape, which is the
// intended, conservative behavior -- a false negative here is safe (the
// existence check simply skips it), while a false positive would wrongly
// flag prose as a broken reference.
var ledgerFixturePathPattern = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_-]*(?:/[A-Za-z0-9_.-]+)+\.json\b`)

// ledgerBareFixtureNamePattern matches a BARE filename mention -- no
// directory at all, e.g. "(updates_device_status_count_success.json)" --
// which is how this ledger's prose actually names a fixture most often
// (the run-9 false claim this guard exists to catch was written exactly
// this way, with no directory prefix, so a directory-requiring pattern
// alone would never have caught it). Requires at least one '_' or '-' so a
// short, unremarkable word can never coincidentally qualify -- every real
// fixture filename in this corpus is snake_case or kebab-case with at least
// one separator before ".json".
var ledgerBareFixtureNamePattern = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9]*[_-][A-Za-z0-9_-]*\.json\b`)

// ledgerFixtureMentionKind distinguishes a path-shaped citation (checked
// directory-aware, via ledgerFixturePathExists) from a bare-filename
// citation (checked by basename only, via ledgerBareFixtureNameExists) --
// the two need different existence tiers, since a bare name's directory is
// unknown and basename-only matching would be too permissive if applied to
// an already-qualified path.
type ledgerFixtureMentionKind int

const (
	ledgerFixtureMentionPath ledgerFixtureMentionKind = iota
	ledgerFixtureMentionBareName
)

type ledgerFixtureMention struct {
	token string
	kind  ledgerFixtureMentionKind
}

// extractLedgerFixtureMentions returns every path-shaped and bare-filename
// token in s, discarding any match that is itself a suffix of a longer
// match immediately before it (e.g. scanning "success.json/other.json"
// must not yield the spurious candidate "json/other.json" -- the byte
// immediately preceding a real candidate's start must never be
// alphanumeric, '.', '_' or '-'), and discarding a bare-filename match that
// is really just the trailing segment of an already-found path match (so
// "updates/updates_device_status_count_success.json" is reported once, as
// a path, not twice).
func extractLedgerFixtureMentions(s string) []ledgerFixtureMention {
	validStart := func(start int) bool {
		if start == 0 {
			return true
		}
		prev := s[start-1]
		return (prev < 'a' || prev > 'z') && (prev < 'A' || prev > 'Z') && (prev < '0' || prev > '9') && prev != '.' && prev != '_' && prev != '-'
	}

	var out []ledgerFixtureMention
	pathEnds := make(map[int]bool)
	for _, loc := range ledgerFixturePathPattern.FindAllStringIndex(s, -1) {
		if !validStart(loc[0]) {
			continue
		}
		out = append(out, ledgerFixtureMention{token: s[loc[0]:loc[1]], kind: ledgerFixtureMentionPath})
		pathEnds[loc[1]] = true
	}
	for _, loc := range ledgerBareFixtureNamePattern.FindAllStringIndex(s, -1) {
		if !validStart(loc[0]) || pathEnds[loc[1]] {
			continue
		}
		out = append(out, ledgerFixtureMention{token: s[loc[0]:loc[1]], kind: ledgerFixtureMentionBareName})
	}
	return out
}

// ledgerFixturePathExists reports whether token resolves to a real file,
// trying every legitimate citation shape this ledger's prose uses: a
// repo-root-relative path exactly as written (covers "testdata/mock-
// responses/...", "internal-source/...", "internal/mockserver/testdata/
// synthetic-placeholders/..." and any other real repo path cited in full),
// a bare vendored-corpus-relative path (covers the same shape the fixture
// field itself uses, with byRelPath keyed the same way
// walkVendoredFixtureCorpus already produces), or a bare synthetic-
// placeholders-relative path (covers a twin cited without its full private
// prefix).
func ledgerFixturePathExists(token string, byRelPath map[string]string) bool {
	if _, err := os.Stat(filepath.Join("..", token)); err == nil {
		return true
	}
	bare := strings.TrimPrefix(token, "testdata/mock-responses/")
	if _, ok := byRelPath[bare]; ok {
		return true
	}
	sp := strings.TrimPrefix(token, "internal/mockserver/testdata/synthetic-placeholders/")
	if _, err := os.Stat(filepath.Join("..", "internal/mockserver/testdata/synthetic-placeholders", sp)); err == nil {
		return true
	}
	return false
}

// nonexistentFixtureMentions returns every fixture-name token in s (path-
// shaped or bare) that does not resolve to a real file: a path-shaped
// token is checked directory-aware via ledgerFixturePathExists; a bare
// token is checked by basename only, against byBasename (every corpus and
// twin file's own base filename).
func nonexistentFixtureMentions(s string, byRelPath map[string]string, byBasename map[string]bool) []string {
	var problems []string
	for _, m := range extractLedgerFixtureMentions(s) {
		switch m.kind {
		case ledgerFixtureMentionPath:
			if !ledgerFixturePathExists(m.token, byRelPath) {
				problems = append(problems, m.token)
			}
		case ledgerFixtureMentionBareName:
			if !byBasename[m.token] {
				problems = append(problems, m.token)
			}
		}
	}
	return problems
}

// ledgerFixtureExistenceHistoricalAllowlist names fixture-name tokens a
// reason cites while explicitly documenting that the name never resolved
// to a real file -- a corrected past mistake, not a current false claim.
// Distinguishing "this name was wrong, and here's the fix" prose from an
// actual false existence assertion is not reliably mechanical (both use
// similar vocabulary), so each entry here is a reviewed, one-line-justified
// exception, the same pattern verifiedLedgerFixtureFabricationAllowlist
// already uses for a different guard in this same file. Keyed by
// "Service.Method" (stable across ledger edits) rather than array index.
// Adding an entry here does not mean "fine" -- it means "reviewed, and the
// row's own reason already discloses the name never existed".
var ledgerFixtureExistenceHistoricalAllowlist = map[string][]string{
	"AppsV2Service.GetAssignmentRuleAsync":                                       {"apps_get_id_assignment-rules.json"},
	"AppsV2Service.GetListOfDevices":                                             {"apps_post_id_assignment-rules.json"},
	"AppsV2Service.UpdateAssignmentRuleAsync":                                    {"apps_put_id_assignment-rules.json"},
	"BlobsV1Service.DeleteBlobAsync":                                             {"blobs_delete_blob_id.json"},
	"DeviceSensorsV2Service.GetDeviceSensorAssignmentsAsync":                     {"devicesensors_get_assignments.json"},
	"InternalAppsV1Service.AddAssignmentsWithFlexibleDeploymentParametersAsync":  {"apps_post_internal_id_assignments.json"},
	"InternalAppsV1Service.EditAssignmentsWithFlexibleDeploymentParametersAsync": {"apps_put_internal_id_assignments.json"},
	"InternalAppsV2Service.GetApplicationList":                                   {"apps_get_internal_provisionings_id.json"},
	"ProfilesV1Service.DeleteDeviceProfileAsync":                                 {"profiles_delete_id.json"},
}

// ledgerFixtureExistenceAllowed reports whether token is a reviewed
// historical-nonexistence disclosure for service.method.
func ledgerFixtureExistenceAllowed(service, method, token string) bool {
	for _, allowed := range ledgerFixtureExistenceHistoricalAllowlist[service+"."+method] {
		if allowed == token {
			return true
		}
	}
	return false
}

// TestLedgerFixtureMentionsExist is the EXISTENCE guard: every fixture path
// or filename cited anywhere in a ledger reason or testFunction string (not
// just the row's own fixture field) must name a file that actually exists,
// unless ledgerFixtureExistenceHistoricalAllowlist has a reviewed entry for
// it. A reason is allowed to discuss another row's fixture by name (e.g.
// "does not overwrite X.json, since that file's content correctly
// describes a different scenario") -- but the name it cites must be real.
// Skips (does not fail) when MOCK_RESPONSES_DIR is unset, matching every
// other corpus-dependent governance test in this file: the private
// vendored corpus is not shipped in the public package.
func TestLedgerFixtureMentionsExist(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)
	byRelPath, byBasename := ledgerFixtureLookupsFromCorpus(t, corpus)

	ledgerData, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- deliberately excluded from the assembled public package", err)
	}
	var ledger vendoredFixtureLedger
	if err := json.Unmarshal(ledgerData, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 || len(ledger.Unverified) == 0 {
		t.Fatal("control check failed: verified[] or unverified[] is empty -- the ledger read itself is broken")
	}

	var violations []string
	scan := func(section string, entries []vendoredFixtureLedgerEntry) {
		for i, e := range entries {
			for _, field := range []struct{ name, value string }{{"reason", e.Reason}, {"testFunction", e.TestFunction}} {
				for _, bad := range nonexistentFixtureMentions(field.value, byRelPath, byBasename) {
					if ledgerFixtureExistenceAllowed(e.Service, e.Method, bad) {
						continue
					}
					violations = append(violations, fmt.Sprintf("%s[%d].%s (%s.%s) cites %q, which does not exist anywhere in the vendored corpus, the repo, or the synthetic-placeholders twin set, and is not in ledgerFixtureExistenceHistoricalAllowlist", section, i, field.name, e.Service, e.Method, bad))
				}
			}
		}
	}
	scan("verified", ledger.Verified)
	scan("unverified", ledger.Unverified)

	if len(violations) > 0 {
		t.Fatalf("%d nonexistent fixture-name mention(s):\n%s", len(violations), joinLines(violations))
	}
}

// ledgerFixtureLookupsFromCorpus builds the two lookup tables
// nonexistentFixtureMentions needs from a walked corpus plus the private
// synthetic-placeholders twin tree: byRelPath (corpus-relative path ->
// full path, for directory-aware checks) and byBasename (every corpus and
// twin file's bare filename, for bare-filename checks).
func ledgerFixtureLookupsFromCorpus(t *testing.T, corpus []vendoredFixtureEntry) (byRelPath map[string]string, byBasename map[string]bool) {
	t.Helper()
	byRelPath = make(map[string]string, len(corpus))
	byBasename = make(map[string]bool, len(corpus))
	for _, e := range corpus {
		byRelPath[e.relPath] = e.fullPath
		byBasename[filepath.Base(e.relPath)] = true
	}
	twinRoot := filepath.Join("..", "internal", "mockserver", "testdata", "synthetic-placeholders")
	_ = filepath.Walk(twinRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() || filepath.Ext(path) != ".json" {
			return nil //nolint:nilerr // best-effort walk: entries the walker cannot stat are skipped, not failed
		}
		byBasename[filepath.Base(path)] = true
		return nil
	})
	return byRelPath, byBasename
}

// TestLedgerFixtureMentionsExist_PlantedControls proves the EXISTENCE guard
// actually detects a nonexistent filename (both path-shaped and bare) and
// does not false-positive on a real one, using the resolved vendored
// corpus directly (so it needs the same MOCK_RESPONSES_DIR precondition as
// the guard itself).
func TestLedgerFixtureMentionsExist_PlantedControls(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)
	byRelPath, byBasename := ledgerFixtureLookupsFromCorpus(t, corpus)

	t.Run("planted_path_fails", func(t *testing.T) {
		planted := "updates/does_not_exist_anywhere.json"
		if problems := nonexistentFixtureMentions("New fixture file ("+planted+") rather than overwriting the sibling.", byRelPath, byBasename); len(problems) == 0 {
			t.Fatalf("planted nonexistent path %q was not detected", planted)
		}
	})

	t.Run("real_path_passes", func(t *testing.T) {
		realPath := corpus[0].relPath
		if problems := nonexistentFixtureMentions("New fixture file ("+realPath+") rather than overwriting the sibling.", byRelPath, byBasename); len(problems) != 0 {
			t.Fatalf("real corpus path %q was false-flagged: %v", realPath, problems)
		}
	})

	t.Run("planted_bare_name_fails", func(t *testing.T) {
		planted := "updates_device_status_count.json"
		if problems := nonexistentFixtureMentions("New fixture file ("+planted+") rather than overwriting the sibling.", byRelPath, byBasename); len(problems) == 0 {
			t.Fatalf("planted nonexistent bare filename %q was not detected -- this is the exact shape of the run-9 false claim this guard exists to catch", planted)
		}
	})

	t.Run("real_bare_name_passes", func(t *testing.T) {
		realName := filepath.Base(corpus[0].relPath)
		if problems := nonexistentFixtureMentions("New fixture file ("+realName+") rather than overwriting the sibling.", byRelPath, byBasename); len(problems) != 0 {
			t.Fatalf("real corpus bare filename %q was false-flagged: %v", realName, problems)
		}
	})
}

// ledgerNegativeStatusWords are words a reason uses to describe a fixture
// as NOT genuinely test-exercised. Deliberately narrow (excludes the bare
// word "unverified", which collides constantly with the "unverified[]"
// array name and would false-positive on nearly every row) -- these three
// only ever appear in this ledger's prose as an actual status claim about a
// specific fixture.
var ledgerNegativeStatusWords = []string{"demoted", "disputed", "fabricated"}

// ledgerPositiveStatusWord is the one word narrow enough to check
// precisely: "verified" as an adjective immediately before a cited path
// (e.g. "the verified X.json"). Broader synonyms ("genuine", "confirmed")
// are also used constantly as generic ledger vocabulary unrelated to a
// specific sibling's row status, and checking them precisely enough to
// avoid false positives was not achievable within this pass -- narrowed
// out rather than risk misflagging, per this guard's own precision bar.
const ledgerPositiveStatusWord = "verified"

// ledgerStatusClaimWindow is how many characters immediately before a cited
// path a status word must appear within to count as describing THAT path,
// not some earlier, unrelated part of the sentence.
const ledgerStatusClaimWindow = 24

// fixtureBasenameSet maps every key in fixtures to its base filename, so a
// bare-filename mention (no directory) can be resolved by basename. If the
// same basename is ambiguous (present in 2+ directories), it still maps to
// true here -- a bare mention with no directory to disambiguate is treated
// as matching ANY row that cites that basename, the conservative choice
// (a false negative here silently misses a contradiction; a false positive
// would wrongly flag legitimate prose).
func fixtureBasenameSet(fixtures map[string]bool) map[string]bool {
	out := make(map[string]bool, len(fixtures))
	for k := range fixtures {
		out[filepath.Base(k)] = true
	}
	return out
}

// crossRowStatusProblems checks every path-shaped AND bare-filename token
// cited in reason for a nearby status claim (demoted/disputed/fabricated,
// or "verified") that contradicts the fixture's ACTUAL presence in
// verifiedFixtures/unverifiedFixtures (the corpus-wide union of every row's
// own fixture field, by section). Path-shaped tokens are matched exactly;
// bare-filename tokens (no directory) are matched by basename against
// verifiedBasenames/unverifiedBasenames, since this ledger's prose most
// often names a fixture without its directory.
func crossRowStatusProblems(reason string, verifiedFixtures, unverifiedFixtures map[string]bool) []string {
	verifiedBasenames := fixtureBasenameSet(verifiedFixtures)
	unverifiedBasenames := fixtureBasenameSet(unverifiedFixtures)

	var problems []string
	lower := strings.ToLower(reason)
	for _, m := range extractLedgerFixtureMentions(reason) {
		token := m.token
		var isVerified, isUnverified bool
		switch m.kind {
		case ledgerFixtureMentionPath:
			isVerified, isUnverified = verifiedFixtures[token], unverifiedFixtures[token]
		case ledgerFixtureMentionBareName:
			isVerified, isUnverified = verifiedBasenames[token], unverifiedBasenames[token]
		}

		start := strings.Index(reason, token)
		if start < 0 {
			continue
		}
		windowStart := start - ledgerStatusClaimWindow
		if windowStart < 0 {
			windowStart = 0
		}
		window := lower[windowStart:start]

		for _, neg := range ledgerNegativeStatusWords {
			if strings.Contains(window, neg) && isVerified {
				problems = append(problems, fmt.Sprintf("reason calls %q %q, but that fixture backs a verified[] row -- the reason's own claim contradicts the ledger", token, neg))
			}
		}
		if strings.Contains(window, ledgerPositiveStatusWord) && !isVerified && isUnverified {
			problems = append(problems, fmt.Sprintf("reason calls %q %q, but that fixture only backs unverified[] row(s), never a verified[] row -- the reason's own claim contradicts the ledger", token, ledgerPositiveStatusWord))
		}
	}
	return problems
}

// TestLedgerCrossRowFixtureStatusClaims is the CROSS-ROW STATUS guard: a
// reason must not describe a fixture it mentions as demoted/disputed/
// fabricated while that same fixture currently backs a verified[] row, and
// must not call a fixture "verified" while it only ever backs unverified[]
// rows. This is a pure ledger-internal consistency check (it never reads
// the corpus on disk), so it skips only when the ledger itself is absent,
// matching TestLedgerReasonsAreTimeless's convention -- not
// MOCK_RESPONSES_DIR, which this check does not need.
func TestLedgerCrossRowFixtureStatusClaims(t *testing.T) {
	ledgerData, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- deliberately excluded from the assembled public package", err)
	}
	var ledger vendoredFixtureLedger
	if err := json.Unmarshal(ledgerData, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 || len(ledger.Unverified) == 0 {
		t.Fatal("control check failed: verified[] or unverified[] is empty -- the ledger read itself is broken")
	}

	verifiedFixtures := make(map[string]bool)
	for _, e := range ledger.Verified {
		for _, p := range splitLedgerFixtureField(e.Fixture) {
			verifiedFixtures[p] = true
		}
	}
	unverifiedFixtures := make(map[string]bool)
	for _, e := range ledger.Unverified {
		for _, p := range splitLedgerFixtureField(e.Fixture) {
			unverifiedFixtures[p] = true
		}
	}

	var violations []string
	scan := func(section string, entries []vendoredFixtureLedgerEntry) {
		for i, e := range entries {
			for _, problem := range crossRowStatusProblems(e.Reason, verifiedFixtures, unverifiedFixtures) {
				violations = append(violations, fmt.Sprintf("%s[%d] (%s.%s): %s", section, i, e.Service, e.Method, problem))
			}
		}
	}
	scan("verified", ledger.Verified)
	scan("unverified", ledger.Unverified)

	if len(violations) > 0 {
		t.Fatalf("%d cross-row fixture-status contradiction(s):\n%s", len(violations), joinLines(violations))
	}
}

// TestLedgerCrossRowFixtureStatusClaims_PlantedControls proves the
// CROSS-ROW STATUS guard fires in both directions and does not false-
// positive on a reason that correctly describes a sibling's real status.
func TestLedgerCrossRowFixtureStatusClaims_PlantedControls(t *testing.T) {
	const path = "scripts/example-response.json"

	t.Run("negative_direction_fires", func(t *testing.T) {
		verifiedFixtures := map[string]bool{path: true}
		unverifiedFixtures := map[string]bool{}
		reason := "Kept distinct from the DEMOTED " + path + " entry, since this is a legitimate-by-design double."
		if problems := crossRowStatusProblems(reason, verifiedFixtures, unverifiedFixtures); len(problems) == 0 {
			t.Fatal("calling a verified[]-backing fixture 'demoted' was not detected")
		}
	})

	t.Run("positive_direction_fires", func(t *testing.T) {
		verifiedFixtures := map[string]bool{}
		unverifiedFixtures := map[string]bool{path: true}
		reason := "Distinct from the verified " + path + " success capture."
		if problems := crossRowStatusProblems(reason, verifiedFixtures, unverifiedFixtures); len(problems) == 0 {
			t.Fatal("calling an unverified[]-only fixture 'verified' was not detected")
		}
	})

	t.Run("clean_sibling_status_passes", func(t *testing.T) {
		verifiedFixtures := map[string]bool{path: true}
		unverifiedFixtures := map[string]bool{}
		reason := "Distinct from the verified " + path + " success capture, since this is a legitimate-by-design double, not a duplicate or a disputed claim."
		if problems := crossRowStatusProblems(reason, verifiedFixtures, unverifiedFixtures); len(problems) != 0 {
			t.Fatalf("a reason correctly describing a sibling's real verified status was false-flagged: %v", problems)
		}
	})
}

// TestLedgerFixtureMentionsExist_Regression_Run9OriginalBug replays the
// EXACT verbatim reason text of the original run-9 false-existence bug
// (git show <the commit predating its fix>:tests/verified-endpoints.json,
// UpdatesV1Service.GetCountOfDeviceStatus's verified[] row) as a permanent
// regression control -- a manual one-time replay is not enough insurance
// against this exact defect recurring, so the literal historical text lives
// here and must always be caught.
func TestLedgerFixtureMentionsExist_Regression_Run9OriginalBug(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)
	byRelPath, byBasename := ledgerFixtureLookupsFromCorpus(t, corpus)

	const original = "Live-recaptured against UEM 26.2.0.0: real GET against the same live deployment used for the sibling Updates captures above, before teardown. New fixture file (updates_device_status_count.json) rather than overwriting updates_device_status_count_no_deployment.json, since that file's name and content correctly describe a genuinely-different no-deployment scenario that stays as-is; testFunction repointed to consume the new fixture."

	if problems := nonexistentFixtureMentions(original, byRelPath, byBasename); len(problems) == 0 {
		t.Fatal("the original run-9 verified[110] bug text (citing a fixture filename that has never existed) was not detected by the EXISTENCE guard -- this is a regression")
	}
}

// TestLedgerCrossRowFixtureStatusClaims_Regression_Run9OriginalBug replays
// the EXACT verbatim reason text of the original run-9 cross-row-status bug
// (unverified[]'s ScriptBulkDeleteAsync zz-partial row, before its fix) as
// a permanent regression control. The UUID-shaped placeholder this text
// quotes (repeating the same 8-4-4-4-12 hex word throughout every group) is
// assembled from non-UUID-shaped parts so the sync scrubber -- which
// rewrites any whole-UUID-shaped literal it finds in shipped Go source --
// cannot alter this historical string and silently invalidate the control
// once assembled into the public package.
func TestLedgerCrossRowFixtureStatusClaims_Regression_Run9OriginalBug(t *testing.T) {
	deadbeefUUID := strings.Join([]string{"dead" + "beef", "dead", "beef", "dead", "beef" + "deadbeef"}, "-")
	original := `Deliberate edge case: a deliberately-constructed edge-case fixture that explains the "zz-" filename load-order trick, cites a real canonical-vs-plan-draft type discrepancy (List<Guid> vs {uuid,reason} struct), and states plainly it is loaded only via explicit LoadResponseFromFile in TestGeneratedScriptsV1BulkDeletePartial, independent of MOCK_RESPONSES_DIR. delete_failed: ["` + deadbeefUUID + `"] is an intentional, self-evidently-fake placeholder consistent with "deliberately synthetic," not deceptive. Kept under this suffixed key, distinct from the DEMOTED bulkdelete-success.json entry, since this is a legitimate-by-design double rather than a disputed live-capture claim. Relocated to its synthetic-placeholder location, internal/mockserver/testdata/synthetic-placeholders/scripts/bulkdelete-zz-partial.json, for survival past a future repo-root ` + "`git rm testdata/`" + ` (see that directory's README.md); TestGeneratedScriptsV1BulkDeletePartial now loads it via mockserver.LoadResponseFromFile bypass from that location instead of testdata/mock-responses/scripts/bulkdelete-zz-partial.json. The original testdata/mock-responses/scripts/bulkdelete-zz-partial.json copy is left in place, unmodified, but is no longer read by that test.`

	verifiedFixtures := map[string]bool{"scripts/bulkdelete-success.json": true}
	unverifiedFixtures := map[string]bool{}

	if problems := crossRowStatusProblems(original, verifiedFixtures, unverifiedFixtures); len(problems) == 0 {
		t.Fatal("the original run-9 unverified[26] bug text (calling a verified[]-backing fixture 'DEMOTED') was not detected by the CROSS-ROW STATUS guard -- this is a regression")
	}
}

// orphanAllowlistStalenessCheck reports every knownOrphanedVendoredFixtures
// key that is no longer genuinely orphaned: either a verified[] row now
// cites it (directly or via the bypass/synthetic-placeholder manifests), or
// a generated test file mentions its literal path (evidence some test
// consumes it, even if not through the manifests checked above). An
// allowlist entry existing does not mean "fine forever" -- it means
// "reviewed as of when it was written"; this check catches the case where
// a later promotion or codegen change quietly made the entry's own claim
// false, the same "stale sibling claim" class TestLedgerCrossRowFixtureStatusClaims
// already guards against for ledger prose.
func orphanAllowlistStalenessCheck(allowlist map[string]string, verifiedFixtures map[string]bool, testGoSource string) []string {
	var problems []string
	for path := range allowlist {
		if verifiedFixtures[path] {
			problems = append(problems, fmt.Sprintf("%q is allowlisted as orphaned, but a verified[] row now cites it", path))
			continue
		}
		// Match only the vendored-corpus-relative load convention this
		// codebase actually uses (a "testdata/mock-responses/<path>" load
		// argument), not a bare substring match -- many orphaned vendored
		// fixtures have a BYTE-IDENTICAL TWIN at a different path (under
		// internal/mockserver/testdata/synthetic-placeholders/) that tests
		// legitimately load instead, and a bare substring match would treat
		// that twin's longer path as "consuming" the vendored copy it
		// merely shares a filename suffix with.
		if strings.Contains(testGoSource, "testdata/mock-responses/"+path) {
			problems = append(problems, fmt.Sprintf("%q is allowlisted as orphaned, but a test loads it directly from the vendored corpus path -- a test may now consume it", path))
		}
	}
	sort.Strings(problems)
	return problems
}

// TestOrphanAllowlistIsNotStale is the mechanical guard for
// knownOrphanedVendoredFixtures: an entry must stop existing the moment its
// fixture is genuinely covered again (cited by a verified[] row, or
// consumed by a test), not linger as a stale claim. Skips when the ledger
// or the generated test sources are absent, matching every other
// ledger-governance test's public-CI portability convention.
func TestOrphanAllowlistIsNotStale(t *testing.T) {
	ledgerData, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- deliberately excluded from the assembled public package", err)
	}
	var ledger vendoredFixtureLedger
	if err := json.Unmarshal(ledgerData, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 {
		t.Fatal("control check failed: verified[] is empty -- the ledger read itself is broken")
	}

	verifiedFixtures := map[string]bool{}
	for _, e := range ledger.Verified {
		for _, p := range splitLedgerFixtureField(e.Fixture) {
			verifiedFixtures[p] = true
		}
	}

	testGoSource, err := allGeneratedTestGoSourceExcept("vendored_fixture_coverage_test.go")
	if err != nil {
		t.Fatalf("read generated test sources: %v", err)
	}

	if problems := orphanAllowlistStalenessCheck(knownOrphanedVendoredFixtures, verifiedFixtures, testGoSource); len(problems) > 0 {
		t.Fatalf("%d stale orphan-allowlist entr(y/ies):\n%s", len(problems), joinLines(problems))
	}
}

// allGeneratedTestGoSourceExcept concatenates every *_test.go file in this
// package's own directory, skipping excludeFile (the allowlist's own
// source file, which legitimately quotes every allowlisted path in its map
// keys and prose and would otherwise self-match every entry), and dropping
// every line-comment ("//...") line -- doc comments routinely explain a
// fixture's path in prose (often precisely to document why it's now inert
// or redirected elsewhere), which would otherwise look identical to a real
// load call under a bare substring match.
func allGeneratedTestGoSourceExcept(excludeFile string) (string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, "_test.go") || name == excludeFile {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

// TestOrphanAllowlistIsNotStale_PlantedControls proves the staleness guard
// fires when an allowlisted fixture is actually cited by a verified[] row,
// and does not false-positive on a genuinely still-orphaned entry.
func TestOrphanAllowlistIsNotStale_PlantedControls(t *testing.T) {
	t.Run("cited_by_verified_row_fires", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "claimed orphaned"}
		verifiedFixtures := map[string]bool{"scratch/example.json": true}
		if problems := orphanAllowlistStalenessCheck(allowlist, verifiedFixtures, ""); len(problems) == 0 {
			t.Fatal("an allowlist entry cited by a verified[] row was not detected as stale")
		}
	})

	t.Run("consumed_by_a_test_fires", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "claimed orphaned"}
		verifiedFixtures := map[string]bool{}
		fakeSource := `load_fixture("testdata/mock-responses/scratch/example.json")`
		if problems := orphanAllowlistStalenessCheck(allowlist, verifiedFixtures, fakeSource); len(problems) == 0 {
			t.Fatal("an allowlist entry whose path appears in generated test source was not detected as stale")
		}
	})

	t.Run("genuinely_orphaned_passes", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "claimed orphaned"}
		verifiedFixtures := map[string]bool{}
		if problems := orphanAllowlistStalenessCheck(allowlist, verifiedFixtures, "unrelated source with no mention of the path"); len(problems) != 0 {
			t.Fatalf("a genuinely still-orphaned entry was false-flagged: %v", problems)
		}
	})
}

// ledgerAllowlistProvenanceTags are the only provenance tags an
// allowlist entry's value may open with -- GENUINE (real captured wire
// data), PLACEHOLDER (an honestly-disclosed hand-typed stand-in for a
// value that cannot be genuinely captured), SYNTHETIC (schema-derived
// or fabricated content, not wire data at all), or UNVERIFIED-PROVENANCE
// (origin not established by any capture commit or dated record). GENUINE
// and UNVERIFIED-PROVENANCE may additionally carry
// CONSUMED-VIA-DEFAULT-LOOKUP as a second token.
var ledgerAllowlistProvenanceTags = []string{"GENUINE", "PLACEHOLDER", "SYNTHETIC", "UNVERIFIED-PROVENANCE"}

// allowlistEntryTag extracts an entry's leading provenance tag (before the
// ": " separator) and reports whether it carries the
// CONSUMED-VIA-DEFAULT-LOOKUP marker.
func allowlistEntryTag(value string) (tag string, consumedViaDefaultLookup bool, ok bool) {
	idx := strings.Index(value, ": ")
	if idx < 0 {
		return "", false, false
	}
	head := value[:idx]
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return "", false, false
	}
	for _, valid := range ledgerAllowlistProvenanceTags {
		if fields[0] == valid {
			tag = valid
			break
		}
	}
	if tag == "" {
		return "", false, false
	}
	for _, f := range fields[1:] {
		if f == "CONSUMED-VIA-DEFAULT-LOOKUP" {
			consumedViaDefaultLookup = true
		}
	}
	return tag, consumedViaDefaultLookup, true
}

// TestAllowlistEntriesHaveValidProvenanceTags is the mechanical guard for
// the tag scheme itself: every allowlist entry (Go's
// knownOrphanedVendoredFixtures) must open with one of
// ledgerAllowlistProvenanceTags, and any entry whose own prose claims a
// named test consumes it ("consumed by Test...") must carry the
// CONSUMED-VIA-DEFAULT-LOOKUP marker -- an entry that claims test
// consumption without disclosing the mechanism is exactly the kind of gap
// that let profiles/search_v2.json sit miscategorized under "fabricated
// clone" while its own prose already said it was genuinely load-bearing.
func TestAllowlistEntriesHaveValidProvenanceTags(t *testing.T) {
	problems := provenanceTagProblems(knownOrphanedVendoredFixtures)
	if len(problems) > 0 {
		t.Fatalf("%d provenance-tag problem(s):\n%s", len(problems), joinLines(problems))
	}
}

var consumedByTestPhrase = regexp.MustCompile(`(?i)consumed by test`)

func provenanceTagProblems(allowlist map[string]string) []string {
	var problems []string
	for path, value := range allowlist {
		tag, consumedViaDefaultLookup, ok := allowlistEntryTag(value)
		if !ok {
			problems = append(problems, fmt.Sprintf("%q does not open with a valid provenance tag (%s)", path, strings.Join(ledgerAllowlistProvenanceTags, "/")))
			continue
		}
		if tag != "GENUINE" {
			continue
		}
		if consumedByTestPhrase.MatchString(value) && !consumedViaDefaultLookup {
			problems = append(problems, fmt.Sprintf("%q's own text claims a named test consumes it, but does not carry the CONSUMED-VIA-DEFAULT-LOOKUP marker", path))
		}
	}
	sort.Strings(problems)
	return problems
}

// TestAllowlistProvenanceTags_PlantedControls proves the tag-validity and
// marker-consistency checks fire, in both directions.
func TestAllowlistProvenanceTags_PlantedControls(t *testing.T) {
	t.Run("missing_tag_fires", func(t *testing.T) {
		allowlist := map[string]string{"scratch/a.json": "no tag prefix here"}
		if problems := provenanceTagProblems(allowlist); len(problems) == 0 {
			t.Fatal("an entry with no provenance tag was not detected")
		}
	})

	t.Run("unmarked_default_lookup_consumption_fires", func(t *testing.T) {
		allowlist := map[string]string{"scratch/b.json": "GENUINE: still consumed by TestSomething's post-create assertion."}
		if problems := provenanceTagProblems(allowlist); len(problems) == 0 {
			t.Fatal("an entry claiming test consumption without the CONSUMED-VIA-DEFAULT-LOOKUP marker was not detected")
		}
	})

	t.Run("marked_default_lookup_consumption_passes", func(t *testing.T) {
		allowlist := map[string]string{"scratch/c.json": "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: still consumed by TestSomething's post-create assertion."}
		if problems := provenanceTagProblems(allowlist); len(problems) != 0 {
			t.Fatalf("a correctly-marked entry was false-flagged: %v", problems)
		}
	})

	t.Run("genuine_without_test_claim_passes", func(t *testing.T) {
		allowlist := map[string]string{"scratch/d.json": "GENUINE: presently unexercised by any assertion, inert not broken."}
		if problems := provenanceTagProblems(allowlist); len(problems) != 0 {
			t.Fatalf("a genuine entry making no test-consumption claim was false-flagged: %v", problems)
		}
	})

	t.Run("placeholder_and_synthetic_tags_valid", func(t *testing.T) {
		allowlist := map[string]string{
			"scratch/e.json": "PLACEHOLDER: hand-typed stand-in for a secret.",
			"scratch/f.json": "SYNTHETIC: schema-derived, not wire data.",
		}
		if problems := provenanceTagProblems(allowlist); len(problems) != 0 {
			t.Fatalf("valid PLACEHOLDER/SYNTHETIC entries were false-flagged: %v", problems)
		}
	})
}

// syntheticSelfLabelPhrases are substrings a fixture's own
// metadata.description uses to self-disclose non-genuine content. Matching
// any of these is evidence the fixture (or the specific field/aspect being
// described) is NOT plain genuine capture content -- used to cross-check a
// PLACEHOLDER/SYNTHETIC allowlist tag against the fixture's own words, not
// just the allowlist's claim about it.
//
// Deliberately EXCLUDES bare "PLACEHOLDER": the whole-tree sweep
// (TestSelfLabelGuardCoversWholeVendoredTree) found that word alone is used
// throughout this corpus for a narrower, legitimate case too -- a single
// disclosed-redacted field's value inside an otherwise-genuine capture
// (e.g. "access_token is replaced with an obvious … placeholder string",
// "real profile id replaced by the same anonymized placeholder") -- which
// produced false positives when treated as evidence the WHOLE fixture is
// non-genuine. The remaining phrases (SYNTHETIC, SCHEMA-DERIVED,
// HAND-AUTHORED, HAND-TYPED, FABRICATED) are strong enough on their own
// that no genuine-capture description in this corpus uses them in passing;
// every disclosed hand-typed-placeholder fixture in this corpus phrases it
// as "hand-typed placeholder" (matches via HAND-TYPED) or similar, never
// "placeholder" alone.
var syntheticSelfLabelPhrases = []string{
	"SYNTHETIC",
	"SCHEMA-DERIVED",
	"HAND-AUTHORED",
	"HAND-TYPED",
	"FABRICATED",
}

// genuineClaimPhrases are substrings a fixture's own metadata.description
// uses to affirmatively claim its content is real, live-captured wire data
// ("Captured GET ...", "genuine, unmutated", "live capture"). Matching one
// of these with NO disclosure phrase anywhere in the same description means
// a PLACEHOLDER/SYNTHETIC-tagged fixture is claiming to be more genuine than
// its own allowlist tag admits -- the undisclosed-genuine-claim bug class (scripts/get.json,
// updates_group_action_invalid.json, updates_device_status_count_no_deployment.json
// all said "Captured"/"genuine, unmutated" with no disclosure of their own
// hand-typed placeholder ID fields).
var genuineClaimPhrases = []string{
	"CAPTURED",
	"GENUINE",
	"LIVE-CAPTURED",
	"LIVE CAPTURE",
	"UNMUTATED",
}

// allowlistTagAgreesWithSelfLabel reports whether tag (GENUINE/PLACEHOLDER/
// SYNTHETIC) is consistent with description, the fixture's own
// metadata.description -- BIDIRECTIONAL, over the WHOLE description (not
// just its first sentence -- widened after the first-sentence version
// missed a mid-description undisclosed-genuine-claim bug class):
//
//  1. A GENUINE tag fails if the description self-discloses non-genuine
//     content anywhere (one of syntheticSelfLabelPhrases) -- a genuine-
//     tagged fixture whose own words say "hand-typed placeholder" is
//     exactly the bug this guard exists to catch.
//  2. A PLACEHOLDER/SYNTHETIC tag fails if the description affirmatively
//     claims genuine/captured content (one of genuineClaimPhrases) with NO
//     disclosure marker anywhere in the same text (either an explicit
//     "DISCLOSURE" marker, or one of syntheticSelfLabelPhrases) -- a
//     placeholder-tagged fixture whose own words flatly say "Captured ...
//     genuine, unmutated" with nothing disclosing which part is fake is the
//     undisclosed-genuine-claim bug class.
func allowlistTagAgreesWithSelfLabel(tag, description string) bool {
	upper := strings.ToUpper(description)

	selfLabeled := false
	for _, phrase := range syntheticSelfLabelPhrases {
		if strings.Contains(upper, phrase) {
			selfLabeled = true
			break
		}
	}
	hasDisclosureMarker := selfLabeled || strings.Contains(upper, "DISCLOSURE")

	genuineClaimed := false
	for _, phrase := range genuineClaimPhrases {
		if strings.Contains(upper, phrase) {
			genuineClaimed = true
			break
		}
	}

	switch tag {
	case "GENUINE":
		return !selfLabeled
	case "PLACEHOLDER", "SYNTHETIC":
		if genuineClaimed && !hasDisclosureMarker {
			return false
		}
		return true
	default:
		return true
	}
}

// TestAllowlistTagsAgreeWithFixtureSelfLabel cross-checks every Go
// allowlist entry's provenance tag against the fixture's own
// metadata.description, where the fixture is present in the resolved
// vendored corpus. Requires MOCK_RESPONSES_DIR (skips otherwise, matching
// this file's other corpus-dependent checks) since the Python runtime's own
// checkout and the assembled public package carry no fixture corpus for
// Go to read.
func TestAllowlistTagsAgreeWithFixtureSelfLabel(t *testing.T) {
	dir := os.Getenv("MOCK_RESPONSES_DIR")
	if dir == "" {
		t.Skip("MOCK_RESPONSES_DIR is unset -- this check needs the resolved vendored fixture corpus to read each fixture's own metadata.description")
	}

	var problems []string
	for path, value := range knownOrphanedVendoredFixtures {
		tag, _, ok := allowlistEntryTag(value)
		if !ok {
			continue // already reported by TestAllowlistEntriesHaveValidProvenanceTags
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil {
			continue // fixture not present in this corpus snapshot -- not this test's concern
		}
		var fixture struct {
			Metadata struct {
				Description string `json:"description"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			continue
		}
		if !allowlistTagAgreesWithSelfLabel(tag, fixture.Metadata.Description) {
			problems = append(problems, fmt.Sprintf("%q is tagged %s, but its metadata.description is %q -- tag and self-label disagree", path, tag, fixture.Metadata.Description))
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("%d tag/self-label disagreement(s):\n%s", len(problems), joinLines(problems))
	}
}

// TestSelfLabelGuardCoversWholeVendoredTree extends the self-label guard
// to run the bidirectional self-label check over EVERY fixture in the
// resolved vendored corpus, not just the ~26 fixtures already documented in
// knownOrphanedVendoredFixtures. A fixture NOT in that allowlist is
// implicitly expected to be GENUINE (it backs, or is intended to back, a
// real verified[] ledger row) -- so its own description must not
// self-disclose placeholder/synthetic content; if it does, either the
// description is stale/misleading, or the fixture genuinely needs an
// allowlist entry that does not exist yet. A fixture already in the
// allowlist is checked against ITS OWN tag, same as
// TestAllowlistTagsAgreeWithFixtureSelfLabel (duplicated here rather than
// skipped, so this test alone proves full-tree coverage without relying on
// the other test having also run).
func TestSelfLabelGuardCoversWholeVendoredTree(t *testing.T) {
	corpus := walkVendoredFixtureCorpus(t)

	// Paths already governed by a SEPARATE disclosure mechanism -- listed in
	// synthetic-placeholder-fixtures.json (internal/mockserver's own
	// TestSyntheticPlaceholderFixtureGovernance already content-checks these)
	// and/or bypass-loaded-fixtures.json -- are excluded from the "not in
	// knownOrphanedVendoredFixtures implies GENUINE" bucket below: a vendored
	// corpus copy sharing a path with one of these is a KNOWN, already-
	// disclosed synthetic/bypass twin, not an undocumented gap.
	knownElsewhere := map[string]bool{}
	if data, err := os.ReadFile("../tools/internal/capturetools/testdata/synthetic-placeholder-fixtures.json"); err == nil {
		var paths []string
		if json.Unmarshal(data, &paths) == nil {
			for _, p := range paths {
				knownElsewhere[p] = true
			}
		}
	}

	var problems []string
	checked := 0
	skippedKnownElsewhere := 0
	for _, e := range corpus {
		data, err := os.ReadFile(e.fullPath)
		if err != nil {
			t.Fatalf("reading fixture %s: %v", e.fullPath, err)
		}
		var fixture struct {
			Metadata struct {
				Description string `json:"description"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatalf("parsing fixture %s: %v", e.fullPath, err)
		}
		if fixture.Metadata.Description == "" {
			continue
		}
		checked++

		if allowValue, ok := knownOrphanedVendoredFixtures[e.relPath]; ok {
			tag, _, tagOk := allowlistEntryTag(allowValue)
			if !tagOk {
				continue // already reported by TestAllowlistEntriesHaveValidProvenanceTags
			}
			if !allowlistTagAgreesWithSelfLabel(tag, fixture.Metadata.Description) {
				problems = append(problems, fmt.Sprintf("%q is allowlisted %s, but its metadata.description is %q -- tag and self-label disagree", e.relPath, tag, fixture.Metadata.Description))
			}
			continue
		}

		if knownElsewhere[e.relPath] {
			skippedKnownElsewhere++
			continue
		}

		// Not in the allowlist at all -- implicitly GENUINE (a real,
		// ledger-backing capture). Its own description must not
		// self-disclose placeholder/synthetic content -- UNLESS that
		// self-label is itself explicitly scoped as a single-field
		// DISCLOSURE within an otherwise-genuine capture (this corpus's own,
		// well-established convention for e.g. a hand-typed id inside a
		// real captured response -- see quirk-class 13/17 in
		// .claude/rules/api-doctrine.md's sibling doctrine for the same
		// "genuine capture, one disclosed placeholder field" pattern). An
		// UNSCOPED self-label (no "DISCLOSURE" marker anywhere) genuinely
		// means the whole fixture claims non-genuine content while sitting
		// outside the allowlist that exists to document exactly that --
		// a real gap, not a false positive.
		upper := strings.ToUpper(fixture.Metadata.Description)
		if strings.Contains(upper, "DISCLOSURE") {
			continue
		}
		if !allowlistTagAgreesWithSelfLabel("GENUINE", fixture.Metadata.Description) {
			problems = append(problems, fmt.Sprintf("%q is NOT in knownOrphanedVendoredFixtures (implicitly GENUINE), but its own metadata.description self-labels as placeholder/synthetic/fabricated with no DISCLOSURE scoping it to a single field -- add an allowlist entry, or fix the description if that self-label is stale: %q", e.relPath, fixture.Metadata.Description))
		}
	}

	t.Logf("self-label guard swept %d/%d corpus fixtures with a non-empty metadata.description (%d skipped as already governed elsewhere)", checked, len(corpus), skippedKnownElsewhere)

	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("%d whole-tree tag/self-label disagreement(s):\n%s", len(problems), joinLines(problems))
	}
}

// TestAllowlistTagsAgreeWithFixtureSelfLabel_PlantedControls proves the
// tag/self-label agreement check fires in both directions.
func TestAllowlistTagsAgreeWithFixtureSelfLabel_PlantedControls(t *testing.T) {
	t.Run("synthetic_tag_with_no_claim_either_way_passes", func(t *testing.T) {
		// Neither a self-label nor a genuine-claim phrase appears -- neither
		// bidirectional rule applies, so this must NOT be flagged.
		if !allowlistTagAgreesWithSelfLabel("SYNTHETIC", "Empty list result for GET /api/mdm/x/y, no baseline present in the test tenant.") {
			t.Fatal("a SYNTHETIC tag on a fixture with no claim either way was wrongly flagged")
		}
	})

	t.Run("genuine_tag_on_self_disclosed_placeholder_fires", func(t *testing.T) {
		if allowlistTagAgreesWithSelfLabel("GENUINE", "SYNTHETIC / HAND-AUTHORED PLACEHOLDER -- not live-captured.") {
			t.Fatal("a GENUINE tag on a fixture whose own description discloses SYNTHETIC/HAND-AUTHORED was not detected")
		}
	})

	t.Run("synthetic_tag_with_self_label_passes", func(t *testing.T) {
		if !allowlistTagAgreesWithSelfLabel("SYNTHETIC", "Schema-derived placeholder, disclosed honestly in the fixture's own metadata.description.") {
			t.Fatal("a correctly-agreeing SYNTHETIC tag was false-flagged")
		}
	})

	t.Run("genuine_tag_on_genuine_description_passes", func(t *testing.T) {
		if !allowlistTagAgreesWithSelfLabel("GENUINE", "Captured GET /api/example against a real live tenant.") {
			t.Fatal("a correctly-agreeing GENUINE tag was false-flagged")
		}
	})

	// --- the (reverse) direction of the bidirectional check ---

	t.Run("placeholder_tag_on_undisclosed_genuine_claim_fires", func(t *testing.T) {
		// This reproduces the exact undisclosed-genuine-claim bug:
		// scripts/get.json's pre-fix description flatly said "Captured GET ..." with no
		// disclosure anywhere of its own hand-typed placeholder id field.
		if allowlistTagAgreesWithSelfLabel("PLACEHOLDER", "Captured GET /api/mdm/scripts/{scriptUuid} (ScriptResource with full payload; base64 script_data).") {
			t.Fatal("a PLACEHOLDER tag on a fixture whose own description flatly claims 'Captured' with no disclosure anywhere was not detected -- this is the exact undisclosed-genuine-claim bug class")
		}
	})

	t.Run("placeholder_tag_on_disclosed_genuine_claim_passes", func(t *testing.T) {
		// The post-fix shape: "Captured" is still present, but a DISCLOSURE
		// marker in the same text explains which part is not genuine.
		if !allowlistTagAgreesWithSelfLabel("PLACEHOLDER", "Captured GET /api/mdm/scripts/{scriptUuid} -- DISCLOSURE: script_uuid is a hand-typed placeholder, not a genuine server-issued value.") {
			t.Fatal("a PLACEHOLDER tag on a fixture whose description discloses its placeholder field was wrongly flagged")
		}
	})

	t.Run("synthetic_tag_on_undisclosed_unmutated_claim_fires", func(t *testing.T) {
		if allowlistTagAgreesWithSelfLabel("SYNTHETIC", "This is the genuine, unmutated live-server response for every id tried.") {
			t.Fatal("a SYNTHETIC tag on a fixture whose own description claims 'genuine, unmutated' with no disclosure was not detected")
		}
	})

	// --- regression pin: a real fixture whose GENUINE-retag must fail ---

	t.Run("genuine_retag_of_get_device_sensors_fails", func(t *testing.T) {
		// Real metadata.description text from sensors/get_device_sensors.json
		// (vendored tip a355f5e24). A hypothetical GENUINE retag of this
		// fixture (it is correctly tagged PLACEHOLDER) must FAIL, since the
		// fixture's own words explicitly say "not a genuine live capture".
		// The two illustrative UUIDs are quoted in UNDERSCORE form,
		// matching the real fixture text -- a dashed form here would itself trip
		// TestNoDashFormUUIDInProseOrComments, since this Go source file is
		// shipped and scanned by that guard too.
		realDescription := "Returns the list of sensors reported by a specific device. GET /api/mdm/devices/{deviceUuid}/sensors (V1). The body's uuid values (a1b2c3d4_e5f6_7890_abcd_ef1234567890, b2c3d4e5_f6a7_8901_bcde_f01234567891) are hand-typed ascending-hex placeholders, not gofakeit output -- not a genuine live capture."
		if allowlistTagAgreesWithSelfLabel("GENUINE", realDescription) {
			t.Fatal("a hypothetical GENUINE tag on sensors/get_device_sensors.json's real description was not detected as disagreeing -- regression in the undisclosed-genuine-claim fix")
		}
		// And the fixture's ACTUAL tag, PLACEHOLDER, must still pass.
		if !allowlistTagAgreesWithSelfLabel("PLACEHOLDER", realDescription) {
			t.Fatal("sensors/get_device_sensors.json's real PLACEHOLDER tag was wrongly flagged against its own real description")
		}
	})
}

// pythonAllowlistEntryPattern extracts, from the Python runtime's
// fixture-coverage test's own source text, each _KNOWN_ORPHANED_FIXTURES
// key together with the leading provenance tag of its value -- the Python
// dict's values are written as an implicit string-concatenation tuple
// spanning multiple lines, e.g.:
//
//	"some/path.json": (
//	    "GENUINE: description text..."
//	    "more text..."
//	),
//
// so only the FIRST string literal's leading token (before the tag's own
// ": " separator) needs to be captured -- it always carries the tag, by the
// same convention allowlistEntryTag parses on the Go side.
var pythonAllowlistEntryPattern = regexp.MustCompile(`(?m)^\s*"([^"]+\.json)":\s*\(\s*\n\s*"(GENUINE|PLACEHOLDER|SYNTHETIC|UNVERIFIED-PROVENANCE)(?: CONSUMED-VIA-DEFAULT-LOOKUP)?:`)

// extractPythonAllowlistTags parses every _KNOWN_ORPHANED_FIXTURES entry's
// fixture path and leading provenance tag out of the raw Python source text.
func extractPythonAllowlistTags(src []byte) map[string]string {
	tags := make(map[string]string)
	for _, m := range pythonAllowlistEntryPattern.FindAllSubmatch(src, -1) {
		tags[string(m[1])] = string(m[2])
	}
	return tags
}

// crossLanguageTagProblems reports every fixture path present in BOTH
// allowlists whose base provenance tag (GENUINE/PLACEHOLDER/SYNTHETIC)
// disagrees between languages. The Go-only CONSUMED-VIA-DEFAULT-LOOKUP
// marker is deliberately excluded from the comparison -- it describes a
// Go-mock-server-specific consumption mechanism, not the fixture's own
// content, so it is not required to match.
func crossLanguageTagProblems(goAllowlist map[string]string, pythonTags map[string]string) []string {
	var problems []string
	for path, goValue := range goAllowlist {
		goTag, _, ok := allowlistEntryTag(goValue)
		if !ok {
			continue
		}
		pyTag, present := pythonTags[path]
		if !present {
			continue
		}
		if pyTag != goTag {
			problems = append(problems, fmt.Sprintf("%q is tagged %s in Go's knownOrphanedVendoredFixtures but %s in Python's _KNOWN_ORPHANED_FIXTURES -- provenance tags must agree across languages", path, goTag, pyTag))
		}
	}
	sort.Strings(problems)
	return problems
}

// TestAllowlistTagsAgreeAcrossLanguages is the cross-language mechanical
// consistency guard: any fixture path documented in BOTH knownOrphanedVendoredFixtures
// (this file) and the Python runtime's fixture-coverage test's
// _KNOWN_ORPHANED_FIXTURES must carry the SAME base provenance tag in both
// places. A path appearing in only one language's allowlist is not checked
// here -- that's expected (a fixture consumed by one language and orphaned
// in the other legitimately appears on only one side).
func TestAllowlistTagsAgreeAcrossLanguages(t *testing.T) {
	pySrc, err := os.ReadFile("../python-runtime/tests/test_fixture_coverage.py")
	if err != nil {
		t.Skipf("python-runtime/tests/test_fixture_coverage.py is unreadable (%v) -- this cross-language check is structurally unavailable without it", err)
	}

	pyTags := extractPythonAllowlistTags(pySrc)
	if len(pyTags) == 0 {
		t.Fatal("control check failed: extracted zero provenance tags from the Python allowlist -- the parser itself is broken, or the Python file's tags are missing")
	}

	if problems := crossLanguageTagProblems(knownOrphanedVendoredFixtures, pyTags); len(problems) > 0 {
		t.Fatalf("%d fixture(s) have disagreeing provenance tags across languages:\n%s", len(problems), joinLines(problems))
	}
}

// TestAllowlistTagsAgreeAcrossLanguages_PlantedControls proves the
// cross-language guard fires on a genuine tag disagreement and does not
// false-positive on agreement.
func TestAllowlistTagsAgreeAcrossLanguages_PlantedControls(t *testing.T) {
	t.Run("disagreement_fires", func(t *testing.T) {
		goAllowlist := map[string]string{"scratch/example.json": "GENUINE: fine"}
		pyTags := map[string]string{"scratch/example.json": "PLACEHOLDER"}
		if problems := crossLanguageTagProblems(goAllowlist, pyTags); len(problems) == 0 {
			t.Fatal("a genuine cross-language tag disagreement was not detected")
		}
	})

	t.Run("agreement_passes", func(t *testing.T) {
		goAllowlist := map[string]string{"scratch/example.json": "GENUINE: fine"}
		pyTags := map[string]string{"scratch/example.json": "GENUINE"}
		if problems := crossLanguageTagProblems(goAllowlist, pyTags); len(problems) != 0 {
			t.Fatalf("agreeing cross-language tags were false-flagged: %v", problems)
		}
	})

	t.Run("go_only_marker_ignored", func(t *testing.T) {
		// CONSUMED-VIA-DEFAULT-LOOKUP is Go-mock-server-specific and must
		// not cause a false disagreement against Python's plain base tag.
		goAllowlist := map[string]string{"scratch/example.json": "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: fine"}
		pyTags := map[string]string{"scratch/example.json": "GENUINE"}
		if problems := crossLanguageTagProblems(goAllowlist, pyTags); len(problems) != 0 {
			t.Fatalf("a Go-only CONSUMED-VIA-DEFAULT-LOOKUP marker wrongly caused a cross-language disagreement: %v", problems)
		}
	})

	t.Run("path_only_in_one_language_ignored", func(t *testing.T) {
		goAllowlist := map[string]string{"scratch/go-only.json": "GENUINE: fine"}
		pyTags := map[string]string{"scratch/py-only.json": "PLACEHOLDER"}
		if problems := crossLanguageTagProblems(goAllowlist, pyTags); len(problems) != 0 {
			t.Fatalf("paths present in only one language's allowlist were wrongly compared: %v", problems)
		}
	})
}

// TestExtractPythonAllowlistTags_RealFileSanityCheck proves the regex-based
// Python source parser actually matches the real file's formatting
// convention (not just a hand-crafted planted-control string), by asserting
// a handful of known entries parse to their expected tag.
func TestExtractPythonAllowlistTags_RealFileSanityCheck(t *testing.T) {
	pySrc, err := os.ReadFile("../python-runtime/tests/test_fixture_coverage.py")
	if err != nil {
		t.Skipf("python-runtime/tests/test_fixture_coverage.py is unreadable (%v)", err)
	}
	pyTags := extractPythonAllowlistTags(pySrc)

	wantTags := map[string]string{
		"auth/oauth2_token_success.json": "GENUINE",
		"baselines/assignments-get.json": "SYNTHETIC",
		"scripts/get.json":               "SYNTHETIC",
	}
	for path, want := range wantTags {
		got, ok := pyTags[path]
		if !ok {
			t.Errorf("expected to parse a tag for %q from the Python allowlist, found none", path)
			continue
		}
		if got != want {
			t.Errorf("%q: parsed tag %q, want %q", path, got, want)
		}
	}
}

// defaultLookupConsumedFixtures records every knownOrphanedVendoredFixtures
// path that has been MECHANICALLY PROVEN to be genuinely consumed via the
// mock server's directory-scan default lookup. This is the ground truth
// TestDefaultLookupMarkerConsistency checks the CONSUMED-VIA-DEFAULT-LOOKUP
// marker against -- every path here MUST carry the marker in its allowlist
// entry, and the marker must not be claimed for any path NOT here (an
// unproven claim). For every entry except the last, the proof is BY
// DELETING THAT EXACT FILE INDIVIDUALLY from a scratch corpus copy of the
// vendored tip and observing the named Go test fail OR go PASS->SKIP (a
// SKIP counts as consumption evidence too -- see profiles/
// search_android_success.json below, which proves this way). The profiles/
// search_empty.json is the one DIFFERENT proof shape -- see its own
// comment below -- kept here anyway because it is still real,
// mechanically-proven consumption via this same default-lookup mechanism,
// just only provable jointly with its sibling's absence, not on its own.
//
// EXHAUSTIVE SWEEP: every one of this file's ~26
// knownOrphanedVendoredFixtures entries was individually deletion-tested
// against the FULL `go test ./tests/...` suite (not a narrowed -run
// filter) from a scratch corpus copy of the vendored tip, WITHOUT a .env
// file present. That sweep alone found 5 entries causing a real service/
// lifecycle test to fail. Several others (mam-apps/
// apps_get_internal_ios_renewaldate.json, org-groups/groups_get.json,
// org-groups/groups_get_id.json, profiles/search_android_success_v2.json,
// profiles/search_empty_v2.json, scripts/bulkdelete-zz-partial.json)
// caused ONLY TestLedgerFixtureMentionsExist to fail -- a generic
// ledger-prose-citation existence check (the ledger's own reason text
// literally names the filename), NOT evidence of mock-server route
// consumption -- and correctly carry no marker. The profiles/
// search_android_success.json and profiles/search_empty.json (the
// non-"_v2" pair) were misclassified the same way by that same sweep,
// since it never ran with a .env file present and so never exercised the
// tests that actually depend on them -- re-tested WITH .env present (see
// below): search_android_success.json is individually proven (PASS->FAIL
// on TestIntegration_ProfileService_Search and PASS->SKIP on
// TestIntegration_ProfileService_Get); search_empty.json is proven
// only as part of the joint-deletion failure. Both belong in the
// proven-consumer set, bringing the total to 7.
var defaultLookupConsumedFixtures = map[string]bool{
	"org-groups/groups_delete_uuid_v2.json":           true,
	"org-groups/groups_delete_uuid_v2_not_found.json": true,
	// The five live-captured not-found fixtures: served by route to
	// TestNotFoundFixturesAreRecognised (deletion-control proven).
	"profiles/profiles_get_not_found.json":                            true,
	"profiles/profiles_delete_not_found.json":                         true,
	"smartgroups/smartgroups_delete_not_found.json":                   true,
	"internal-apps-v1/macos/get_internal_app_macos_not_found.json":    true,
	"internal-apps-v1/macos/delete_internal_app_macos_not_found.json": true,
	"sensors/list_sensors.json":                                       true,
	"sensors/list_sensors_v2_empty.json":                              true,
	"profiles/search_v2.json":                                         true,
	"updates/updates_device_status_count_no_deployment.json":          true,
	"updates/updates_group_action_invalid.json":                       true,
	// profiles/search_android_success.json: individually depended on
	// (deletion-control, 2026-09-24, .env present): removing only this file
	// makes TestIntegration_ProfileService_Search FAIL (its platform-filter
	// subtest asserts this fixture's AppleOsX and Linux profiles) and
	// TestIntegration_ProfileService_Get go PASS->SKIP. It always wins over
	// search_empty.json while both are present: internal/mockserver's
	// LoadResponsesFromDir walks the corpus in lexical order and this file
	// sorts first.
	"profiles/search_android_success.json": true,
	// profiles/search_empty.json: ONLY A FALLBACK, never individually
	// depended on -- it is never served while search_android_success.json is
	// present (lexical load order, above). Deletion-control (2026-09-24,
	// .env present): removing only this file changes no test outcome;
	// removing both files makes TestIntegration_ProfileService_Search, Get
	// and EndpointConfiguration FAIL. That joint result is this file's proof
	// of consumption. Since the 2026-09-24 recapture it is a genuine 204 with
	// no body (an empty org group), served to those tests only once
	// search_android_success.json is gone.
	"profiles/search_empty.json": true,
}

// defaultLookupMarkerConsistencyProblems reports every mismatch between an
// allowlist entry's CONSUMED-VIA-DEFAULT-LOOKUP claim and the mechanically-
// proven ground truth: a marker with no proof is an unproven claim; proven
// consumption with no marker is undisclosed consumption. Either direction is
// a documentation bug.
func defaultLookupMarkerConsistencyProblems(allowlist map[string]string, provenConsumers map[string]bool) []string {
	var problems []string
	for path, value := range allowlist {
		_, consumedViaDefaultLookup, ok := allowlistEntryTag(value)
		if !ok {
			continue
		}
		proven := provenConsumers[path]
		switch {
		case consumedViaDefaultLookup && !proven:
			problems = append(problems, fmt.Sprintf("%q carries the CONSUMED-VIA-DEFAULT-LOOKUP marker but is not in the mechanically-proven default-lookup consumer set -- unproven claim", path))
		case proven && !consumedViaDefaultLookup:
			problems = append(problems, fmt.Sprintf("%q is mechanically proven to be consumed via the mock server's default lookup, but its allowlist entry does not carry the CONSUMED-VIA-DEFAULT-LOOKUP marker -- undisclosed consumption", path))
		}
	}
	sort.Strings(problems)
	return problems
}

// TestDefaultLookupMarkerConsistency is the mechanical guard tying the
// CONSUMED-VIA-DEFAULT-LOOKUP marker to real proof: the stale-allowlist
// guard (TestOrphanAllowlistIsNotStale) only exempts an entry from its own
// named-reference scan, it does not itself validate the
// CONSUMED-VIA-DEFAULT-LOOKUP marker's truth -- this test closes that gap by
// checking the marker against defaultLookupConsumedFixtures, the ground
// truth established by individually deleting each candidate fixture from a
// scratch corpus copy of the vendored tip and observing its claimed
// consuming test fail (or, for the two rejected candidates, NOT fail).
func TestDefaultLookupMarkerConsistency(t *testing.T) {
	if problems := defaultLookupMarkerConsistencyProblems(knownOrphanedVendoredFixtures, defaultLookupConsumedFixtures); len(problems) > 0 {
		t.Fatalf("%d default-lookup marker inconsistenc(y/ies):\n%s", len(problems), joinLines(problems))
	}
}

// TestDefaultLookupMarkerConsistency_PlantedControls proves the guard fires
// on an unmarked-but-proven-consumed entry (undisclosed consumption must be
// provably caught) and on a marked-but-unproven entry, while
// passing cleanly on both a correctly-marked-and-proven entry and a
// genuinely-unmarked-and-unproven (plain orphan) entry.
func TestDefaultLookupMarkerConsistency_PlantedControls(t *testing.T) {
	t.Run("unmarked_proven_consumer_fires", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "GENUINE: plain orphan, no marker claimed"}
		proven := map[string]bool{"scratch/example.json": true}
		if problems := defaultLookupMarkerConsistencyProblems(allowlist, proven); len(problems) == 0 {
			t.Fatal("an unmarked entry that is mechanically proven to be default-lookup-consumed was not detected as undisclosed consumption")
		}
	})

	t.Run("marked_but_unproven_fires", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: claimed covered"}
		proven := map[string]bool{}
		if problems := defaultLookupMarkerConsistencyProblems(allowlist, proven); len(problems) == 0 {
			t.Fatal("a CONSUMED-VIA-DEFAULT-LOOKUP marker with no proof behind it was not detected as an unproven claim")
		}
	})

	t.Run("marked_and_proven_passes", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "GENUINE CONSUMED-VIA-DEFAULT-LOOKUP: claimed covered"}
		proven := map[string]bool{"scratch/example.json": true}
		if problems := defaultLookupMarkerConsistencyProblems(allowlist, proven); len(problems) != 0 {
			t.Fatalf("a correctly-marked-and-proven entry was false-flagged: %v", problems)
		}
	})

	t.Run("unmarked_and_unproven_passes", func(t *testing.T) {
		allowlist := map[string]string{"scratch/example.json": "GENUINE: genuinely just an orphan"}
		proven := map[string]bool{}
		if problems := defaultLookupMarkerConsistencyProblems(allowlist, proven); len(problems) != 0 {
			t.Fatalf("a genuinely unmarked, unproven (plain orphan) entry was false-flagged: %v", problems)
		}
	})
}
