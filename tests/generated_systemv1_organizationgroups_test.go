package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// testOrgGroupID is the ID used across the OrganizationGroups tests below.
// The mockserver matches by path template ({id}), not by exact ID value,
// so this is arbitrary -- it does NOT need to match the real ID used during
// live capture (which was anonymized out of the fixtures anyway).
const testOrgGroupID = 12346

// TestGeneratedOrganizationGroupsV1Search exercises the live-captured (and
// anonymized) GET /api/system/groups/search fixture:
// org-groups/groups_get_search.json.
func TestGeneratedOrganizationGroupsV1Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewOrganizationGroupsService(c)
	ctx := context.Background()

	_, got, err := svc.LocationGroupSearch(ctx, nil)
	if err != nil {
		t.Fatalf("LocationGroupSearch failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil search result")
	}

	if got.Page == nil || *got.Page != 0 {
		t.Errorf("Page: got %v, want 0", got.Page)
	}
	if got.PageSize == nil || *got.PageSize != 500 {
		t.Errorf("PageSize: got %v, want 500", got.PageSize)
	}
	if got.Total == nil || *got.Total != 4 {
		t.Errorf("Total: got %v, want 4", got.Total)
	}

	if len(got.LocationGroups) != 4 {
		t.Fatalf("LocationGroups: got %d items, want 4", len(got.LocationGroups))
	}

	first := got.LocationGroups[0]
	if first.Name != "Test Koaladid" {
		t.Errorf("LocationGroups[0].Name: got %q, want %q", first.Name, "Test Koaladid")
	}
	if first.GroupID != "Schmitt5713" {
		t.Errorf("LocationGroups[0].GroupID: got %q, want %q", first.GroupID, "Schmitt5713")
	}
	if first.LocationGroupType != "Container" {
		t.Errorf("LocationGroups[0].LocationGroupType: got %q, want %q", first.LocationGroupType, "Container")
	}
	if first.ID == nil || first.ID.Value == nil || *first.ID.Value != 12346 {
		t.Errorf("LocationGroups[0].ID.Value: got %+v, want 12346", first.ID)
	}

	third := got.LocationGroups[2]
	if third.Name != "Test Tennisdo" {
		t.Errorf("LocationGroups[2].Name: got %q, want %q", third.Name, "Test Tennisdo")
	}
	if third.GroupID != "Parisianllama" {
		t.Errorf("LocationGroups[2].GroupID: got %q, want %q", third.GroupID, "Parisianllama")
	}
	if third.LocationGroupType != "Customer" {
		t.Errorf("LocationGroups[2].LocationGroupType: got %q, want %q", third.LocationGroupType, "Customer")
	}

	// Search-result rows never carry a Uuid on the wire (confirmed by the
	// fixture: no item has a "Uuid" key at all), unlike the single-item
	// GET-by-id response below. Assert this explicitly so a regression that
	// starts populating Uuid from search doesn't slip by unnoticed.
	for i, lg := range got.LocationGroups {
		if lg.UUID != "" {
			t.Errorf("LocationGroups[%d].UUID: got %q, want empty (search rows carry no Uuid)", i, lg.UUID)
		}
	}
}

// TestGeneratedOrganizationGroupsV1GetByID exercises the live-captured (and
// anonymized) GET /api/system/groups/{id} fixture: org-groups/groups_get_id.json.
//
// Divergence side 1: per canonical doc
// docs/guides/org-groups/canonical-models/org-groups-response-v1.md:33, "The
// ParentLocationGroup property is explicitly set to null before returning"
// for this single-item GET -- and indeed the fixture has no
// "ParentLocationGroup" key at all. Compare against
// TestGeneratedOrganizationGroupsV1GetChildren below, where the same field
// is populated.
//
// NOTE on fixture collision: org-groups/groups_get.json (a different,
// unrelated ledger entry backing OrgGroupResolution.GetOrganizationGroup)
// declares the exact same endpoint/method (GET /api/system/groups/{id})
// with no distinguishing query params as org-groups/groups_get_id.json. A
// full-corpus mock server load always resolves a plain GET to the
// alphabetically-first fixture ("groups_get.json" < "groups_get_id.json",
// since '.' sorts before '_'). This mirrors the identical collision already
// documented for SmartGroupsV1 (see generated_smartgroupsv1_test.go). An
// isolated single-fixture mock server is required here to exercise
// groups_get_id.json specifically.
func TestGeneratedOrganizationGroupsV1GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/org-groups/groups_get_id.json"))
	if err != nil {
		t.Fatalf("load groups_get_id fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewOrganizationGroupsService(c)
	ctx := context.Background()

	_, got, err := svc.GetAsync(ctx, testOrgGroupID)
	if err != nil {
		t.Fatalf("GetAsync failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil location group")
	}

	if got.UUID != "e8a0a223-9cf1-eaa1-af17-a7b985793f5c" {
		t.Errorf("UUID: got %q, want %q", got.UUID, "e8a0a223-9cf1-eaa1-af17-a7b985793f5c")
	}
	if got.Name != "Test Standwill" {
		t.Errorf("Name: got %q, want %q", got.Name, "Test Standwill")
	}
	if got.GroupID != "sdktest" {
		t.Errorf("GroupID: got %q, want %q", got.GroupID, "sdktest")
	}
	if got.LocationGroupType != "Customer" {
		t.Errorf("LocationGroupType: got %q, want %q", got.LocationGroupType, "Customer")
	}

	// Divergence assertion, side 1 (see doc comment above).
	if got.ParentLocationGroup != nil {
		t.Errorf("ParentLocationGroup: got %+v, want nil -- GET /groups/{id} explicitly nulls this field before returning (org-groups-response-v1.md:33)", got.ParentLocationGroup)
	}
}

// TestGeneratedOrganizationGroupsV1GetChildren exercises the live-captured
// (and anonymized) GET /api/system/groups/{id}/children fixture:
// org-groups/groups_get_id_children.json.
//
// Divergence side 2: per canonical doc
// docs/guides/org-groups/canonical-models/org-groups-response-v1.md:166,
// "Each child organization group is fully populated" -- and indeed the
// fixture's one array element has a populated ParentLocationGroup. Compare
// against TestGeneratedOrganizationGroupsV1GetByID above, where the same
// field is explicitly nulled.
func TestGeneratedOrganizationGroupsV1GetChildren(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewOrganizationGroupsService(c)
	ctx := context.Background()

	_, got, err := svc.GetChildLocationGroups(ctx, testOrgGroupID)
	if err != nil {
		t.Fatalf("GetChildLocationGroups failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil children slice")
	}
	if len(*got) != 1 {
		t.Fatalf("children: got %d items, want 1 (fixture reflects the OG's own row, no real children exist)", len(*got))
	}

	child := (*got)[0]
	if child.Name != "Test Wateropen" {
		t.Errorf("child.Name: got %q, want %q", child.Name, "Test Wateropen")
	}
	if child.UUID != "689df136-a2c8-1c6d-ae6d-e0d9b0b6fec4" {
		t.Errorf("child.UUID: got %q, want %q", child.UUID, "689df136-a2c8-1c6d-ae6d-e0d9b0b6fec4")
	}

	// Divergence assertion, side 2 (see doc comment above).
	if child.ParentLocationGroup == nil {
		t.Fatal("ParentLocationGroup: got nil, want populated -- GET /groups/{id}/children fully populates this field (org-groups-response-v1.md:166)")
	}
	if child.ParentLocationGroup.ID == nil || child.ParentLocationGroup.ID.Value == nil || *child.ParentLocationGroup.ID.Value != 12347 {
		t.Errorf("ParentLocationGroup.ID.Value: got %+v, want 12347", child.ParentLocationGroup.ID)
	}
	if child.ParentLocationGroup.UUID != "10fdc858-9266-1ebe-40a6-47b043411839" {
		t.Errorf("ParentLocationGroup.UUID: got %q, want %q", child.ParentLocationGroup.UUID, "10fdc858-9266-1ebe-40a6-47b043411839")
	}
}

// TestGeneratedOrganizationGroupsV1GetParents exercises the live-captured
// (and anonymized) GET /api/system/groups/{uuid}/parents fixture:
// org-groups/groups_get_id_parents.json.
//
// This endpoint is a genuine swagger gap-fill (see
// internal-source/systemv1.organizationgroups-parents.overlay.yaml and
// api-doctrine.md quirk 23): documented in canonical C# source
// (org-groups-response-v1.md:330-358) but entirely absent from the vendored
// 26.2 swagger. Live capture confirmed the response's array field is wire
// lowercase "items" -- matching the pre-existing (if previously unused)
// OrganizationGroupCollectionV1Model schema in systemv1.json -- so no
// casing override was required.
func TestGeneratedOrganizationGroupsV1GetParents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewOrganizationGroupsService(c)
	ctx := context.Background()

	_, got, err := svc.GetParents(ctx, "4bb2b420-a03d-f150-fdbc-139c49f6357d")
	if err != nil {
		t.Fatalf("GetParents failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil parents collection")
	}

	want := []string{
		"340c16e2-0750-f33b-5449-477fd584e16d",
		"de6a7cca-2cb3-9aaa-2450-1c84b5077a9f",
		"580192fa-028d-ee8e-13f5-f71d9c91130b",
	}
	if len(got.Items) != len(want) {
		t.Fatalf("Items: got %d entries, want %d", len(got.Items), len(want))
	}
	for i, w := range want {
		if got.Items[i] != w {
			t.Errorf("Items[%d]: got %q, want %q", i, got.Items[i], w)
		}
	}
}
