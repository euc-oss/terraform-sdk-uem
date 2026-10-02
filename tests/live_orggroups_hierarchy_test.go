package tests

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/orgtest"
	systemv1 "github.com/euc-oss/terraform-sdk-uem/v26/internal/system/v1"
	"github.com/euc-oss/terraform-sdk-uem/v26/resources"
)

// profileSecondDeleteStatusEnvVar overrides the HTTP status the profile's
// second-DELETE proof expects. The default, 400, is live-observed against
// the 26.2 tenant (2026-09-24): a repeat DELETE of an already-deleted profile
// returns 400 "Profile not found or User does not have access to the
// Profile", the same shape as the org-group contract in api-doctrine.md
// quirk 20.
const profileSecondDeleteStatusEnvVar = "LIVE_PROFILE_SECOND_DELETE_STATUS"

func profileSecondDeleteStatus(t *testing.T) int {
	t.Helper()
	raw := getEnvOrDefault(profileSecondDeleteStatusEnvVar, "400")
	status, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s=%q is not a valid integer: %v", profileSecondDeleteStatusEnvVar, raw, err)
	}
	return status
}

// readonlyFixturesChildName is the name this test looks for among the
// parent OG's children when checking for a persistent, shared read-only
// fixtures OG to snapshot before/after (in addition to the parent itself).
const readonlyFixturesChildName = "readonly_fixtures"

// TestLive_OrgGroupsHierarchy exercises orgtest.NewDisposableHierarchy plus
// the generated OrganizationGroupsService read surface (search/get/
// children/parents) against a disposable root+child OG pair, with a macOS
// profile created inside the child only. Containment: everything mutating
// happens inside the disposable sdktestrunN root or its one child; the
// shared ORG_GROUP_ID parent and its readonly_fixtures child (if present)
// are snapshotted before and after and asserted unchanged.
func TestLive_OrgGroupsHierarchy(t *testing.T) {
	skipUnlessLiveMutate(t)
	c := getTestClient(t, "basic")
	ctx := context.Background()
	parent := liveOrgGroupID(t)

	orgSvc := systemv1.NewOrganizationGroupsService(c)

	// Locate the persistent readonly_fixtures child, if present.
	var readonlyFixturesID int
	if _, children, err := orgSvc.GetChildLocationGroups(ctx, parent); err != nil {
		t.Logf("could not list children of parent %d to find %s: %v", parent, readonlyFixturesChildName, err)
	} else if children != nil {
		for _, child := range *children {
			if child.Name == readonlyFixturesChildName && child.ID != nil && child.ID.Value != nil {
				readonlyFixturesID = int(*child.ID.Value)
				break
			}
		}
	}
	if readonlyFixturesID == 0 {
		t.Logf("no %s child found under parent %d; snapshotting only the parent", readonlyFixturesChildName, parent)
	}

	parentBefore, err := orgtest.SnapshotOrgGroup(ctx, c, parent)
	if err != nil {
		t.Fatalf("snapshot parent %d before: %v", parent, err)
	}
	var fixturesBefore orgtest.OrgGroupSnapshot
	if readonlyFixturesID != 0 {
		fixturesBefore, err = orgtest.SnapshotOrgGroup(ctx, c, readonlyFixturesID)
		if err != nil {
			t.Fatalf("snapshot %s %d before: %v", readonlyFixturesChildName, readonlyFixturesID, err)
		}
	}
	// Registered FIRST so it runs LAST (after all other cleanup below),
	// proving the shared OGs were left untouched by everything this test did.
	t.Cleanup(func() {
		parentAfter, err := orgtest.SnapshotOrgGroup(context.Background(), c, parent)
		if err != nil {
			t.Errorf("snapshot parent %d after: %v", parent, err)
			return
		}
		if diff := parentBefore.Diff(parentAfter); diff != "" {
			t.Errorf("parent OG %d changed during test: %s", parent, diff)
		}
		if readonlyFixturesID != 0 {
			fixturesAfter, err := orgtest.SnapshotOrgGroup(context.Background(), c, readonlyFixturesID)
			if err != nil {
				t.Errorf("snapshot %s %d after: %v", readonlyFixturesChildName, readonlyFixturesID, err)
				return
			}
			if diff := fixturesBefore.Diff(fixturesAfter); diff != "" {
				t.Errorf("%s OG %d changed during test: %s", readonlyFixturesChildName, readonlyFixturesID, diff)
			}
		}
	})

	h := orgtest.NewDisposableHierarchy(t, c, parent, orgtest.WithChildren(1))
	child := h.Children[0]

	// One macOS device profile inside the child OG only. Its teardown is
	// registered after the hierarchy's, so LIFO removes it before the OGs.
	profileID := createHierarchyProfile(t, c, child.ID, child.Name+"-profile")
	l1 := sdk.NewProfilesV2Service(c)

	// --- Assertions using the generated OrganizationGroupsService ---

	// Search by root name.
	rootName := h.Root.Name
	if _, result, err := orgSvc.LocationGroupSearch(ctx, &systemv1.OrganizationGroupsLocationGroupSearchOptions{Name: &rootName}); err != nil {
		t.Errorf("LocationGroupSearch(root name %q): %v", rootName, err)
	} else if !containsLocationGroupID(result.LocationGroups, h.Root.ID) {
		t.Errorf("LocationGroupSearch(root name %q) did not include root id %d", rootName, h.Root.ID)
	}

	// Search by child name.
	childName := child.Name
	if _, result, err := orgSvc.LocationGroupSearch(ctx, &systemv1.OrganizationGroupsLocationGroupSearchOptions{Name: &childName}); err != nil {
		t.Errorf("LocationGroupSearch(child name %q): %v", childName, err)
	} else if !containsLocationGroupID(result.LocationGroups, child.ID) {
		t.Errorf("LocationGroupSearch(child name %q) did not include child id %d", childName, child.ID)
	}

	// GetAsync(child.ID).
	if _, got, err := orgSvc.GetAsync(ctx, child.ID); err != nil {
		t.Errorf("GetAsync(child %d): %v", child.ID, err)
	} else {
		if got.UUID != child.UUID {
			t.Errorf("GetAsync(child %d).UUID = %q, want %q", child.ID, got.UUID, child.UUID)
		}
		if got.Name != child.Name {
			t.Errorf("GetAsync(child %d).Name = %q, want %q", child.ID, got.Name, child.Name)
		}
		if got.LocationGroupType != "Container" {
			t.Errorf("GetAsync(child %d).LocationGroupType = %q, want %q", child.ID, got.LocationGroupType, "Container")
		}
		if got.ParentLocationGroup != nil {
			// Doctrine quirk 24: GetAsync (by-id) does not populate ParentLocationGroup.
			t.Errorf("GetAsync(child %d).ParentLocationGroup = %+v, want nil (doctrine quirk 24)", child.ID, got.ParentLocationGroup)
		}
	}

	// GetChildLocationGroups(root.ID).
	if _, children, err := orgSvc.GetChildLocationGroups(ctx, h.Root.ID); err != nil {
		t.Errorf("GetChildLocationGroups(root %d): %v", h.Root.ID, err)
	} else if children == nil {
		t.Errorf("GetChildLocationGroups(root %d) returned nil", h.Root.ID)
	} else {
		found := false
		sawRootRow := false
		for _, row := range *children {
			if row.ID != nil && row.ID.Value != nil && int(*row.ID.Value) == h.Root.ID {
				sawRootRow = true
			}
			if row.ID != nil && row.ID.Value != nil && int(*row.ID.Value) == child.ID {
				found = true
				if row.ParentLocationGroup == nil {
					t.Errorf("GetChildLocationGroups(root %d): child row has nil ParentLocationGroup", h.Root.ID)
				} else {
					if row.ParentLocationGroup.ID == nil || row.ParentLocationGroup.ID.Value == nil || int(*row.ParentLocationGroup.ID.Value) != h.Root.ID {
						t.Errorf("GetChildLocationGroups(root %d): child row's ParentLocationGroup.ID.Value = %+v, want %d", h.Root.ID, row.ParentLocationGroup.ID, h.Root.ID)
					}
					if row.ParentLocationGroup.UUID != h.Root.UUID {
						t.Errorf("GetChildLocationGroups(root %d): child row's ParentLocationGroup.Uuid = %q, want %q", h.Root.ID, row.ParentLocationGroup.UUID, h.Root.UUID)
					}
				}
			}
		}
		if !found {
			t.Errorf("GetChildLocationGroups(root %d) did not include child %d", h.Root.ID, child.ID)
		}
		if !sawRootRow {
			// Doctrine quirk 29: /children is self-inclusive.
			t.Errorf("GetChildLocationGroups(root %d) did not include the root's own row (doctrine quirk 29)", h.Root.ID)
		}
	}

	// GetParents(child.UUID).
	_, parentOG, err := orgSvc.GetAsync(ctx, parent)
	if err != nil {
		t.Fatalf("GetAsync(parent %d) to resolve its UUID: %v", parent, err)
	}
	if _, parents, err := orgSvc.GetParents(ctx, child.UUID); err != nil {
		t.Errorf("GetParents(child %s): %v", child.UUID, err)
	} else {
		t.Logf("GetParents(child %s) order: %v", child.UUID, parents.Items)
		// Doctrine quirk 30: self first, then ancestors nearest-first.
		if len(parents.Items) == 0 || parents.Items[0] != child.UUID {
			t.Errorf("GetParents(child %s)[0] = %v, want the child itself (doctrine quirk 30)", child.UUID, parents.Items)
		}
		rootIdx, parentIdx := indexOfString(parents.Items, h.Root.UUID), indexOfString(parents.Items, parentOG.UUID)
		if rootIdx < 0 {
			t.Errorf("GetParents(child %s) missing root uuid %s", child.UUID, h.Root.UUID)
		}
		if parentIdx < 0 {
			t.Errorf("GetParents(child %s) missing parent OG uuid %s", child.UUID, parentOG.UUID)
		}
		if rootIdx >= 0 && parentIdx >= 0 && rootIdx > parentIdx {
			t.Errorf("GetParents(child %s): root at %d after parent OG at %d, want nearest-first (doctrine quirk 30)", child.UUID, rootIdx, parentIdx)
		}
	}

	// Profile placement: read the profile back and assert it landed in the child OG.
	if _, details, err := l1.GetDeviceProfileDetailsAsync(ctx, profileID); err != nil {
		t.Errorf("GetDeviceProfileDetailsAsync(%d): %v", profileID, err)
	} else if details.General == nil || details.General.ManagedLocationGroupID == nil || *details.General.ManagedLocationGroupID != child.ID {
		var got any
		if details.General != nil {
			got = details.General.ManagedLocationGroupID
		}
		t.Errorf("profile %d General.ManagedLocationGroupID = %v, want %d", profileID, got, child.ID)
	}
}

func containsLocationGroupID(groups []systemv1.LocationGroupV1, id int) bool {
	for _, g := range groups {
		if g.ID != nil && g.ID.Value != nil && int(*g.ID.Value) == id {
			return true
		}
	}
	return false
}

func indexOfString(items []string, want string) int {
	for i, item := range items {
		if item == want {
			return i
		}
	}
	return -1
}

// createHierarchyProfile creates one appleosx profile managed by ogID via the
// generated create, and registers its teardown (delete, then a second delete
// that must return profileSecondDeleteStatus).
func createHierarchyProfile(t *testing.T, c *client.Client, ogID int, name string) int {
	t.Helper()
	ctx := context.Background()
	_, profileID, err := sdk.NewProfilesV2Service(c).CreateAppleOsXDeviceProfileAsync(ctx, &sdk.AppleOsXDeviceProfileEntityV2{
		General: &sdk.GeneralPayloadV2Entity{
			Name:                   name,
			ProfileType:            "Device",
			ManagedLocationGroupID: intPtr(ogID),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
		},
		CustomSettingsList: []sdk.AppleOsXCustomSettingsPayloadEntityV2{{
			CustomSettings: "<dict><key>PayloadType</key><string>com.apple.example.test</string><key>PayloadIdentifier</key><string>com.example.hierarchy.custom</string><key>PayloadDisplayName</key><string>Hierarchy Custom</string><key>PayloadVersion</key><integer>1</integer><key>MyCustomKey</key><string>MyCustomValue</string></dict>",
		}},
	})
	if err != nil {
		t.Fatalf("CreateAppleOsXDeviceProfileAsync(%s): %v", name, err)
	}
	if profileID == 0 {
		t.Fatalf("CreateAppleOsXDeviceProfileAsync(%s) returned zero profile id", name)
	}
	profileSvc := resources.NewProfileService(c)
	orgtest.RegisterDeleteWithProof(t, fmt.Sprintf("profile %d", profileID),
		func() error { return profileSvc.Delete(context.Background(), profileID) },
		func() (int, error) {
			err := profileSvc.Delete(context.Background(), profileID)
			if err == nil {
				return 200, nil
			}
			var apiErr *client.APIError
			if errors.As(err, &apiErr) {
				return apiErr.StatusCode, err
			}
			return 0, err
		},
		profileSecondDeleteStatus(t),
	)
	return profileID
}

// TestLive_ProfileDiscoverOrgGroupScope proves the Layer-2
// WithProfileServiceOrganizationGroupID option really scopes Discover.
// Scoped to a disposable root, Discover finds a profile created in that
// root's child (doctrine quirk 35: the scope covers descendants). Scoped to a
// separate sibling disposable root, it does not -- the negative control that
// shows the scope is applied, since an unscoped Discover would find the
// profile tenant-wide.
func TestLive_ProfileDiscoverOrgGroupScope(t *testing.T) {
	skipUnlessLiveMutate(t)
	c := getTestClient(t, "basic")
	ctx := context.Background()
	parent := liveOrgGroupID(t)

	h := orgtest.NewDisposableHierarchy(t, c, parent, orgtest.WithChildren(1))
	sibling := orgtest.NewDisposableHierarchy(t, c, parent)
	child := h.Children[0]
	profileID := createHierarchyProfile(t, c, child.ID, child.Name+"-discover")

	scoped := sdk.NewProfileServiceWithoutDiscovery(c, sdk.WithProfileServiceOrganizationGroupID(h.Root.ID))
	if err := scoped.Discover(ctx); err != nil {
		t.Fatalf("Discover scoped to root %d: %v", h.Root.ID, err)
	}
	if got, err := scoped.Get(ctx, profileID); err != nil {
		t.Errorf("Get(%d) after Discover scoped to root %d: %v (want the child-OG profile found)", profileID, h.Root.ID, err)
	} else if got == nil {
		t.Errorf("Get(%d) after Discover scoped to root %d returned nil", profileID, h.Root.ID)
	}

	other := sdk.NewProfileServiceWithoutDiscovery(c, sdk.WithProfileServiceOrganizationGroupID(sibling.Root.ID))
	if err := other.Discover(ctx); err != nil {
		t.Fatalf("Discover scoped to sibling root %d: %v", sibling.Root.ID, err)
	}
	_, err := other.Get(ctx, profileID)
	var notFound *client.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("Get(%d) with Discover scoped to sibling root %d: err = %v, want *client.NotFoundError (scope not applied?)", profileID, sibling.Root.ID, err)
	}
}
