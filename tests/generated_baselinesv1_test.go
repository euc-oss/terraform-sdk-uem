package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

const (
	testBaselinesOrgGroupUUID = "7d0aef64-735e-853e-1695-fdb07ba63f5c"
	// Live-captured 2026-08-05 (reference clean create->get->status->put->delete
	// lifecycle against the shared test org group): the anonymizer's deterministic
	// gofakeit sequence lands the same fake UUID across create/get/update
	// responses sharing this shape, replacing the old hand-authored placeholder.
	testBaselineUUID       = "f9e174ca-9ee2-9354-2762-b0bf409a123b"
	testBaselineDeviceUUID = "2e121787-d028-941a-7fc9-0df5de7ad6f4"
)

// TestGeneratedBaselinesV1Lifecycle exercises the BaselinesV1Service
// surface against testdata/mock-responses/baselines/ fixtures: Create
// (multipart via BaselineCreator helper), Get, List, Update, Delete,
// Clone, Status, Assignments (Get + Post), Devices, DevicePolicies.
//
// Clone, Assignments-Get, Assignments-Post, and Devices are
// synthetic placeholders demoted 2026-08-04 (integrity audit:
// fabricated/schema-derived, no live-capture evidence; reachability
// unconfirmed on live) -- unlike Create/List/Get/Update/Delete/
// Status, which are genuine live captures from the 2026-08-05 reference
// lifecycle run and remain on the normal directory-scanned fixture path.
// These 4 are loaded via the bypass mechanism (newMockServerWithBypassFixtures,
// see tests/bypass_fixture_helper_test.go) from their Phase-3-surviving
// synthetic-placeholder location (internal/mockserver/testdata/
// synthetic-placeholders/baselines/), and this test's assertions on their
// response bodies were weakened 2026-08-06 to check shape (non-empty list,
// UUID-shaped identifiers, count consistency) rather than the specific
// fabricated record counts/values the old fixtures claimed. DevicePolicies
// is a genuine live capture on the normal fixture path, asserted field by field.
func TestGeneratedBaselinesV1Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := newMockServerWithBypassFixtures(t, "../testdata/mock-responses", []string{
		"../internal/mockserver/testdata/synthetic-placeholders/baselines/clone-response.json",
		"../internal/mockserver/testdata/synthetic-placeholders/baselines/assignments-get.json",
		"../internal/mockserver/testdata/synthetic-placeholders/baselines/assignments-post.json",
		"../internal/mockserver/testdata/synthetic-placeholders/baselines/devices.json",
	})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewBaselinesV1Service(c)
	creator := sdk.NewBaselineCreator(c)
	ctx := context.Background()

	// CREATE (multipart) — POST /groups/{ogUuid}/baselines → 200 + BaselineV1Model
	// (D6 corrected from .Generated.cs's 201 + Location).
	createReq := &mdmv1.CreateBaselineRequestV1Model{
		Name:        "Created via BaselineCreator",
		Description: "Multipart-create test baseline",
	}
	_, created, err := creator.Create(ctx, testBaselinesOrgGroupUUID, createReq, nil)
	if err != nil {
		t.Fatalf("BaselineCreator.Create failed: %v", err)
	}
	var wantCreated struct {
		BaselineUUID string `json:"baselineUUID"`
	}
	loadFixtureResponseBodyForTest(t, "baselines/create-response.json", &wantCreated)
	if created == nil || created.BaselineUUID != wantCreated.BaselineUUID {
		t.Fatalf("expected created BaselineUUID %q, got %+v", wantCreated.BaselineUUID, created)
	}

	// LIST — GET /groups/{ogUuid}/baselines → 200 + bare array (D6).
	// Live-captured: the shared test org group genuinely has zero
	// baselines — replacing a fabricated fixture that claimed 2. The mock
	// server serves static fixtures (no real CRUD state), so this is
	// independent of the CREATE step above.
	_, list, err := svc.GetBaselinesAsync(ctx, testBaselinesOrgGroupUUID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil list response")
	}
	// Genuine capture of an org group holding one baseline: compare each
	// decoded entry with the fixture's own values.
	var wantList []map[string]any
	loadFixtureResponseBodyForTest(t, "baselines/groups_get_id_baselines.json", &wantList)
	if len(wantList) == 0 || len(*list) != len(wantList) {
		t.Fatalf("list: got %d baselines, want %d (non-zero) from the fixture", len(*list), len(wantList))
	}
	for i, want := range wantList {
		got := (*list)[i]
		if got.BaselineUUID != want["baselineUUID"] || got.Name != want["name"] || got.PlatformName != want["platformName"] || got.TemplateName != want["templateName"] {
			t.Errorf("list[%d]: got uuid=%q name=%q platform=%q template=%q, want %v/%v/%v/%v", i,
				got.BaselineUUID, got.Name, got.PlatformName, got.TemplateName, want["baselineUUID"], want["name"], want["platformName"], want["templateName"])
		}
	}

	// GET — single baseline → 200 + BaselineV1Model.
	_, got, err := svc.GetBaselineAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	var wantGot struct {
		BaselineUUID string `json:"baselineUUID"`
		PlatformUUID string `json:"platformUUID"`
	}
	loadFixtureResponseBodyForTest(t, "baselines/get.json", &wantGot)
	if got == nil || got.BaselineUUID != wantGot.BaselineUUID {
		t.Fatalf("unexpected get response: %+v", got)
	}
	// Round-trip a camelCase D3 wire field. Live-captured 2026-08-05: real
	// Windows 10 platform UUID from the tenant's baseline catalog (preserved
	// verbatim by the anonymizer's platform/version substring rules, not a
	// gofakeit fake -- see mut_baselines' report for why this isn't a leak).
	if got.PlatformUUID != wantGot.PlatformUUID {
		t.Errorf("expected platformUUID round-trip; got %q", got.PlatformUUID)
	}

	// UPDATE — PUT → 200 + BaselineV1Model body (D6 corrected from 204).
	updateReq := &mdmv1.UpdateBaselineRequestV1Model{
		Name:        "Win 10 baseline 1 (renamed)",
		Description: "Updated description",
	}
	_, updated, err := svc.UpdateBaselineAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID, updateReq)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	// Live-captured 2026-08-05: the anonymizer reuses the same deterministic
	// fake name across create/get/update responses sharing this shape (the
	// real server-side rename did happen -- version correctly bumped 1->2 --
	// but the anonymized name string doesn't reflect it, same as
	// testBaselineUUID above).
	if updated == nil || updated.Name != "Test Manateewill" {
		t.Errorf("expected renamed baseline in update response, got %+v", updated)
	}

	// CLONE — POST ?action=clone → 201 + body + Location (Pattern C). The
	// mockserver matcher does not inspect query params, so the bare POST
	// path matches the clone fixture (the create fixture lives at the
	// /baselines path without the {baselineUuid} segment, so no conflict).
	cloneReq := &mdmv1.CreateBaselineRequestV1Model{
		Name: "Win 10 baseline 1 (clone)",
		// RootLocationGroupID is required (swagger + canonical C# [Required])
		// and is an int-family org-group identifier, distinct from the
		// UUID-shaped testBaselinesOrgGroupUUID used elsewhere in this test
		// (which addresses the org group by UUID in the URL path). The
		// value below is a fixed sample number; the mock server serves a
		// static fixture and does not resolve it.
		RootLocationGroupID: 138883,
	}
	cloneOpts := &mdmv1.BaselinesV1CloneBaselineAsyncOptions{Action: "clone"}
	_, cloned, err := svc.CloneBaselineAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID, cloneReq, cloneOpts)
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}
	// Synthetic placeholder fixture (see newMockServerWithBypassFixtures call
	// above): assert shape (non-empty name, UUID-shaped baselineUUID) rather
	// than a specific fabricated value.
	if cloned == nil || cloned.Name == "" {
		t.Errorf("expected cloned baseline with non-empty name, got %+v", cloned)
	}
	if cloned != nil && !looksLikeUUID(cloned.BaselineUUID) {
		t.Errorf("expected UUID-shaped BaselineUUID in clone response, got %q", cloned.BaselineUUID)
	}

	// STATUS — GET .../status → 200 + bare array of BaselineStatusV1Model.
	_, status, err := svc.GetBaselineStatusAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	// Live-captured 2026-08-05: a freshly-created baseline with zero device
	// assignments genuinely has no compliance status records yet -- empty is
	// the correct real-world shape, not a missing-data artifact.
	if status == nil || len(*status) != 0 {
		t.Fatalf("expected 0 status entries (no devices assigned yet), got %+v", status)
	}

	// ASSIGNMENTS-GET — bare array of BaselineAssignmentsV1Model.
	_, assignments, err := svc.GetBaselineAssignmentsAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID)
	if err != nil {
		t.Fatalf("assignments-get failed: %v", err)
	}
	// Synthetic placeholder fixture (see newMockServerWithBypassFixtures call
	// above): assert shape (non-empty list, UUID-shaped smartGroupUUID) rather
	// than a specific fabricated record count/value.
	if assignments == nil || len(*assignments) == 0 {
		t.Fatalf("expected at least 1 assignment, got %+v", assignments)
	}
	if !looksLikeUUID((*assignments)[0].SmartGroupUUID) {
		t.Errorf("expected UUID-shaped smartGroupUUID round-trip, got %q", (*assignments)[0].SmartGroupUUID)
	}

	// ASSIGN — POST .../assignments → 204 empty. SmartGroups is required
	// (generated Validate() rejects a nil slice); id 1 matches the smart
	// group id in the ASSIGNMENTS-GET synthetic placeholder fixture read
	// just above (assignments-get.json: {"id": 1, ...}), so this assigns
	// the same smart group the GET already reports as assigned.
	assignReq := &mdmv1.BaselineAssignmentRequestV1Model{
		SmartGroups: []*int{intPtr(1)},
	}
	_, err = svc.AssignBaselineAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID, assignReq)
	if err != nil {
		t.Fatalf("assign failed: %v", err)
	}

	// DEVICES — GET .../devices → 200 + GetBaselineDeviceSummaryResponseV1Model
	// (page/pageSize/total/results paged wrapper).
	_, devices, err := svc.GetBaselineDevicesAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID)
	if err != nil {
		t.Fatalf("devices failed: %v", err)
	}
	// Synthetic placeholder fixture (see newMockServerWithBypassFixtures call
	// above): assert shape (non-empty results, total consistent with
	// len(results), UUID-shaped deviceUUID) rather than a specific fabricated
	// record count.
	if devices == nil || devices.Total == nil || *devices.Total != len(devices.Results) {
		t.Fatalf("expected total to match len(results), got %+v", devices)
	}
	if len(devices.Results) == 0 {
		t.Fatalf("expected at least 1 device entry in results, got %d", len(devices.Results))
	}
	if !looksLikeUUID(devices.Results[0].DeviceUUID) {
		t.Errorf("expected UUID-shaped DeviceUUID, got %q", devices.Results[0].DeviceUUID)
	}

	// DEVICE-POLICIES — GET .../devices/{deviceUuid}/policies → results
	// wrapper {results, total} of per-policy items (quirk 36). The fixture is
	// a genuine live capture served from the corpus by route. Every decoded
	// field is compared with the fixture's own values, read independently
	// as plain JSON, so a field the model silently drops fails here.
	_, policies, err := svc.GetBaselineDevicePoliciesAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID, testBaselineDeviceUUID, nil)
	if err != nil {
		t.Fatalf("device-policies failed: %v", err)
	}
	var wantPolicies struct {
		Results []map[string]any `json:"results"`
		Total   *float64         `json:"total"`
	}
	loadFixtureResponseBodyForTest(t, "baselines/device-policies.json", &wantPolicies)
	if policies == nil || len(wantPolicies.Results) == 0 || len(policies.Results) != len(wantPolicies.Results) {
		t.Fatalf("device-policies: got %+v, want %d results", policies, len(wantPolicies.Results))
	}
	if wantPolicies.Total == nil || policies.Total == nil || *policies.Total != int(*wantPolicies.Total) {
		t.Errorf("device-policies total: got %v, want %v", policies.Total, wantPolicies.Total)
	}
	for i, want := range wantPolicies.Results {
		got := policies.Results[i]
		wantCompliance, _ := want["compliance"].(map[string]any)
		checks := []struct {
			field     string
			got, want any
		}{
			{"uuid", got.UUID, want["uuid"]},
			{"name", got.Name, want["name"]},
			{"className", got.ClassName, want["className"]},
			{"path", got.Path, want["path"]},
			{"status", got.Status, want["status"]},
		}
		if got.Compliance == nil || wantCompliance == nil {
			t.Errorf("results[%d].compliance: got %+v, want %v", i, got.Compliance, want["compliance"])
		} else {
			checks = append(checks, struct {
				field     string
				got, want any
			}{"compliance.status", got.Compliance.Status, wantCompliance["status"]})
			// errorCode is a nullable int: absent/null in the fixture must
			// decode to nil, and a present value must round-trip exactly.
			wantErr, hasErr := wantCompliance["errorCode"].(float64)
			switch {
			case hasErr && (got.Compliance.ErrorCode == nil || *got.Compliance.ErrorCode != int(wantErr)):
				t.Errorf("results[%d].compliance.errorCode: got %v, want %v", i, got.Compliance.ErrorCode, wantErr)
			case !hasErr && got.Compliance.ErrorCode != nil:
				t.Errorf("results[%d].compliance.errorCode: got %v, want nil (fixture has none)", i, *got.Compliance.ErrorCode)
			}
		}
		for _, c := range checks {
			if c.want == nil || c.want == "" || c.got != c.want {
				t.Errorf("results[%d].%s: got %v, want %v (fixture value must be non-empty)", i, c.field, c.got, c.want)
			}
		}
	}

	// DELETE — DELETE → 200 empty (D6 corrected from 204).
	_, err = svc.DeleteBaselineAsync(ctx, testBaselinesOrgGroupUUID, testBaselineUUID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}
