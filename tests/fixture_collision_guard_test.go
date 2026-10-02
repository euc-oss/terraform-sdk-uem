package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// fixtureEndpointKey identifies a fixture's (endpoint, method, version) --
// the exact tuple internal/mockserver groups fixtures by before scoring.
type fixtureEndpointKey struct {
	endpoint string
	method   string
	version  string
}

// fixtureEntry is one fixture file found under a given fixtureEndpointKey.
type fixtureEntry struct {
	path        string
	queryParams map[string]string
}

// TestNoDuplicateFixtureEndpoints is a permanent structural guard: no two
// fixture files under testdata/mock-responses/ may declare the same
// (endpoint, method, version) unless they're distinguished by a recorded
// request.query_params (a real disambiguator the mock server's scoreMatch
// honors) or are explicitly allowlisted below with a reason.
//
// Why this exists: three same-endpoint
// duplicates surfaced REACTIVELY (baselines osversions/
// platforms/templates, then a stray upload_certificate_success.json) --
// each only when a test happened to break. The internal/mockserver's
// RequestMatcher.Match resolves a tie by directory-walk load order (first
// equal-or-higher score wins, ties never override), which is deterministic
// per run but not meaningful -- nothing about it reflects which fixture a
// test actually wants. A 4th case (profiles/{id} GET v2: get_apple_68749/
// get_android_68748/get_profile_12345/profiles_get_apple_id, fixed in this
// same sweep) was ALREADY silently ambiguous and just hadn't broken a test
// yet, because the consuming test's assertion was too weak to notice.
// This guard turns "we find out when a test happens to break" into
// "the suite refuses to load a genuinely ambiguous state at all".
func TestNoDuplicateFixtureEndpoints(t *testing.T) {
	groups := map[fixtureEndpointKey][]fixtureEntry{}

	err := filepath.Walk(mockserver.ResolveResponsesDir("../testdata/mock-responses"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", path, readErr)
		}
		var fixture struct {
			Metadata struct {
				Endpoint string `json:"endpoint"`
				Method   string `json:"method"`
				Version  string `json:"version"`
			} `json:"metadata"`
			Request struct {
				QueryParams map[string]string `json:"query_params"`
			} `json:"request"`
		}
		if unmarshalErr := json.Unmarshal(raw, &fixture); unmarshalErr != nil {
			// Not every JSON file under this tree is a capture fixture in this
			// exact shape (e.g. manifest files) -- skip rather than fail the walk.
			return nil //nolint:nilerr // not every JSON file here is a capture fixture; non-fixture JSON is skipped
		}
		if fixture.Metadata.Endpoint == "" || fixture.Metadata.Method == "" {
			return nil
		}
		k := fixtureEndpointKey{fixture.Metadata.Endpoint, fixture.Metadata.Method, fixture.Metadata.Version}
		groups[k] = append(groups[k], fixtureEntry{path: path, queryParams: fixture.Request.QueryParams})
		return nil
	})
	if err != nil {
		t.Fatalf("walking testdata/mock-responses: %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("control check failed: found zero fixture groups -- the walk itself is broken")
	}

	for k, entries := range groups {
		if len(entries) < 2 {
			continue
		}

		// Real disambiguator: every entry has a NON-EMPTY, mutually distinct
		// query_params map. internal/mockserver's scoreMatch gives +10 per
		// matching query key/value, so distinct required query params let the
		// mock server pick correctly regardless of load order (e.g. the
		// updates_group_action_{start,stop,invalid}.json trio, disambiguated
		// by ?action=).
		if allDistinctByQueryParams(entries) {
			continue
		}

		if reason, ok := knownFixtureCollisionAllowlist[k]; ok {
			t.Logf("allowlisted collision %+v (%d files): %s", k, len(entries), reason)
			continue
		}

		paths := make([]string, len(entries))
		for i, e := range entries {
			paths[i] = e.path
		}
		sort.Strings(paths)
		t.Errorf("duplicate fixture endpoint %+v: %d files claim it with no query-param disambiguation and no allowlist entry — exactly one canonical fixture per endpoint, or add a documented, justified entry to knownFixtureCollisionAllowlist:\n  %v",
			k, len(entries), paths)
	}
}

func allDistinctByQueryParams(entries []fixtureEntry) bool {
	seen := map[string]bool{}
	for _, e := range entries {
		if len(e.queryParams) == 0 {
			return false
		}
		key := fmt.Sprintf("%v", e.queryParams)
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

// knownFixtureCollisionAllowlist documents every (endpoint, method, version)
// group that legitimately has more than one fixture file, and WHY it's safe
// despite scoring identically in internal/mockserver's RequestMatcher. Every
// entry here was individually verified by
// reading its consuming test(s), not assumed. Add a new entry ONLY after the
// same verification -- an unverified entry defeats the whole guard.
var knownFixtureCollisionAllowlist = map[fixtureEndpointKey]string{
	{"/api/mdm/scripts/{scriptUuid}", "GET", "1"}: "get.json auto-loads (lexically first) and is consumed by TestGeneratedScriptsV1Lifecycle and the CatalogDisplay overlay regression guard (TestGeneratedScriptResourceCatalogDisplay), both of which depend on its specific pre-existing shape; get_live.json is a genuine live capture, loaded via LoadResponseFromFile in the isolated TestGeneratedScriptsV1GetLiveCapture, never in the same server instance as get.json.",

	{"/api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}/device-status", "GET", "1"}: "updates_device_status_count_no_deployment.json auto-loads (lexically first) and backs TestGeneratedUpdatesV1ReadOnly's genuine 404-no-deployment case; updates_device_status_count_success.json is a genuine live capture of the 200-success case (a real deployment exists), loaded via LoadResponseFromFile in the isolated TestGeneratedUpdatesV1GetCountOfDeviceStatusSuccess, never in the same server instance as the no-deployment fixture. Named _success (not just a bare replacement) specifically so it sorts after _no_deployment and never wins the full-tree tie.",

	{"/api/mam/apps/internal/application", "POST", "1"}: "each platform variant (apk/ipa/msi/pkg/zip) is loaded via LoadMockResponses scoped to its OWN subdirectory (tests/generated_internalappsv1_lifecycle_test.go's runInternalAppsV1Lifecycle) -- never loaded together, never actually contend for the same request. Guards its own prior regression (TestMamV1InternalAppGetByIdRenewalDateFixture) via literal per-platform IDs.",

	{"/api/mdm/groups/{organizationGroupUuid}/scripts/bulkdelete", "POST", "1"}: "bulkdelete-success.json auto-loads (lexically first); bulkdelete-zz-partial.json is loaded via mockserver.LoadResponseFromFile in an isolated mock server built just for TestGeneratedScriptsV1BulkDeletePartial -- documented '-zz-' filename convention, never in the same server instance as the success fixture.",

	{"/api/mdm/profiles/68910", "GET", "2"}: "profiles_apple_get_after_{create,update,delete}.json: after_create wins the full-tree auto-load by alphabetical order (matches TestGeneratedProfilesV2AppleLifecycle's immediate post-create GET intent); after_update/after_delete are loaded via LoadResponseFromFile in their own dedicated, isolated tests (TestGeneratedProfilesV2AppleGetAfterUpdate/Delete) and never reached through the auto-scan pool.",

	{"/api/mdm/profiles/68911", "GET", "2"}: "profiles_appleosx_get_after_{create,update,delete}.json: same pattern as the Apple (68910) group above, for the AppleOsX lifecycle test.",

	{"/api/mdm/profiles/68912", "GET", "2"}: "profiles_winrt_get_after_{create,update,delete}.json: same pattern as the Apple (68910) group above, for the WinRT lifecycle test.",

	{"/api/mdm/smartgroups/{id}", "GET", "1"}: "smartgroups_get_after_{create,update,delete}.json: three lifecycle stages of one route. The shared mock server serves after_create by default and after_update/after_delete once it has observed a 2xx PUT/DELETE for that id (TestSharedMockServer_SmartGroupGetFollowsObservedLifecycle); TestGeneratedSmartGroupsV1GetAfterUpdate/Delete still load theirs via LoadResponseFromFile.",

	{"/api/system/groups/{id}", "GET", "1"}: "org-groups/groups_get.json and org-groups/groups_get_id.json are two genuinely different live captures of the same v1 route. groups_get_id.json backs the OrganizationGroupsService.GetAsync verified[] row: TestGeneratedOrganizationGroupsV1GetByID loads it into an isolated mock server because groups_get.json sorts first under directory-scan lookup, and deleting groups_get_id.json makes that test FAIL (deletion-control, 2026-09-24). groups_get.json is cited by no verified[] row and consumed by no functional test (deletion-control, 2026-09-24: only ledger and corpus guards react); its inert status is recorded in vendored_fixture_coverage_test.go. metadata.version=\"1\" is correct on both: GET /api/system/groups/{id} has no v2 route (systemv2.json's only route on this path is POST; doctrine quirk 1). No query-param disambiguator is possible (neither request records one, and {id} is templated out of the match key), so the tie is permanent and allowlisted, not fixed.",

	{"/api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}", "POST", "1"}: "updates_group_action_{start,stop,invalid}.json are ALREADY correctly disambiguated by a real, recorded request.query_params (?action=Start/Stop/Pause) -- would also pass the query-param check above; listed here only as a documented cross-reference to TestGeneratedUpdatesV1DevicesUpdateAction's own extensive comment.",

	{"/api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}/deployment", "POST", "1"}: "updates_deployment_create.json auto-loads; updates_deployment_create_conflict.json is loaded via LoadResponseFromFile in the isolated TestGeneratedUpdatesV1CreateConflict.",

	{"/api/mdm/profiles/search", "GET", "1"}: "KNOWN, DELIBERATE: search_android_success.json and search_empty.json are both GENUINE captures (vendored 4f0dd5fe5, 2026-09-24) of different scenarios on this v1 route: a populated org group (one AppleOsX and one Linux profile; the filename is historical) and an empty org group, which the server answers with 204 and no body. Neither is a duplicate or a guess; the mock server cannot serve both. Their consumers are TestIntegration_ProfileService_Search, TestIntegration_ProfileService_Get and TestIntegration_ProfileService_EndpointConfiguration (tests/profile_integration_test.go), which skip unless a ../.env file is present. Selection is deterministic: internal/mockserver's LoadResponsesFromDir walks the corpus in lexical order, so search_android_success.json is served while both are present and search_empty.json is only a fallback. deletion-control on 2026-09-24 with a .env present shows: removing only search_android_success.json makes TestIntegration_ProfileService_Search FAIL (its platform-filter subtest asserts the fixture's AppleOsX and Linux profiles) and TestIntegration_ProfileService_Get go PASS->SKIP; removing only search_empty.json changes no outcome; removing both makes Search, Get and EndpointConfiguration FAIL. TestIntegration_BasicAuth_ProfileSearch uses its own bypass fixture instead.",

	{"/api/mdm/profiles/search", "GET", "2"}: "RESOLVED at the ledger/codegen level, as a follow-up to the open item on the sibling v1-route row above: of the 5 files (search_v2.json, profiles_search_v2.json, search_android_success_v2.json, search_empty_v2.json, profiles_get_search.json), content comparison against the real v2 model (AdditionalInfo/ProfileList/TotalResults, confirmed by TestGeneratedProfilesV2SearchV2) and the real v1 model (Page/PageSize/Total/Profiles[].Id.Value) showed profiles_search_v2.json (2026-07-30 live capture), search_v2.json, and profiles_get_search.json (cap_profiles, 2026-08-05 live capture) are genuine, independently-provenanced v2-shaped captures, while search_android_success_v2.json/search_empty_v2.json are near-certainly mislabeled clones of the v1-shaped search_android_success.json/search_empty.json (as those files were before their 2026-09-24 recapture) (Page/PageSize/Total wrapper with a hand-renamed ProfileList field, no capture narrative) -- matching neither model. verified-endpoints.json's SearchProfiles-collision entry was updated to drop the two mislabeled clones from its fixture list, so internal-source/emit/python's JoinFixtures no longer joins them into the generated Python SearchProfiles test. This entry stays allowlisted because all 5 files still physically collide on (endpoint, method, version) here at the mock-server structural level (files are kept on disk, not deleted, per 'a wrong deletion is worse than leaving it allowlisted-but-tracked') -- only the ledger-driven codegen join was fixed, not this file-level collision.",

	{"/api/mdm/profiles/platforms/appleosx/create", "POST", "2"}: "appleosx/create-v2.json (TestAppleOsXCreateProfile) and profiles_appleosx_create.json (TestGeneratedProfilesV2AppleOsXLifecycle) both go through the full-tree auto-scan and DO genuinely tie -- but both consuming tests deliberately assert only `profileID != 0` (see TestGeneratedProfilesV2AppleOsXLifecycle's own comment: 'why this only checks non-zero rather than the fixture's own captured id'), since the real endpoint returns a bare int with no other content to verify. Whichever wins the tie is functionally inert to both tests' pass/fail. A third, unreferenced duplicate (create_appleosx_network_success.json) was dead code and was deleted during this sweep.",

	{"/api/mam/apps/internal/begininstall", "POST", "1"}: "internal-apps-v1/begininstall-response.json and internal-apps-v1/ipa/internal_app_begininstall_ipa.json are both genuine, independently-provenanced live captures, not a fabricated-vs-real duplicate: the latter's response body's field set (AppRank/AppVersion/ApplicationSource/AssignedDeviceCount/AutoUpdateVersion/ContentGatewayId/etc.) matches InternalApplicationEntityV1's real shape and its metadata.description documents a full live capture (tools/internalapps-capture --use-begininstall, UEM 26.2.0.0, teardown-confirmed, anonymization listed) -- captured to verify that diagnostic tooling flag, not to replace the canonical fixture. The two never actually contend: begininstall-response.json sorts lexically before the ipa/ subdirectory ('b' < 'i'), so it deterministically wins TestChunkedUploadEndToEnd_BeginInstall's full-tree auto-load (internal_apps_chunked_upload_test.go) -- confirmed by that test's own assertion, which requires the Uuid to have an '11111111-' prefix, matching only begininstall-response.json. No test scopes a mock server to internal-apps-v1/ipa/ and also calls BeginInstall (the ipa-scoped lifecycle test in generated_internalappsv1_lifecycle_test.go calls CreateInternalApplication instead), so internal_app_begininstall_ipa.json is presently unexercised by any assertion -- inert, not broken.",

	{"/api/mdm/groups/{organizationGroupUuid}/baselines/{baselineUuid}/assignments", "GET", "1"}: "baselines/assignments-get.json is a synthetic placeholder (demoted by a prior integrity audit, loaded only via the bypass mechanism for TestGeneratedBaselinesV1Lifecycle); baselines/assignments_get_live.json is a genuine live capture against a persistent hand-created baseline, loaded via LoadResponseFromFile in the isolated TestGeneratedBaselinesV1AssignmentsGetLiveCapture, never in the same server instance as the placeholder.",

	{"/api/mdm/groups/{organizationGroupUuid}/baselines/{baselineUuid}/assignments", "POST", "1"}: "baselines/assignments-post.json is a synthetic placeholder (demoted by a prior integrity audit, loaded only via the bypass mechanism for TestGeneratedBaselinesV1Lifecycle); baselines/assignments_post_live.json is a genuine live capture, loaded via LoadResponseFromFile in the isolated TestGeneratedBaselinesV1AssignLiveCapture, never in the same server instance as the placeholder.",

	{"/api/mdm/groups/{organizationGroupUuid}/baselines/{baselineUuid}", "POST", "1"}: "the vendored baselines/clone-response.json is a synthetic placeholder (demoted by a prior integrity audit) with no functional consumer (deletion-control, 2026-09-24); TestGeneratedBaselinesV1Lifecycle loads a separate, non-identical copy from internal/mockserver/testdata/synthetic-placeholders/ via the bypass mechanism; baselines/clone_response_live.json is a genuine live capture proving a cross-organization-group clone (the clone lands under a disposable OG distinct from the source baseline's own OG), loaded via LoadResponseFromFile in the isolated TestGeneratedBaselinesV1CloneLiveCapture, never in the same server instance as the placeholder.",

	{"/api/mdm/groups/{organizationGroupUuid}/baselines/{baselineUuid}/devices", "GET", "1"}: "baselines/devices.json is a synthetic placeholder (demoted by a prior integrity audit, loaded only via the bypass mechanism for TestGeneratedBaselinesV1Lifecycle); baselines/devices_live.json is a genuine live capture -- the tenant has no enrolled devices, so a real, empty paged result -- loaded via LoadResponseFromFile in the isolated TestGeneratedBaselinesV1DevicesLiveCapture, never in the same server instance as the placeholder.",

	{"/api/mdm/devicesensors/list/{organizationGroupUuid}", "GET", "1"}: "sensors/list_sensors.json auto-loads (lexically first) and is a genuine empty-list capture (0 pre-existing sensors on the org group at capture time), consumed by TestGeneratedSensorLifecycleV1's post-create list assertion; sensors/list_sensors_populated.json is a genuine live capture against a persistent hand-populated organization group holding 2 real sensors, loaded via LoadResponseFromFile in the isolated TestGeneratedSensorsV1ListPopulatedLiveCapture, never in the same server instance as the empty fixture. This populated fixture is what exercises DeviceSensorResponseV1Model.QueryType's real x-flex-enum wire shape (a JSON number for a macOS/BASH sensor) -- the empty fixture never decodes a single sensor item.",

	{"/api/mdm/devicesensors/list/{organizationGroupUuid}", "GET", "2"}: "sensors/list_sensors_v2_empty.json auto-loads (lexically first) and is a genuine empty-list capture (0 pre-existing sensors on the org group at capture time), consumed by TestGeneratedSensorsV2List; sensors/list_sensors_v2_populated.json is a genuine live capture against the same persistent hand-populated organization group as the V1 populated fixture above, loaded via LoadResponseFromFile in the isolated TestGeneratedSensorsV2ListPopulatedLiveCapture, never in the same server instance as the empty fixture. Confirms the V2 response's query_type is genuinely a plain string on the wire (no flex-enum needed on this side).",
}
