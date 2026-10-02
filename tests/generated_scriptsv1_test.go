package tests

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/v26/internal/mdm/v1"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

const testScriptsOrgGroupUUID = "7d0aef64-735e-853e-1695-fdb07ba63f5c"

// TestGeneratedScriptsV1Lifecycle exercises the in-scope ScriptsV1Service
// surface against the testdata/mock-responses/scripts/ fixtures: List, Create
// (Pattern B 201 + Location), Get, Update (Pattern D-variant 204 + Location),
// BulkDelete success (200 + DeleteScriptResource).
//
// The partial-bulkdelete (206) and samples (204) paths run in separate
// tests because they need a different mock-server set than the auto-loaded
// fixtures: see TestGeneratedScriptsV1BulkDeletePartial below.
func TestGeneratedScriptsV1Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// create-response.json, update-response.json, and bulkdelete-success.json
	// are honestly-labeled synthetic placeholders with no live-tenant
	// equivalent to capture, so they are intentionally never migrated to the
	// vendored fixture set. 2026-08-06: relocated to
	// internal/mockserver/testdata/synthetic-placeholders/scripts/
	// (Phase-3-surviving location -- see that directory's README.md) and
	// loaded via the override-blind bypass path
	// (mockserver.LoadResponseFromFile) rather than the MOCK_RESPONSES_DIR-
	// aware directory scan. Previously these fixtures lived only under
	// testdata/mock-responses/, so this test was unaffected by a
	// MOCK_RESPONSES_DIR override for those three steps but would have
	// broken outright once repo-root testdata/ is removed (Phase 3); now
	// relocated, it is genuinely unaffected by BOTH a vendored-branch
	// override AND testdata/ removal. list.json and get.json (used below)
	// are separate fixtures unaffected by this change and keep loading via
	// the normal directory scan.
	ms := newMockServerWithBypassFixtures(t, "../testdata/mock-responses", []string{
		"../internal/mockserver/testdata/synthetic-placeholders/scripts/create-response.json",
		"../internal/mockserver/testdata/synthetic-placeholders/scripts/update-response.json",
		"../internal/mockserver/testdata/synthetic-placeholders/scripts/bulkdelete-success.json",
	})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewScriptsV1Service(c)
	ctx := context.Background()

	// LIST — GET /api/mdm/groups/{ogUuid}/scripts → 200 + ScriptsSearchResult.
	// Codegen-generated Options struct is required (validated as non-nil at
	// the top of the generated method); Expand defaults to false.
	listOpts := &sdk.ScriptsV1GetScriptsByOrganizationGroupAsyncOptions{}
	_, list, err := svc.GetScriptsByOrganizationGroupAsync(ctx, testScriptsOrgGroupUUID, listOpts)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil list result")
	}
	if list.RecordCount == nil || *list.RecordCount != 2 {
		got := -1
		if list.RecordCount != nil {
			got = *list.RecordCount
		}
		t.Fatalf("expected RecordCount=2, got %d", got)
	}
	if len(list.SearchResults) != 2 {
		t.Fatalf("expected 2 SearchResults, got %d", len(list.SearchResults))
	}
	// Live-captured against the real test tenant:
	// list.json now holds a genuine 2-script capture, replacing a fabricated
	// fixture. get.json (used by the GET step below) is a separate,
	// untouched fixture, so it keeps its own pre-existing values.
	if list.SearchResults[0].Name != "Test Koaladid" {
		t.Errorf("expected first script name 'Test Koaladid', got %q", list.SearchResults[0].Name)
	}
	if list.SearchResults[0].Platform != "APPLE_OSX" {
		t.Errorf("expected first script Platform 'APPLE_OSX' (string-enum), got %q", list.SearchResults[0].Platform)
	}

	// CREATE — POST /api/mdm/groups/{ogUuid}/scripts → 201 + Location, empty body.
	// Pattern B: Location header carries the new script's URL; ParseLocationID
	// extracts the trailing path segment (the UUID).
	createReq := &sdk.CreateScriptV1{
		Name:       "Sample script 1",
		Platform:   "WIN_RT",
		ScriptType: "POWERSHELL",
		ScriptData: "R2V0LURhdGUgfCBPdXQtRmlsZSBDOlx0ZXN0LnR4dA==",
	}
	createHeaders, err := svc.CreateScriptAsync(ctx, testScriptsOrgGroupUUID, createReq)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	scriptUUID, err := client.ParseLocationID(createHeaders.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location header failed: %v", err)
	}
	if scriptUUID == "" {
		t.Fatal("scriptUUID missing; fixture must include server-hydrated Location header value")
	}

	// GET — GET /api/mdm/scripts/{scriptUuid} → 200 + ScriptResource.
	_, got, err := svc.GetScriptAsync(ctx, scriptUUID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil get response")
	}
	if got.Name != "Sample script 1" {
		t.Errorf("expected name 'Sample script 1', got %q", got.Name)
	}
	if got.Platform != "WIN_RT" {
		t.Errorf("expected ScriptResource.Platform 'WIN_RT' (string-enum), got %q", got.Platform)
	}
	if got.ScriptData != "R2V0LURhdGUgfCBPdXQtRmlsZSBDOlx0ZXN0LnR4dA==" {
		t.Errorf("expected base64 script_data round-trip; got %q", got.ScriptData)
	}

	// UPDATE — PUT /api/mdm/scripts/{scriptUuid} → 204 + Location (canonical
	// impl returns Location even on 204; verified by the fixture).
	updateReq := &sdk.UpdateScriptV1{
		Name:       "Sample script 1 (renamed)",
		Platform:   "WIN_RT",
		ScriptType: "POWERSHELL",
		ScriptData: "R2V0LURhdGUgfCBPdXQtRmlsZSBDOlx0ZXN0LnR4dA==",
	}
	updateHeaders, err := svc.ReplaceScriptDefinitionAsync(ctx, scriptUUID, updateReq)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if loc := updateHeaders.Get("Location"); loc == "" {
		t.Fatal("expected Location header on 204 update response per canonical impl, got empty")
	}

	// BULKDELETE success — POST /api/mdm/groups/{ogUuid}/scripts/bulkdelete →
	// 200 + DeleteScriptResource. Codegen takes *[]string (pointer-to-slice).
	uuids := []string{scriptUUID, "05d17100-b346-c29d-6760-a0fdedcf8623"}
	_, deleteResult, err := svc.ScriptBulkDeleteAsync(ctx, testScriptsOrgGroupUUID, &uuids)
	if err != nil {
		t.Fatalf("bulkdelete failed: %v", err)
	}
	if deleteResult == nil {
		t.Fatal("expected non-nil DeleteScriptResource on 200")
	}
	if deleteResult.ScriptsDeleted == nil || *deleteResult.ScriptsDeleted != 1 {
		got := -1
		if deleteResult.ScriptsDeleted != nil {
			got = *deleteResult.ScriptsDeleted
		}
		t.Errorf("expected ScriptsDeleted=1, got %d", got)
	}
	if len(deleteResult.DeleteFailed) != 0 {
		t.Errorf("expected no failed deletes, got %+v", deleteResult.DeleteFailed)
	}
}

// TestGeneratedScriptsV1BulkDeletePartial exercises the 206-partial path.
// The lifecycle test above auto-loads bulkdelete-success.json (the directory
// scan picks one fixture per (method, endpoint, version) tuple). To exercise
// the partial-failure fixture, build a focused mock-server with just that
// single fixture loaded.
func TestGeneratedScriptsV1BulkDeletePartial(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Note: file is suffixed -zz- so the lexical-order auto-load (LoadMockResponses)
	// picks bulkdelete-success.json over this one for the lifecycle test; this
	// partial-only test loads it directly via LoadResponseFromFile. 2026-08-06:
	// relocated to internal/mockserver/testdata/synthetic-placeholders/scripts/
	// (Phase-3-surviving location -- see that directory's README.md), so this
	// direct-load test survives a future repo-root `git rm testdata/` as well
	// as a MOCK_RESPONSES_DIR override.
	resp, err := mockserver.LoadResponseFromFile("../internal/mockserver/testdata/synthetic-placeholders/scripts/bulkdelete-zz-partial.json")
	if err != nil {
		t.Fatalf("load partial fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewScriptsV1Service(c)
	ctx := context.Background()

	uuids := []string{
		"bafde89c-041e-1756-082b-933aaf16cad8",
		"2fea58d4-8d07-f072-ca6b-44a64414f081",
	}
	_, deleteResult, err := svc.ScriptBulkDeleteAsync(ctx, testScriptsOrgGroupUUID, &uuids)
	if err != nil {
		t.Fatalf("bulkdelete-partial failed: %v", err)
	}
	if deleteResult == nil {
		t.Fatal("expected non-nil DeleteScriptResource on 206")
	}
	if deleteResult.ScriptsDeleted == nil || *deleteResult.ScriptsDeleted != 1 {
		got := -1
		if deleteResult.ScriptsDeleted != nil {
			got = *deleteResult.ScriptsDeleted
		}
		t.Errorf("expected ScriptsDeleted=1 on partial, got %d", got)
	}
	if len(deleteResult.DeleteFailed) != 1 {
		t.Errorf("expected 1 entry in DeleteFailed (List<Guid> per canonical), got %d", len(deleteResult.DeleteFailed))
	} else if deleteResult.DeleteFailed[0] != "2fea58d4-8d07-f072-ca6b-44a64414f081" {
		t.Errorf("expected failed UUID 'deadbeef-...', got %q", deleteResult.DeleteFailed[0])
	}
}

// TestGeneratedScriptsV1Samples exercises POST /scripts/samples. Canonical
// impl returns 204 empty; the codegen-emitted method signature returns
// (http.Header, error) with no body parameter since the response carries
// none. Plan D6 corrected the swagger's erroneous 200 + ScriptSampleSearchResponseModel.
//
// The samples-response.json fixture is a genuine live capture (vendored 4f0dd5fe5,
// 2026-09-24: a real enrolled AppleOsX device with an APPLE_OSX script,
// 204 No Content) and backs a verified[] ledger row. It is loaded via the
// bypass mechanism (newMockServerWithBypassFixtures, see
// tests/bypass_fixture_helper_test.go) from its byte-identical twin at
// internal/mockserver/testdata/synthetic-placeholders/scripts/
// samples-response.json, which the twin guard keeps identical to the
// vendored fixture. The body is empty, so the test checks only err == nil.
func TestGeneratedScriptsV1Samples(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := newMockServerWithBypassFixtures(t, "../testdata/mock-responses", []string{
		"../internal/mockserver/testdata/synthetic-placeholders/scripts/samples-response.json",
	})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewScriptsV1Service(c)
	ctx := context.Background()

	samplesReq := &sdk.ScriptSampleSearchRequestModelV1{}
	_, err := svc.GetScriptSamplesAsync(ctx, testScriptsOrgGroupUUID, samplesReq)
	if err != nil {
		t.Fatalf("samples request failed: %v", err)
	}
}

// TestGeneratedScriptsV1GetLiveCapture exercises GetScriptAsync against a
// genuine live-captured response (scripts/get_live.json), loaded via an
// isolated single-fixture mock server. This fixture shares its exact
// endpoint, method, and version with the pre-existing scripts/get.json
// fixture (both are a bare GET /api/mdm/scripts/{scriptUuid} template with
// no distinguishing query params), so loading the full
// testdata/mock-responses corpus together resolves the tie in favor of the
// alphabetically-first-loaded get.json -- same collision class as
// TestGeneratedUpdatesV1CreateConflict. The get.json fixture is deliberately left at its
// original content (consumed by TestGeneratedScriptsV1Lifecycle and the
// CatalogDisplay overlay regression guard, TestGeneratedScriptResourceCatalogDisplay,
// both of which depend on its specific hand-authored shape), so the genuine
// capture lives under its own filename and is exercised in isolation here.
func TestGeneratedScriptsV1GetLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/scripts/get_live.json"))
	if err != nil {
		t.Fatalf("load get_live fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewScriptsV1Service(c)
	ctx := context.Background()

	_, got, err := svc.GetScriptAsync(ctx, "64c7d5c5-ac02-505f-9131-99ed7bf4b7f7")
	if err != nil {
		t.Fatalf("GetScriptAsync failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil get response")
	}
	if got.Name != "Test Hamsterbuy" {
		t.Errorf("expected name 'Test Hamsterbuy', got %q", got.Name)
	}
	if got.Platform != "APPLE_OSX" {
		t.Errorf("expected Platform 'APPLE_OSX' (string-enum), got %q", got.Platform)
	}
	if got.ScriptData != "eGZneGc=" {
		t.Errorf("expected base64 script_data round-trip; got %q", got.ScriptData)
	}
}

// TestCreateScriptV1_AllowedInCatalogAlwaysOnWire is a pure marshal test (no
// mock server): CreateScript is swagger-required for allowed_in_catalog live,
// even though upstream swagger itself declares no required array at all for
// this schema. With the (unset -> zero value, dropped-by-omitempty) shape a
// caller who leaves AllowedInCatalog unset would silently omit the key and
// the server would reject the request with "Required property
// 'allowed_in_catalog' not found in JSON". Asserts the key is present on the
// wire even at its zero value, false.
func TestCreateScriptV1_AllowedInCatalogAlwaysOnWire(t *testing.T) {
	body, err := json.Marshal(mdmv1.CreateScriptV1{})
	if err != nil {
		t.Fatalf("json.Marshal(CreateScriptV1{}): %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	v, present := decoded["allowed_in_catalog"]
	if !present {
		t.Fatalf("expected 'allowed_in_catalog' key present on the wire even when unset; got %s", body)
	}
	if v != false {
		t.Errorf("expected zero-value allowed_in_catalog to serialize as false, got %v", v)
	}
}
