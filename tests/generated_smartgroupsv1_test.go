package tests

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
	"github.com/euc-oss/terraform-sdk-uem/internal/orgtest"
)

// testSmartGroupID is the ID used across the SmartGroups lifecycle test.
// The mockserver matches by path template ({id}), not by exact ID value,
// so this is arbitrary — it does NOT need to match the real ID used during
// live capture (which was anonymized out of the fixtures anyway).
const testSmartGroupID = 450125

// TestGeneratedSmartGroupsV1Lifecycle exercises the live-captured (and
// anonymized) create/get/update/delete SmartGroups V1 lifecycle against
// UEM 26.2.0.0 fixtures in testdata/mock-responses/smartgroups/:
//   - CREATE  POST /api/mdm/smartgroups           → 201 {Value, uuid}
//   - GET     GET  /api/mdm/smartgroups/{id}       → 200, immediately-after-create shape
//   - UPDATE  PUT  /api/mdm/smartgroups/{id}       → 204 empty
//   - DELETE  DELETE /api/mdm/smartgroups/{id}     → 204 empty
//
// NOTE on fixture collisions: smartgroups_get_after_create.json,
// smartgroups_get_after_update.json, and smartgroups_get_after_delete.json
// all share the exact same endpoint/method (GET /api/mdm/smartgroups/{id})
// with no distinguishing query params, so a full-corpus mock server load
// always resolves a plain GET to the alphabetically-first fixture
// (smartgroups_get_after_create.json — "create" < "delete" < "update").
// This mirrors the identical collision already documented and worked
// around for UpdatesV1 in TestGeneratedUpdatesV1CreateConflict. The
// after-update and after-delete shapes are therefore verified in their own
// isolated-mock-server tests below
// (TestGeneratedSmartGroupsV1GetAfterUpdate,
// TestGeneratedSmartGroupsV1GetAfterDelete) rather than in this test.
func TestGeneratedSmartGroupsV1Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewSmartGroupsService(c)
	ctx := context.Background()

	// CREATE — POST /api/mdm/smartgroups → 201 {Value, uuid}.
	// Fixture: smartgroups/smartgroups_create.json
	createReq := &sdk.SmartGroupEditV1Model{
		Name:                         "Test SmartGroup",
		CriteriaType:                 "All",
		ManagedByOrganizationGroupID: "12345",
	}
	_, created, err := svc.CreateSmartGroupAsync(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateSmartGroupAsync failed: %v", err)
	}
	if created == nil || created.Value == nil || *created.Value != 12346 {
		t.Fatalf("expected created Value=12346, got %+v", created)
	}
	// Regression guard for internal-source/mdmv1.smartgroups.overlay.yaml:
	// EntityId (the un-patched swagger response schema) only declares Value.
	// The real wire response also includes uuid, which the overlay's
	// scoped SmartGroupCreateResponseV1 gap-fill schema adds back. If the
	// overlay entry regresses, this field disappears from the generated
	// struct and the line below fails to compile -- the same "compile
	// failure is the proof" pattern used in
	// generated_deviceprofilev2_response_fields_test.go.
	var wantCreate struct {
		UUID string `json:"uuid"`
	}
	loadFixtureResponseBodyForTest(t, "smartgroups/smartgroups_create.json", &wantCreate)
	if created.UUID != wantCreate.UUID {
		t.Errorf("uuid: got %q, want %q -- was it dropped from SmartGroupCreateResponseV1?", created.UUID, wantCreate.UUID)
	}

	// GET — GET /api/mdm/smartgroups/{id} → 200, immediately-after-create shape.
	// Fixture: smartgroups/smartgroups_get_after_create.json
	_, got, err := svc.LoadSmartGroupAsync(ctx, testSmartGroupID)
	if err != nil {
		t.Fatalf("LoadSmartGroupAsync failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil smart group")
	}
	if got.Name == "" {
		t.Error("expected non-empty name (anonymized fixture content)")
	}
	if got.CriteriaType != "All" {
		t.Errorf("criteriaType: got %q, want %q", got.CriteriaType, "All")
	}
	// Real-wire finding: creating a smart group with an empty Ownerships
	// list does not stay empty on the wire — the server defaults it to a
	// single-element ["allownerships"] list. Confirmed live, 2026-07-30.
	if len(got.Ownerships) != 1 || got.Ownerships[0] != "allownerships" {
		t.Errorf("ownerships: got %v, want [allownerships] (server-defaulted, not what was sent)", got.Ownerships)
	}

	// UPDATE — PUT /api/mdm/smartgroups/{id} → 204 empty. Confirmed to
	// persist on the wire via a live re-GET (see
	// smartgroups_get_after_update.json and
	// TestGeneratedSmartGroupsV1GetAfterUpdate below) — Name really changed
	// on the live capture, not a silent no-op.
	// Fixture: smartgroups/smartgroups_update.json
	updateReq := &sdk.SmartGroupEditV1Model{
		Name:                         "Test SmartGroup Renamed",
		CriteriaType:                 "All",
		ManagedByOrganizationGroupID: "12345",
	}
	if _, err := svc.UpdateSmartGroupAsync(ctx, testSmartGroupID, updateReq); err != nil {
		t.Fatalf("UpdateSmartGroupAsync failed: %v", err)
	}

	// DELETE — DELETE /api/mdm/smartgroups/{id} → 204 empty. Confirmed
	// removed on the wire via a live re-GET that returned a not-found error
	// (see smartgroups_get_after_delete.json and
	// TestGeneratedSmartGroupsV1GetAfterDelete below).
	// Fixture: smartgroups/smartgroups_delete.json
	if _, err := svc.DeleteAsync(ctx, testSmartGroupID); err != nil {
		t.Fatalf("DeleteAsync failed: %v", err)
	}
}

// TestGeneratedSmartGroupsV1GetAfterUpdate verifies the LoadSmartGroupAsync
// response shape immediately after UpdateSmartGroupAsync — an isolated
// single-fixture mock server is required because
// smartgroups_get_after_update.json is shadowed by
// smartgroups_get_after_create.json in a full-corpus load (see the
// collision note on TestGeneratedSmartGroupsV1Lifecycle above).
func TestGeneratedSmartGroupsV1GetAfterUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/smartgroups/smartgroups_get_after_update.json"))
	if err != nil {
		t.Fatalf("load get_after_update fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewSmartGroupsService(c)
	ctx := context.Background()

	_, got, err := svc.LoadSmartGroupAsync(ctx, testSmartGroupID)
	if err != nil {
		t.Fatalf("LoadSmartGroupAsync failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil smart group")
	}
	if got.Name == "" {
		t.Error("expected non-empty post-update name (anonymized fixture content)")
	}
}

// TestGeneratedSmartGroupsV1GetAfterDelete verifies the LoadSmartGroupAsync
// error path immediately after DeleteAsync — an isolated single-fixture
// mock server is required for the same shadowing reason as
// TestGeneratedSmartGroupsV1GetAfterUpdate above.
//
// Real-wire finding: the live API does NOT return HTTP 404 for a
// deleted/missing smart group. It returns HTTP 400 with errorCode 6 and a
// descriptive "Smart Group not found with Id: ..." message. This test
// confirms the generated client surfaces that 400/errorCode:6 shape as an
// error rather than a 404.
func TestGeneratedSmartGroupsV1GetAfterDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/smartgroups/smartgroups_get_after_delete.json"))
	if err != nil {
		t.Fatalf("load get_after_delete fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewSmartGroupsService(c)
	ctx := context.Background()

	_, _, err = svc.LoadSmartGroupAsync(ctx, testSmartGroupID)
	if err == nil {
		t.Fatal("expected error for deleted smart group, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to mention status 400, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
}

// TestLive_SmartGroupCreate_FullLiveWithOGContainment hand-writes the
// full-live-with-OG-containment tier for SmartGroups V1 create (hand-written,
// not template-generated; see .claude/rules/api-doctrine.md and
// testdata/live-policy.yaml's SmartGroups entry). Posts the live-validated
// testdata/request-templates/smartgroup-create.json body against a
// disposable child OG created under liveOrgGroupID(t), never against the
// shared fixed OG directly. Gated by skipUnlessLiveMutate: never runs
// under `go test -short` or a plain TEST_MODE=live run -- only
// TEST_MODE=live AND LIVE_MUTATE=1 together.
func TestLive_SmartGroupCreate_FullLiveWithOGContainment(t *testing.T) {
	skipUnlessLiveMutate(t)
	c := getTestClient(t, "basic")
	ctx := context.Background()
	ogID := orgtest.NewDisposableOrgGroupOrFatal(t, c, liveOrgGroupID(t), nil)

	data, err := os.ReadFile("../testdata/request-templates/smartgroup-create.json")
	if err != nil {
		t.Fatalf("read request template: %v", err)
	}
	var req sdk.SmartGroupEditV1Model
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal request template: %v", err)
	}
	req.ManagedByOrganizationGroupID = strconv.Itoa(ogID) // override to the disposable OG

	svc := sdk.NewSmartGroupsService(c)
	_, resp, err := svc.CreateSmartGroupAsync(ctx, &req)
	if err != nil {
		t.Fatalf("CreateSmartGroupAsync: %v", err)
	}
	if resp == nil || resp.Value == nil {
		t.Fatal("CreateSmartGroupAsync response had no Value (id) -- cannot register cleanup, refusing to leave the resource untracked")
	}
	smartGroupID := int(*resp.Value)
	// Registered AFTER the OG create above, so LIFO fires this delete
	// BEFORE the OG's own delete cleanup -- do not depend on OG-delete
	// cascading to child resources (unverified for SmartGroups).
	t.Cleanup(func() {
		if _, err := svc.DeleteAsync(context.Background(), smartGroupID); err != nil {
			t.Logf("cleanup: SmartGroupsService.DeleteAsync(%d) failed: %v", smartGroupID, err)
		}
	})
}
