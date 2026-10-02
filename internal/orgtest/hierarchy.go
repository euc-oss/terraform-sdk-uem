// hierarchy.go adds a multi-OG (root + children) containment helper on top
// of orggroup.go's single-OG CreateDisposableOrgGroup, plus a read-only
// snapshot/diff helper for asserting a shared OG was left untouched. Both
// are additive: CreateDisposableOrgGroup's own behavior and signature are
// unchanged; this file's create path reuses the same request shape via a
// small unexported helper rather than duplicating it.
package orgtest

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/euc-oss/terraform-sdk-uem/client"
	systemv1 "github.com/euc-oss/terraform-sdk-uem/internal/system/v1"
)

// HierarchyT is a narrow testing interface so both *testing.T and small
// adapters can drive NewDisposableHierarchy and RegisterDeleteWithProof.
type HierarchyT interface {
	Helper()
	Cleanup(func())
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// OrgGroupRef identifies one created org group.
type OrgGroupRef struct {
	ID   int
	UUID string
	Name string
}

// Hierarchy is a disposable root OG plus zero or more disposable child OGs
// created directly under it.
type Hierarchy struct {
	Root     OrgGroupRef
	Children []OrgGroupRef // same order as created
}

type hierarchyConfig struct {
	children int
}

// HierarchyOption configures NewDisposableHierarchy.
type HierarchyOption func(*hierarchyConfig)

// WithChildren requests n disposable child OGs under the disposable root.
func WithChildren(n int) HierarchyOption {
	return func(cfg *hierarchyConfig) {
		cfg.children = n
	}
}

// createOrgGroup issues the create call shared by CreateDisposableOrgGroup
// and NewDisposableHierarchy -- same request shape/headers, no cleanup
// registration (callers register their own).
func createOrgGroup(ctx context.Context, c *client.Client, parentID int, name string) (OrgGroupResponse, error) {
	req := AddOrgGroupRequest{
		Name:                  name,
		GroupCode:             name,
		Country:               "United States",
		OrganizationGroupType: "Container",
	}
	var resp OrgGroupResponse
	_, err := c.DoRequest(ctx, "POST", fmt.Sprintf("/api/system/groups/%d", parentID), OrgGroupCreateAcceptHeader, OrgGroupContentType, req, &resp)
	if err != nil {
		return OrgGroupResponse{}, fmt.Errorf("create org group %q under parent %d: %w", name, parentID, err)
	}
	return resp, nil
}

// deleteOrgGroup issues the delete call shared by both the primary and
// proof deletes: same Accept/content-type as create. On a non-2xx response
// it returns the HTTP status extracted from *client.APIError; if the
// server unexpectedly returns a 2xx, it returns that status with a nil
// error so a caller checking "did the second delete come back 400" gets a
// real status instead of a false pass.
func deleteOrgGroup(ctx context.Context, c *client.Client, uuid string) (status int, err error) {
	_, doErr := c.DoRequest(ctx, "DELETE", "/api/system/groups/"+uuid, OrgGroupCreateAcceptHeader, OrgGroupContentType, nil, nil)
	if doErr == nil {
		return 200, nil
	}
	if apiErr, ok := doErr.(*client.APIError); ok {
		return apiErr.StatusCode, doErr
	}
	return 0, doErr
}

// RegisterDeleteWithProof registers (via t.Cleanup) deleteFn followed by
// secondDeleteFn, run in that order when cleanup fires. It fails the test
// (t.Errorf, never t.Fatalf -- cleanup must not abort other cleanups) if
// deleteFn errors, if secondDeleteFn returns an error that carries no
// usable HTTP status, or if the second call's status != wantSecondStatus.
func RegisterDeleteWithProof(t HierarchyT, desc string, deleteFn func() error, secondDeleteFn func() (status int, err error), wantSecondStatus int) {
	t.Cleanup(func() {
		if err := deleteFn(); err != nil {
			t.Errorf("%s: delete failed: %v", desc, err)
			return
		}
		status, err := secondDeleteFn()
		if err == nil {
			t.Errorf("%s: second delete unexpectedly succeeded (status %d); want a %d \"not found\" per doctrine quirk 20", desc, status, wantSecondStatus)
			return
		}
		if status == 0 {
			t.Errorf("%s: second delete error carried no HTTP status: %v", desc, err)
			return
		}
		if status != wantSecondStatus {
			t.Errorf("%s: second delete status = %d, want %d (doctrine quirk 20 proof-of-deletion check): %v", desc, status, wantSecondStatus, err)
		}
	})
}

// NewDisposableHierarchy creates a disposable sdktestrunN root OG under
// parentOGID and, if requested via WithChildren, n child OGs directly under
// that root. Any create failure calls t.Fatalf -- it never falls back to
// the shared parent OG. Teardown is registered with t.Cleanup so anything
// the caller registers AFTER this call is torn down BEFORE these OGs
// (LIFO); children are deleted before the root, each followed by a second
// DELETE that must return HTTP 400 (doctrine quirk 20).
func NewDisposableHierarchy(t HierarchyT, c *client.Client, parentOGID int, opts ...HierarchyOption) *Hierarchy {
	t.Helper()
	cfg := hierarchyConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	ctx := context.Background()
	existing, err := DiscoverSdkTestRunNames(ctx, c, parentOGID)
	if err != nil {
		t.Fatalf("NewDisposableHierarchy: discover existing sdktestrunN names under parent %d: %v", parentOGID, err)
	}
	rootName := NextSdkTestRunName(existing)

	rootResp, err := createOrgGroup(ctx, c, parentOGID, rootName)
	if err != nil {
		t.Fatalf("NewDisposableHierarchy: create root %q under parent %d: %v", rootName, parentOGID, err)
	}
	root := OrgGroupRef{ID: rootResp.ID, UUID: rootResp.UUID, Name: rootName}

	// Register root's teardown before any children so LIFO cleanup order
	// runs children first, root last.
	RegisterDeleteWithProof(t, fmt.Sprintf("org group %s (%s)", root.Name, root.UUID),
		func() error {
			_, err := deleteOrgGroup(context.Background(), c, root.UUID)
			return err
		},
		func() (int, error) {
			return deleteOrgGroup(context.Background(), c, root.UUID)
		},
		400,
	)

	h := &Hierarchy{Root: root}

	for i := 1; i <= cfg.children; i++ {
		childName := root.Name + "c" + strconv.Itoa(i)
		childResp, err := createOrgGroup(ctx, c, root.ID, childName)
		if err != nil {
			t.Fatalf("NewDisposableHierarchy: create child %q under root %d: %v", childName, root.ID, err)
		}
		child := OrgGroupRef{ID: childResp.ID, UUID: childResp.UUID, Name: childName}
		h.Children = append(h.Children, child)

		RegisterDeleteWithProof(t, fmt.Sprintf("org group %s (%s)", child.Name, child.UUID),
			func() error {
				_, err := deleteOrgGroup(context.Background(), c, child.UUID)
				return err
			},
			func() (int, error) {
				return deleteOrgGroup(context.Background(), c, child.UUID)
			},
			400,
		)
	}

	return h
}

// OrgGroupSnapshot is a point-in-time read of one OG's identity and
// immediate subtree, used to prove a shared OG wasn't mutated by a test.
type OrgGroupSnapshot struct {
	ID                int
	UUID              string
	Name              string
	GroupID           string
	LocationGroupType string
	SubtreeIDs        []int // sorted; excludes disposable sdktestrunN OGs
}

// SnapshotOrgGroup reads an OG's identity (GetAsync) and immediate children
// (GetChildLocationGroups) via the generated OrganizationGroups service.
func SnapshotOrgGroup(ctx context.Context, c *client.Client, id int) (OrgGroupSnapshot, error) {
	svc := systemv1.NewOrganizationGroupsService(c)

	_, group, err := svc.GetAsync(ctx, id)
	if err != nil {
		return OrgGroupSnapshot{}, fmt.Errorf("SnapshotOrgGroup: get %d: %w", id, err)
	}
	_, children, err := svc.GetChildLocationGroups(ctx, id)
	if err != nil {
		return OrgGroupSnapshot{}, fmt.Errorf("SnapshotOrgGroup: get children of %d: %w", id, err)
	}

	snap := OrgGroupSnapshot{
		ID:                id,
		UUID:              group.UUID,
		Name:              group.Name,
		GroupID:           group.GroupID,
		LocationGroupType: group.LocationGroupType,
	}
	if children != nil {
		for _, child := range *children {
			// Disposable sdktestrunN OGs (any child whose name starts with
			// the prefix) come and go as any concurrent live run
			// creates and tears them down; they aren't part of the shared state
			// this snapshot guards.
			if strings.HasPrefix(child.Name, SdkTestRunPrefix) {
				continue
			}
			if child.ID != nil && child.ID.Value != nil {
				snap.SubtreeIDs = append(snap.SubtreeIDs, int(*child.ID.Value))
			}
		}
	}
	sort.Ints(snap.SubtreeIDs)
	return snap, nil
}

// Diff returns "" if a and b are equal, else a human-readable description
// of what differs.
func (a OrgGroupSnapshot) Diff(b OrgGroupSnapshot) string {
	var diffs []string
	if a.ID != b.ID {
		diffs = append(diffs, fmt.Sprintf("ID: %d != %d", a.ID, b.ID))
	}
	if a.UUID != b.UUID {
		diffs = append(diffs, fmt.Sprintf("UUID: %q != %q", a.UUID, b.UUID))
	}
	if a.Name != b.Name {
		diffs = append(diffs, fmt.Sprintf("Name: %q != %q", a.Name, b.Name))
	}
	if a.GroupID != b.GroupID {
		diffs = append(diffs, fmt.Sprintf("GroupID: %q != %q", a.GroupID, b.GroupID))
	}
	if a.LocationGroupType != b.LocationGroupType {
		diffs = append(diffs, fmt.Sprintf("LocationGroupType: %q != %q", a.LocationGroupType, b.LocationGroupType))
	}
	if fmt.Sprint(a.SubtreeIDs) != fmt.Sprint(b.SubtreeIDs) {
		diffs = append(diffs, fmt.Sprintf("SubtreeIDs: %v != %v", a.SubtreeIDs, b.SubtreeIDs))
	}
	if len(diffs) == 0 {
		return ""
	}
	out := "org group snapshot diff:"
	for _, d := range diffs {
		out += " " + d + ";"
	}
	return out
}
