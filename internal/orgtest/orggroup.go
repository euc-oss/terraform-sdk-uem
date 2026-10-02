// Package orgtest provides disposable-org-group helpers shared by the live
// test suite (package tests) and the capture tooling (package main). It lives
// at the top-level internal/ (not tools/internal/) deliberately: Go's
// internal-import rule only allows importing internal/... from code rooted
// ABOVE the internal segment, and both the capture tooling and tests need to
// reach this package -- tools/internal/orgtest would not be importable from
// tests, since tests does not live under tools/.
//
// This package ships public-but-dormant (D2, Model P): TEST_MODE-gated,
// no strip step, present in every build regardless of language target.
// Consumers who want to run it against their own tenant supply their own
// .env; the helpers here never talk to a real tenant unless a caller
// explicitly constructs a live *client.Client and invokes them.
//
// The LIVE_MUTATE opt-in (a second gate separate from TEST_MODE=live, meant
// to gate the mutating tier specifically) is a Phase-2 dependency
// (skipUnlessLiveMutate, Task 2.4.2) and does NOT exist anywhere in this
// branch yet -- do not assume it protects a live call made through these
// functions today; it doesn't, because nothing here wires a live client in
// except a caller that constructs one itself.
//
// No tenant hostname, real OG ID/UUID, or tracker reference appears
// anywhere in this file by construction -- it only ever talks to
// httptest.Server URLs (unit tests) or env-supplied hosts (live mode).
package orgtest

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/euc-oss/terraform-sdk-uem/client"
)

// SdkTestRunPrefix is the naming convention every disposable org group
// created by this package uses: SdkTestRunPrefix + a positive integer
// (e.g. "sdktestrun1"). The capture tooling's write-safety gate accepts this
// prefix (alongside "mocktest") specifically so OGs created through this
// package satisfy that gate without an operator having to rename anything.
const SdkTestRunPrefix = "sdktestrun"

// NextSdkTestRunName scans existing child-OG names for the sdktestrunN
// convention and returns the next unused name (sdktestrun<max+1>), or
// sdktestrun1 if none match. Non-matching names are ignored, not errors.
func NextSdkTestRunName(existing []string) string {
	highest := 0
	for _, name := range existing {
		if !strings.HasPrefix(name, SdkTestRunPrefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(name, SdkTestRunPrefix))
		if err != nil {
			continue
		}
		if n > highest {
			highest = n
		}
	}
	return SdkTestRunPrefix + strconv.Itoa(highest+1)
}

// AddOrgGroupRequest mirrors AddOrganizationGroupRequestModel (canonical
// shape captured in docs/guides/org-groups/canonical-models/, never a
// codegen output -- no OG resource exists in the SDK surface). Deliberately
// minimal: only the fields required to satisfy Create; customer_industry_type
// is omitted because the helper always creates Container-type disposable OGs.
type AddOrgGroupRequest struct {
	Name                  string `json:"name"`
	GroupCode             string `json:"group_code"`
	Country               string `json:"country"`
	OrganizationGroupType string `json:"organization_group_type"`
}

// OrgGroupResponse mirrors OrganizationGroupResponseModel's Create-response
// shape: {Id, Uuid}, 201 body, per docs/guides/org-groups/canonical-models/
// org-groups-response-v2.md.
type OrgGroupResponse struct {
	ID   int    `json:"Id"`
	UUID string `json:"Uuid"`
}

// OrgGroupCreateAcceptHeader and OrgGroupContentType are shared by every
// create/delete call this package makes, and by the capture tooling's
// org-group delete path for symmetry with CreateDisposableOrgGroup's own
// accept header.
const OrgGroupCreateAcceptHeader = "application/json;version=3"
const OrgGroupContentType = "application/json"

// CreateDisposableOrgGroup creates one child OG under parentID named name,
// registers a LIFO, log-don't-fail t.Cleanup that deletes it, and returns
// its numeric ID and UUID. Uses c.DoRequest directly (no generated OG
// resource exists in the SDK -- greenfield per findings_4.md).
//
// The t interface{Cleanup; Logf} parameter style lets both *testing.T and a
// non-test caller (e.g. the capture tooling's disposable-org-group creation)
// satisfy it -- a plain main-package caller passes a small adapter struct
// implementing Cleanup/Logf instead of a real *testing.T. A CLI caller's
// Cleanup should be a no-op: t.Cleanup never fires in a one-shot main(), so
// a CLI wrapper script must do its own explicit delete-og step instead of
// relying on this to tear anything down.
func CreateDisposableOrgGroup(t interface {
	Cleanup(func())
	Logf(string, ...interface{})
}, c *client.Client, parentID int, name string) (int, string, error) {
	ctx := context.Background()
	req := AddOrgGroupRequest{
		Name:                  name,
		GroupCode:             name,
		Country:               "United States",
		OrganizationGroupType: "Container",
	}
	var resp OrgGroupResponse
	_, err := c.DoRequest(ctx, "POST", fmt.Sprintf("/api/system/groups/%d", parentID), OrgGroupCreateAcceptHeader, OrgGroupContentType, req, &resp)
	if err != nil {
		return 0, "", fmt.Errorf("create disposable org group %q under parent %d: %w", name, parentID, err)
	}

	t.Cleanup(func() {
		if _, err := c.DoRequest(context.Background(), "DELETE", "/api/system/groups/"+resp.UUID, OrgGroupCreateAcceptHeader, OrgGroupContentType, nil, nil); err != nil {
			t.Logf("cleanup: delete disposable org group %s (%s) failed: %v", name, resp.UUID, err)
		}
	})

	return resp.ID, resp.UUID, nil
}

// NewDisposableOrgGroup is the harness-facing entrypoint: given a parent OG
// ID and the list of existing child-OG names already discovered under it
// (nil is fine -- an empty/unknown discovery set, e.g. when only a probe
// scenario is available), it creates sdktestrun<N+1> and returns its ID.
// On ANY create failure (e.g. 403 6310/6311 -- creds cannot create OGs) it
// logs and falls back to the shared-fixed parentID itself, per D5's
// documented fallback -- it does NOT register a delete cleanup in the
// fallback case, since there is no disposable resource to delete.
func NewDisposableOrgGroup(t interface {
	Cleanup(func())
	Logf(string, ...interface{})
}, c *client.Client, parentID int, existingNames []string) int {
	name := NextSdkTestRunName(existingNames)
	id, _, err := CreateDisposableOrgGroup(t, c, parentID, name)
	if err != nil {
		t.Logf("NewDisposableOrgGroup: create %q failed (%v); falling back to shared-fixed parent OG %d per D5 -- ESCALATE if this persists (see api-doctrine.md D6 probe)", name, err, parentID)
		return parentID
	}
	return id
}

// NewDisposableOrgGroupOrFatal is NewDisposableOrgGroup's MUTATE-TIER
// variant. The read-only fallback-to-shared-OG semantics (above) are
// correct for a read-only caller -- there is nothing to contain, so
// proceeding against the shared, real OG is safe. A MUTATING caller must
// never do that: if the disposable-OG create failed (e.g. narrower
// OG-create permissions than resource-create permissions) and
// NewDisposableOrgGroup fell back to parentID, silently proceeding would
// mutate the SHARED real OG with no containment. It calls t.Fatalf
// immediately instead, before any mutating call is attempted.
func NewDisposableOrgGroupOrFatal(t interface {
	Cleanup(func())
	Logf(string, ...interface{})
	Fatalf(string, ...interface{})
}, c *client.Client, parentID int, existingNames []string) int {
	ogID := NewDisposableOrgGroup(t, c, parentID, existingNames)
	if ogID == parentID {
		t.Fatalf("NewDisposableOrgGroupOrFatal: disposable-OG create failed and fell back to the shared OG %d -- refusing to run a mutating test against the shared/real OG (see NewDisposableOrgGroup's log line above for the underlying create error)", parentID)
	}
	return ogID
}

// NewDisposableOrgGroupWithUUIDOrFatal is NewDisposableOrgGroupOrFatal's
// UUID-returning variant, for a MUTATE-TIER caller whose request body
// scopes by OG UUID rather than OG ID (e.g.
// InternalAppChunkTransactionV1Model.OrganizationGroupUUID). Same
// never-proceed-against-the-shared-OG contract: it calls CreateDisposableOrgGroup
// directly (skipping NewDisposableOrgGroup's fallback-to-parentID path,
// which has no UUID to return) and fails loudly via t.Fatalf on any create
// error instead of returning a value the caller could mistake for a real
// disposable OG.
func NewDisposableOrgGroupWithUUIDOrFatal(t interface {
	Cleanup(func())
	Logf(string, ...interface{})
	Fatalf(string, ...interface{})
}, c *client.Client, parentID int, existingNames []string) (int, string) {
	name := NextSdkTestRunName(existingNames)
	id, uuid, err := CreateDisposableOrgGroup(t, c, parentID, name)
	if err != nil {
		t.Fatalf("NewDisposableOrgGroupWithUUIDOrFatal: create %q under parent %d failed: %v -- refusing to run a mutating test against the shared/real OG", name, parentID, err)
	}
	return id, uuid
}

// DiscoverSdkTestRunNames walks the live org-group tree under parentID and
// returns the existing sdktestrunN child names. Confirmed against a live
// tenant (Task 1.2.10 D6 probe):
//   - GET /api/system/groups/{parentID} (Accept application/json;version=2)
//     returns PascalCase {Id, Uuid, Name, ...} -- resp["Uuid"] is the group's
//     UUID needed for the tree call.
//   - GET /api/system/groups/{uuid}/tree (Accept application/json;version=2)
//     returns a DIFFERENT, lowercase shape: {uuid, name, type, full_name,
//     group_id, children, id} -- children is a []interface{} of maps each
//     with a lowercase "name" key. This is the parsing shape
//     ParseSdkTestRunNamesFromTree expects; do not assume the two GETs share
//     a casing convention.
//
// The capture tooling's disposable-org-group creation calls this; the live
// test suite is the intended test-suite caller.
func DiscoverSdkTestRunNames(ctx context.Context, c *client.Client, parentID int) ([]string, error) {
	var parent map[string]interface{}
	if _, err := c.DoRequest(ctx, "GET", fmt.Sprintf("/api/system/groups/%d", parentID), "application/json;version=2", OrgGroupContentType, nil, &parent); err != nil {
		return nil, fmt.Errorf("DiscoverSdkTestRunNames: get parent %d: %w", parentID, err)
	}
	parentUUID, _ := parent["Uuid"].(string)
	if parentUUID == "" {
		return nil, fmt.Errorf("DiscoverSdkTestRunNames: parent %d response had no Uuid field", parentID)
	}

	var tree map[string]interface{}
	if _, err := c.DoRequest(ctx, "GET", "/api/system/groups/"+parentUUID+"/tree", "application/json;version=2", OrgGroupContentType, nil, &tree); err != nil {
		return nil, fmt.Errorf("DiscoverSdkTestRunNames: get tree for %s: %w", parentUUID, err)
	}
	return ParseSdkTestRunNamesFromTree(tree), nil
}

// ParseSdkTestRunNamesFromTree extracts the child names that start with
// SdkTestRunPrefix from a GET .../tree response. It does not check that the
// suffix is numeric (NextSdkTestRunName does), so a name such as
// "sdktestrun-prod" is returned too. Empirically confirmed shape uses lowercase
// "children"/"name" keys (see DiscoverSdkTestRunNames's doc comment), but
// this tolerates a PascalCase "Children"/"Name" variant too -- the sibling
// single-group GET uses PascalCase, and only the lowercase tree shape has
// been directly observed live, not proven stable across every tenant/version.
func ParseSdkTestRunNamesFromTree(tree map[string]interface{}) []string {
	children, ok := tree["children"].([]interface{})
	if !ok {
		children, ok = tree["Children"].([]interface{})
		if !ok {
			return nil
		}
	}
	var names []string
	for _, ch := range children {
		m, ok := ch.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := m["name"].(string)
		if !ok {
			name, ok = m["Name"].(string)
		}
		if !ok || !strings.HasPrefix(name, SdkTestRunPrefix) {
			continue
		}
		names = append(names, name)
	}
	return names
}
