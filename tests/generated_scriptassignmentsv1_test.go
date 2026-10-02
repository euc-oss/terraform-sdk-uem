package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestGeneratedScriptAssignmentsV1Lifecycle exercises the four
// ScriptAssignmentV1Controller endpoints against the
// testdata/mock-responses/scripts-assignments/ fixtures.
func TestGeneratedScriptAssignmentsV1Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// All four fixtures this test depends on (list.json, create-response.json,
	// get.json, bulkupdate-response.json) are honestly-labeled synthetic
	// placeholders with no live-tenant equivalent to capture, so they are
	// intentionally never migrated to the vendored fixture set. 2026-08-06:
	// relocated to internal/mockserver/testdata/synthetic-placeholders/
	// scripts-assignments/ (Phase-3-surviving location -- see that
	// directory's README.md) and loaded via the override-blind bypass path
	// (mockserver.LoadResponseFromFile) rather than the MOCK_RESPONSES_DIR-
	// aware directory scan. Previously these fixtures lived only under
	// testdata/mock-responses/, so this test was self-contained against a
	// MOCK_RESPONSES_DIR override but would have broken outright once
	// repo-root testdata/ is removed (Phase 3); now relocated, it is
	// genuinely self-contained against BOTH a vendored-branch override AND
	// testdata/ removal.
	ms := newMockServerWithBypassFixtures(t, "../testdata/mock-responses", []string{
		"../internal/mockserver/testdata/synthetic-placeholders/scripts-assignments/list.json",
		"../internal/mockserver/testdata/synthetic-placeholders/scripts-assignments/create-response.json",
		"../internal/mockserver/testdata/synthetic-placeholders/scripts-assignments/get.json",
		"../internal/mockserver/testdata/synthetic-placeholders/scripts-assignments/bulkupdate-response.json",
	})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewScriptAssignmentV1Service(c)
	ctx := context.Background()

	scriptUUID := "bafde89c-041e-1756-082b-933aaf16cad8"

	// LIST — GET /api/mdm/scripts/{scriptUuid}/assignments → 200 +
	// ScriptAssignmentsSearchResult ({SearchResults+RecordCount} wrapper).
	// list.json is a synthetic placeholder (see its own metadata.description):
	// no live-tenant capture exists for this endpoint. It now asserts an
	// empty result set (RecordCount=0, no SearchResults), matching the
	// fixture's real recaptured content.
	_, list, err := svc.GetScriptAssignmentsAsync(ctx, scriptUUID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil list result")
	}
	if list.RecordCount == nil || *list.RecordCount != 0 {
		got := -1
		if list.RecordCount != nil {
			got = *list.RecordCount
		}
		t.Fatalf("expected RecordCount=0, got %d", got)
	}
	if len(list.SearchResults) != 0 {
		t.Fatalf("expected 0 SearchResults, got %d", len(list.SearchResults))
	}

	// CREATE — POST /api/mdm/scripts/{scriptUuid}/assignments → 201 +
	// Location (Pattern B). Action is marked [Obsolete] in canonical;
	// callers should prefer BulkUpdate but the route remains live.
	createReq := &sdk.CreateScriptAssignmentV1{
		Name:           "Sample assignment 1",
		DeploymentMode: "AUTO",
		Memberships: []sdk.SmartGroupDataV1{
			{
				SmartGroupName: "All Devices",
				SmartGroupUUID: "eec458d3-722f-678a-52b9-22398b02009e",
			},
		},
	}
	createHeaders, err := svc.AddScriptAssignmentAsync(ctx, scriptUUID, createReq)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	assignmentUUID, err := client.ParseLocationID(createHeaders.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location header failed: %v", err)
	}
	if assignmentUUID == "" {
		t.Fatal("assignmentUUID missing; fixture must include server-hydrated Location header value")
	}

	// GET — GET /api/mdm/scriptassignments/{assignmentUuid} → 200 +
	// ScriptAssignmentResource. get.json is a real recaptured fixture (not a
	// synthetic placeholder tied to the CREATE step above), so it does not
	// round-trip the CREATE response's assignmentUUID/name; assert against
	// its own real content instead.
	_, got, err := svc.GetScriptAssignmentAsync(ctx, assignmentUUID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil get response")
	}
	if got.AssignmentUUID == "" {
		t.Error("expected non-empty AssignmentUUID")
	}
	if got.Name != "Test Koaladid" {
		t.Errorf("expected name 'Test Koaladid', got %q", got.Name)
	}

	// BULKUPDATE — POST /api/mdm/scripts/{scriptUuid}/updateassignments →
	// 204 empty (Pattern D). Assignments is required (swagger
	// required: ["assignments"]); populate one element mirroring the
	// CREATE assignment above (same script, deployment mode, and smart
	// group membership) so this exercises a realistic "re-assert the
	// same assignment" bulk-update rather than a vacuous empty slice.
	bulkReq := &sdk.BulkUpdateScriptAssignmentV1{
		Assignments: []sdk.BaseScriptAssignmentV1{
			{
				ScripUUID:      scriptUUID,
				AssignmentUUID: assignmentUUID,
				Name:           "Sample assignment 1",
				DeploymentMode: "AUTO",
				Memberships: []sdk.SmartGroupDataV1{
					{
						SmartGroupName: "All Devices",
						SmartGroupUUID: "eec458d3-722f-678a-52b9-22398b02009e",
					},
				},
			},
		},
	}
	_, err = svc.BulkUpdateScriptAssignmentsAsync(ctx, scriptUUID, bulkReq)
	if err != nil {
		t.Fatalf("bulkupdate failed: %v", err)
	}
}
