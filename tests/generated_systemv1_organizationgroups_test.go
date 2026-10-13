package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
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

	// Expected values are read from the fixture's own body (the genuine
	// 26.4 recapture), not hardcoded.
	var want struct {
		LocationGroups []struct {
			Name              string `json:"Name"`
			GroupID           string `json:"GroupId"`
			LocationGroupType string `json:"LocationGroupType"`
			ID                struct {
				Value *int64 `json:"Value"`
			} `json:"Id"`
		} `json:"LocationGroups"`
		Page     *int `json:"Page"`
		PageSize *int `json:"PageSize"`
		Total    *int `json:"Total"`
	}
	loadFixtureResponseBodyForTest(t, "org-groups/groups_get_search.json", &want)
	if len(want.LocationGroups) == 0 || want.Total == nil {
		t.Fatal("org-groups/groups_get_search.json has no LocationGroups rows or no Total -- cannot derive expected values")
	}

	if got.Page == nil || want.Page == nil || *got.Page != *want.Page {
		t.Errorf("Page: got %v, want %v", got.Page, want.Page)
	}
	if got.PageSize == nil || want.PageSize == nil || *got.PageSize != *want.PageSize {
		t.Errorf("PageSize: got %v, want %v", got.PageSize, want.PageSize)
	}
	if got.Total == nil || *got.Total != *want.Total {
		t.Errorf("Total: got %v, want %v", got.Total, *want.Total)
	}

	if len(got.LocationGroups) != len(want.LocationGroups) {
		t.Fatalf("LocationGroups: got %d items, want %d", len(got.LocationGroups), len(want.LocationGroups))
	}

	first := got.LocationGroups[0]
	wantFirst := want.LocationGroups[0]
	if first.Name != wantFirst.Name {
		t.Errorf("LocationGroups[0].Name: got %q, want %q", first.Name, wantFirst.Name)
	}
	if first.GroupID != wantFirst.GroupID {
		t.Errorf("LocationGroups[0].GroupID: got %q, want %q", first.GroupID, wantFirst.GroupID)
	}
	if first.LocationGroupType != wantFirst.LocationGroupType {
		t.Errorf("LocationGroups[0].LocationGroupType: got %q, want %q", first.LocationGroupType, wantFirst.LocationGroupType)
	}
	if first.ID == nil || first.ID.Value == nil || wantFirst.ID.Value == nil || *first.ID.Value != *wantFirst.ID.Value {
		t.Errorf("LocationGroups[0].ID.Value: got %+v, want %v", first.ID, wantFirst.ID.Value)
	}

	last := got.LocationGroups[len(got.LocationGroups)-1]
	wantLast := want.LocationGroups[len(want.LocationGroups)-1]
	if last.Name != wantLast.Name {
		t.Errorf("LocationGroups[last].Name: got %q, want %q", last.Name, wantLast.Name)
	}
	if last.GroupID != wantLast.GroupID {
		t.Errorf("LocationGroups[last].GroupID: got %q, want %q", last.GroupID, wantLast.GroupID)
	}
	if last.LocationGroupType != wantLast.LocationGroupType {
		t.Errorf("LocationGroups[last].LocationGroupType: got %q, want %q", last.LocationGroupType, wantLast.LocationGroupType)
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
// anonymized) GET /api/system/groups/{id} fixture: org-groups/groups_get.json
// (the genuine 26.4.0.0 capture; on 26.2 this scenario was
// org-groups/groups_get_id.json, which the 26.4 corpus supersedes). Expected
// values are read from the fixture's own body, so the test asserts the SDK
// decodes the capture faithfully rather than pinning one tenant's data.
//
// Divergence side 1: per canonical doc
// docs/guides/org-groups/canonical-models/org-groups-response-v1.md:33, "The
// ParentLocationGroup property is explicitly set to null before returning"
// for this single-item GET -- and indeed the fixture has no
// "ParentLocationGroup" key at all. Compare against
// TestGeneratedOrganizationGroupsV1GetChildren below, where the same field
// is populated.
func TestGeneratedOrganizationGroupsV1GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const fixture = "org-groups/groups_get.json"
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/org-groups/groups_get.json"))
	if err != nil {
		t.Fatalf("load %s: %v", fixture, err)
	}
	var want struct {
		UUID              string `json:"Uuid"`
		Name              string `json:"Name"`
		GroupID           string `json:"GroupId"`
		LocationGroupType string `json:"LocationGroupType"`
	}
	loadFixtureResponseBodyForTest(t, fixture, &want)
	if want.UUID == "" || want.Name == "" || want.GroupID == "" || want.LocationGroupType == "" {
		t.Fatalf("%s: fixture body lacks Uuid/Name/GroupId/LocationGroupType: %+v", fixture, want)
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

	if got.UUID != want.UUID {
		t.Errorf("UUID: got %q, want the fixture's %q", got.UUID, want.UUID)
	}
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want the fixture's %q", got.Name, want.Name)
	}
	if got.GroupID != want.GroupID {
		t.Errorf("GroupID: got %q, want the fixture's %q", got.GroupID, want.GroupID)
	}
	if got.LocationGroupType != want.LocationGroupType {
		t.Errorf("LocationGroupType: got %q, want the fixture's %q", got.LocationGroupType, want.LocationGroupType)
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

	// Expected count and first-item field values are read from the
	// fixture's own body (the genuine 26.4 recapture), not hardcoded.
	var wantChildren []struct {
		Name                string `json:"Name"`
		UUID                string `json:"Uuid"`
		ParentLocationGroup *struct {
			ID struct {
				Value *int64 `json:"Value"`
			} `json:"Id"`
			UUID string `json:"Uuid"`
		} `json:"ParentLocationGroup"`
	}
	loadFixtureResponseBodyForTest(t, "org-groups/groups_get_id_children.json", &wantChildren)
	if len(wantChildren) == 0 {
		t.Fatal("org-groups/groups_get_id_children.json has no rows -- cannot derive expected values")
	}
	if len(*got) != len(wantChildren) {
		t.Fatalf("children: got %d items, want %d (from the fixture)", len(*got), len(wantChildren))
	}

	child := (*got)[0]
	wantChild := wantChildren[0]
	if child.Name != wantChild.Name {
		t.Errorf("child.Name: got %q, want %q", child.Name, wantChild.Name)
	}
	if child.UUID != wantChild.UUID {
		t.Errorf("child.UUID: got %q, want %q", child.UUID, wantChild.UUID)
	}

	// Divergence assertion, side 2 (see doc comment above).
	if child.ParentLocationGroup == nil {
		t.Fatal("ParentLocationGroup: got nil, want populated -- GET /groups/{id}/children fully populates this field (org-groups-response-v1.md:166)")
	}
	if wantChild.ParentLocationGroup == nil {
		t.Fatal("fixture's first row has no ParentLocationGroup -- cannot derive expected values")
	}
	if child.ParentLocationGroup.ID == nil || child.ParentLocationGroup.ID.Value == nil || wantChild.ParentLocationGroup.ID.Value == nil || *child.ParentLocationGroup.ID.Value != *wantChild.ParentLocationGroup.ID.Value {
		t.Errorf("ParentLocationGroup.ID.Value: got %+v, want %v", child.ParentLocationGroup.ID, wantChild.ParentLocationGroup.ID.Value)
	}
	if child.ParentLocationGroup.UUID != wantChild.ParentLocationGroup.UUID {
		t.Errorf("ParentLocationGroup.UUID: got %q, want %q", child.ParentLocationGroup.UUID, wantChild.ParentLocationGroup.UUID)
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

	// Expected items are read from the fixture's own body (the genuine 26.4
	// recapture), not hardcoded.
	var wantBody struct {
		Items []string `json:"items"`
	}
	loadFixtureResponseBodyForTest(t, "org-groups/groups_get_id_parents.json", &wantBody)
	want := wantBody.Items
	if len(want) == 0 {
		t.Fatal("org-groups/groups_get_id_parents.json has no items -- cannot derive expected values")
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
