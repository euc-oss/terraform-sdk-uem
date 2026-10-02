package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	mamv1 "github.com/euc-oss/terraform-sdk-uem/internal/mam/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// macOsAppAsset describes one asset's worth of inputs and expected findings
// for a macOS internal-app create/get/delete lifecycle run against the mock
// server. See runMacOsAppsV1Lifecycle for the shared drive logic this feeds.
//
// The orgGroupID field matters beyond being "the org group to call with": both macOS
// lifecycle tests load the FULL ../testdata/mock-responses tree (unlike the
// apk/msi sibling in generated_internalappsv1_lifecycle_test.go, which loads
// an isolated per-platform fixtureDir) because they also need the shared,
// platform-agnostic blobs/upload_blob_v2.json fixture. Loading the full tree
// means every asset's CREATE fixture is visible to every asset's create
// call, and internal/mockserver's RequestMatcher scores an exact-literal
// path match above a template-path match (see matcher.go's pathMatchScore).
// Two assets sharing the same orgGroupID with only one of them baking a
// literal org-group id into its fixture's endpoint would silently collide:
// the literal fixture wins regardless of which asset's create call actually
// fired. Each asset here MUST use its own
// distinguishing orgGroupID, matching the literal id baked into its own
// create fixture's endpoint.
type macOsAppAsset struct {
	orgGroupID int

	appFileName     string
	appContent      []byte
	iconFileName    string
	iconContent     []byte
	pkginfoFileName string
	pkginfoContent  []byte
	version         string

	// wantLocationID is the app id baked into this asset's OWN create
	// fixture's Location header. Asserting this exact value -- not just
	// "the Location header parses as an int" -- is what proves this
	// asset's own create fixture was served rather than a sibling's; see
	// the orgGroupID doc comment above for why that is not automatic.
	wantLocationID int

	// capturedAppID is the literal, anonymized app id baked into this
	// asset's GET/DELETE fixture paths (see those fixtures' own doc
	// comments) -- deliberately NOT the id returned by create.
	capturedAppID int

	wantPkginfoContains    string
	wantPkginfoNotContains string // "" to skip
}

// runMacOsAppsV1Lifecycle drives 3 Blobs V2 uploads -> CreateMacOSApplication
// -> InternalAppsV1 GET/DELETE against an isolated mock server for one
// macOS asset, mirroring the shape of runInternalAppsV1Lifecycle in
// generated_internalappsv1_lifecycle_test.go. See TestGeneratedMacOsAppsV1Lifecycle
// and TestGeneratedMacOsAppsV1Lifecycle_WS1Assist for the asset-specific
// documentation (provenance, real-wire findings) this drives.
func runMacOsAppsV1Lifecycle(t *testing.T, asset macOsAppAsset) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	ctx := context.Background()

	// 1. Upload 3 blobs — POST /api/mam/blobs/uploadblob → 200. All three
	// hit the same shared, platform-agnostic fixture (blobs/upload_blob_v2.json);
	// the live capture produced genuinely distinct blob ids for each of the
	// three real uploads, but the mock fixture is a single canned response,
	// which is sufficient to exercise the SDK call shape.
	blobSvc := sdk.NewBlobsV2Service(c)

	_, appBlob, err := blobSvc.UploadBlobAsync(ctx, asset.appContent, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            asset.appFileName,
		OrganizationGroupID: asset.orgGroupID,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync (app) failed: %v", err)
	}
	_, iconBlob, err := blobSvc.UploadBlobAsync(ctx, asset.iconContent, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            asset.iconFileName,
		OrganizationGroupID: asset.orgGroupID,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync (icon) failed: %v", err)
	}
	_, pkginfoBlob, err := blobSvc.UploadBlobAsync(ctx, asset.pkginfoContent, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            asset.pkginfoFileName,
		OrganizationGroupID: asset.orgGroupID,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync (pkginfo) failed: %v", err)
	}
	if appBlob.Value == nil || iconBlob.Value == nil || pkginfoBlob.Value == nil {
		t.Fatal("expected non-nil Value (int blob id) on all 3 uploads -- CreateMacOSApplication's request model requires *int, not the uuid string Get/Head/Delete use")
	}

	// 2. Create — POST /api/mam/groups/{id}/macos/apps → real response is a
	// 2xx with an EMPTY body; the new app id is only in the Location header.
	macSvc := sdk.NewMacOsAppsV1Service(c)
	headers, err := macSvc.CreateMacOSApplication(ctx, asset.orgGroupID, &sdk.MacOsCreateApplicationRequestV1Model{
		ApplicationBlobID: appBlob.Value,
		ApplicationIconID: iconBlob.Value,
		PkgInfoBlobID:     pkginfoBlob.Value,
		Version:           asset.version,
	})
	if err != nil {
		t.Fatalf("CreateMacOSApplication failed: %v", err)
	}
	// The fabricated response-body schema was removed from this
	// endpoint's overlay, so CreateMacOSApplication now returns
	// (http.Header, error) -- there is no int return to assert against. The
	// new app id is exclusively available via the Location header below.
	if headers == nil {
		t.Fatal("expected non-nil headers from CreateMacOSApplication")
	}
	location := headers.Get("Location")
	if location == "" {
		t.Fatal("expected non-empty Location header carrying the real new app id")
	}
	idxSlash := strings.LastIndex(location, "/")
	if idxSlash == -1 {
		t.Fatalf("Location header %q has no path segment to parse an id from", location)
	}
	gotLocationID, err := strconv.Atoi(location[idxSlash+1:])
	if err != nil {
		t.Fatalf("could not parse an integer app id out of Location header %q: %v", location, err)
	}
	// Asserting the EXACT expected id (not just "it parsed") is what proves
	// THIS asset's own create fixture was served -- see the orgGroupID doc
	// comment on macOsAppAsset for the collision this guards against.
	if gotLocationID != asset.wantLocationID {
		t.Errorf("Location header %q parsed to id %d, want %d (own create fixture not served -- got a different asset's fixture instead)", location, gotLocationID, asset.wantLocationID)
	}

	// 3. Get — GET /api/mam/apps/internal/{applicationId} → 200. Uses a
	// separate, literal, anonymized app id baked into the fixture path
	// rather than the id parsed from Location above -- deliberately, same
	// pattern as the apk/msi lifecycle test: this fixture holds the real
	// platform-specific captured data for a DIFFERENT app instance than the
	// one this mock "create" call pretends to have made.
	internalSvc := mamv1.NewInternalAppsV1Service(c)
	_, got, err := internalSvc.GetInternalAppByIdAsync(ctx, asset.capturedAppID)
	if err != nil {
		t.Fatalf("GetInternalAppByIdAsync failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetInternalAppByIdAsync returned nil app")
	}
	if got.Platform != "AppleOsX" {
		t.Errorf("Platform: got %q, want %q", got.Platform, "AppleOsX")
	}
	if got.UUID == "" {
		t.Error("expected non-empty uuid in get response")
	}
	if got.MacOsSoftwareDeploymentSummary == nil {
		t.Fatal("expected non-nil MacOsSoftwareDeploymentSummary -- macOS-specific field not exercised by apk/msi")
	}
	if got.MacOsSoftwareDeploymentSummary.Pkginfo == "" {
		t.Error("expected non-empty Pkginfo content")
	}
	if asset.wantPkginfoContains != "" && !strings.Contains(got.MacOsSoftwareDeploymentSummary.Pkginfo, asset.wantPkginfoContains) {
		t.Errorf("expected the pkginfo content to contain %q", asset.wantPkginfoContains)
	}
	if asset.wantPkginfoNotContains != "" && strings.Contains(got.MacOsSoftwareDeploymentSummary.Pkginfo, asset.wantPkginfoNotContains) {
		t.Errorf("expected the pkginfo content to NOT contain %q -- fixture drifted from the real captured asset", asset.wantPkginfoNotContains)
	}

	// 4. Delete — DELETE /api/mam/apps/internal/{applicationid} → 204
	// (teardown; this call also served as live-tenant cleanup during
	// capture, confirmed gone via a follow-up GET returning 401/errorCode
	// 7000 on the real tenant). See the fixture's REAL-WIRE FINDING note on
	// the blob cascade-delete behavior this call triggers live, which this
	// mock replay does not attempt to simulate.
	if _, err := internalSvc.DeleteInternalAppAsync(ctx, asset.capturedAppID); err != nil {
		t.Fatalf("DeleteInternalAppAsync failed: %v", err)
	}

	// 5. Delete the pkginfo blob explicitly -- the one blob confirmed NOT
	// cascade-deleted by the app delete on the real tenant. Uses the mock
	// blob uuid from the shared upload fixture (Get/Head/Delete key off
	// uuid, not Value -- a finding from the sensors/blobs V2 verification batch).
	if pkginfoBlob.UUID == "" {
		t.Fatal("expected non-empty uuid on pkginfo blob upload response")
	}
	if _, err := blobSvc.Delete(ctx, pkginfoBlob.UUID); err != nil {
		t.Fatalf("Delete (pkginfo blob) failed: %v", err)
	}
}

// TestGeneratedMacOsAppsV1Lifecycle verifies the macOS internal-app lifecycle:
// 3 Blobs V2 uploads (application binary, icon, pkginfo) ->
// MacOsAppsV1Service.CreateMacOSApplication -> InternalAppsV1Service GET/DELETE.
// Re-captured live against UEM 26.2.1614.12 using the fully
// synthetic vendored MacOSDMGTestApp.dmg + MacOSDMGTestApp-1.0.plist
// (chore/vendored-sdk-test-apps), replacing the original 2026-07-31 capture's
// real Chrome DMG + its shipped munki pkginfo plist -- to remove real-
// vendor-name residue from the committed fixture. Verified the synthetic swap passes live
// FIRST (fail-honest) via TestLiveMacOsAppsV1Lifecycle/DMG before touching
// this mock fixture -- see that test's doc comment.
//
// ARCHITECTURE NOTE — why this is a different shape from the apk/msi lifecycle
// test (generated_internalappsv1_lifecycle_test.go): macOS apps do NOT use
// the chunked-upload flow at all. They require 3 independent Blobs V2
// uploads first (application binary, icon, pkginfo), then a single metadata
// POST to MacOsAppsV1Service.CreateMacOSApplication. Unlike the apk/msi
// create path, this create endpoint is NOT intercepted by mockserver's
// ChunkHandler, so a normal per-directory fixture (internal_app_create_macos.json)
// is servable here and IS exercised below, not just kept as documentation.
//
// REAL-WIRE FINDING (confirmed SDK/codegen defect, now FIXED): the
// real CreateMacOSApplication response has an EMPTY body (Content-Length: 0)
// -- the new application id is only available via the response's Location
// header, never the body. This was originally captured live as a silent
// wrong-answer defect: client.Client.handleResponse's decode guard skipped
// json.Unmarshal on the empty body, so a typed int return silently stayed 0
// on a genuinely successful create, contradicting the "create returns a bare
// integer" doctrine (api-doctrine.md quirk 6). This was fixed at the
// source (removed the overlay's fabricated response-body schema), so
// CreateMacOSApplication no longer returns an int at all -- see the
// doc comment above the create call in runMacOsAppsV1Lifecycle explaining
// the removed schema.
// This test now asserts the correct, permanent shape (non-nil headers,
// Location header parseable to the expected id) instead of pinning the old
// defect. See internal_app_create_macos.json for the full live-capture writeup.
//
// REAL-WIRE FINDING (teardown): deleting the macOS app cascade-deletes its
// attached application-binary and icon blobs automatically -- confirmed live
// by a follow-up GET/HEAD on both blobs returning 500 (this tenant's
// established Blobs V2 not-found signature, see blobs/delete_blob_v2.json)
// immediately after the app delete, with no separate blob-delete call made
// against them. The third blob (pkginfo) was NOT cascade-deleted and
// required its own explicit BlobsV2Service.Delete call. See
// delete_internal_app_macos.json for the full writeup. This mock test does
// not attempt to simulate the cascade (mockserver's stateless fixture
// matcher has no notion of it) -- it documents the finding in comments and
// exercises the explicit pkginfo-blob delete only, matching what the live
// capture actually had to do.
//
// Fixtures: testdata/mock-responses/internal-apps-v1/macos/{get,delete,internal_app_create}_internal_app_macos.json,
// testdata/mock-responses/blobs/upload_blob_v2.json (shared, platform-agnostic).
func TestGeneratedMacOsAppsV1Lifecycle(t *testing.T) {
	runMacOsAppsV1Lifecycle(t, macOsAppAsset{
		// orgGroupID matches internal_app_create_macos.json's literal
		// endpoint path -- this asset was the FIRST captured, so it holds
		// the "plain" org group id; the PKGStyle asset below deliberately
		// uses a distinct one (see macOsAppAsset's orgGroupID doc comment).
		orgGroupID: 12345,

		appFileName: "MacOSDMGTestApp.dmg",
		// appContent is intentionally throwaway inline bytes, not the real
		// vendored MacOSDMGTestApp.dmg: mockserver's RequestMatcher matches
		// on request path/filename only and never inspects upload content,
		// so mock mode gets no benefit from reading the real file. Only
		// TestLiveMacOsAppsV1Lifecycle (live_resource_lifecycle_test.go)
		// reads the real vendored binaries, since that's what a live
		// server's create actually processes.
		appContent:      []byte("fake dmg bytes"),
		iconFileName:    "op3g_icon.png",
		iconContent:     []byte("fake png bytes"),
		pkginfoFileName: "op3g_pkginfo.plist",
		pkginfoContent:  []byte("fake pkginfo plist bytes"),
		version:         "1.0.0",

		wantLocationID: 12357,
		capturedAppID:  67003,

		wantPkginfoContains: "com.omnissa.sdktest.macosdmg",
	})
}

// TestGeneratedMacOsAppsV1Lifecycle_WS1Assist verifies the same macOS
// internal-app lifecycle using a second, independent asset.
// Re-captured live against UEM 26.2.1614.12 using the fully
// synthetic vendored MacOSPkgTestApp-1.0.pkg + MacOSPkgStyleTestApp-1.0.plist
// (chore/vendored-sdk-test-apps), replacing the original 2026-07-31 capture's
// real WS1 Assist .pkg installer + its shipped Munki plist -- to remove
// real-vendor-name residue from the committed fixture. The test function name
// is kept for historical continuity/traceability with the original
// capture and cross-references in tests/verified-endpoints.json;
// the asset itself is no longer WS1 Assist-derived.
//
// Purpose: this asset exercises the pkg-install path (no type-of-installer
// key in the pkginfo, Munki's native-pkg-install default) versus the DMG
// asset's explicit copy_from_dmg -- confirming, as expected from the
// canonical source review before this capture, that CreateMacOSApplication
// is the SAME endpoint/controller for both; the omitted key is purely a
// client-side (Munki agent) install-behavior hint the server never
// inspects.
//
// REAL-WIRE FINDING: the empty-create-body defect (id only in
// Location header -- since fixed in the overlay, see runMacOsAppsV1Lifecycle)
// reproduced identically on this second, independent binary+pkginfo pairing
// -- confirms it was a property of the endpoint itself, not asset-specific.
//
// REAL-WIRE FINDING (teardown): the cascade-delete behavior from the DMG
// asset reproduced exactly -- deleting the app auto-removed the pkg and
// icon blobs (confirmed via live HEAD returning 500 on both immediately
// after, no explicit delete call made against them); only the pkginfo blob
// needed its own explicit delete. See delete_internal_app_macos_ws1assist.json.
//
// Fixtures: testdata/mock-responses/internal-apps-v1/macos/{get,delete,internal_app_create}_internal_app_macos_ws1assist.json
// (filenames kept for historical continuity; content is now fully synthetic)
//
// orgGroupID uses a distinct literal (67004, matching this asset's own
// GET/DELETE app id family) rather than the DMG asset's 12345: both tests
// load the full ../testdata/mock-responses tree, and internal/mockserver's
// RequestMatcher scores an exact-literal path match above a template-path
// match, so this asset's create fixture MUST bake in its own literal
// org-group id (internal_app_create_macos_ws1assist.json's endpoint) or the
// DMG asset's literal "/api/mam/groups/12345/macos/apps" fixture would win
// the match for both tests' create calls.
func TestGeneratedMacOsAppsV1Lifecycle_WS1Assist(t *testing.T) {
	runMacOsAppsV1Lifecycle(t, macOsAppAsset{
		orgGroupID: 67004,

		appFileName: "MacOSPkgTestApp-1.0.pkg",
		// appContent is intentionally throwaway inline bytes, not the real
		// vendored MacOSPkgTestApp-1.0.pkg -- same reasoning as the DMG
		// asset above (mockserver never inspects upload content; only
		// TestLiveMacOsAppsV1Lifecycle/PKGStyle reads the real file).
		appContent:      []byte("fake pkg bytes"),
		iconFileName:    "op3g_icon2.png",
		iconContent:     []byte("fake png bytes"),
		pkginfoFileName: "MacOSPkgStyleTestApp-1.0.plist",
		pkginfoContent:  []byte("fake pkginfo plist bytes"),
		version:         "1.0.0",

		wantLocationID: 67004,
		capturedAppID:  67004,

		wantPkginfoContains:    "com.omnissa.sdktest.macospkgstyle",
		wantPkginfoNotContains: "installer_type",
	})
}
