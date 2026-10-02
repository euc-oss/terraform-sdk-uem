package tests

import (
	"bytes"
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	mamv1 "github.com/euc-oss/terraform-sdk-uem/internal/mam/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedInternalAppsV1Lifecycle_APK and its MSI sibling below verify
// the GET and DELETE internal-app operations,
// live-captured against UEM 26.2.0.0, for the apk and
// msi platforms. (iOS/ipa was attempted in the same live-capture pass but
// blocked at create by a genuinely expired provisioning profile embedded in
// the staged sample .ipa -- not an SDK defect, no fixture exists for it.)
//
// ARCHITECTURE NOTE — why create is not platform-specific here: internal/
// mockserver's ChunkHandler.handleInternalApplication now CAN select a
// platform-specific create fixture (see
// selectInternalApplicationFixture in chunk_handler.go), keyed on the
// create request's file_name extension -- testdata/mock-responses/
// internal-apps-v1/{apk,msi}/internal_app_create_*.json are wired in and no
// longer documentation-only (see TestChunkHandler_InternalApplication_
// ApkFileName_ServesApkFixture / _MsiFileName_ServesMsiFixture in
// internal/mockserver). These generated lifecycle tests were left calling
// CreateInternalApplication without a FileName, deliberately continuing to
// exercise the pre-existing shared fixture (already covered by
// TestChunkedUploadEndToEnd_CreateInternalApplication) rather than
// widening this change's scope to also rewrite their create step; their
// platform-specific, newly-verified assertions stay on GET and DELETE,
// which route through the normal per-directory fixture matcher.
//
// EMPIRICAL REQUIRED-FIELD FINDINGS from the live capture (see the two
// create fixtures above for the exact 400 bodies hit, in order): file_name,
// supported_models (validated against the real per-platform picklist from
// GET /api/mdm/picklists/platforms/{platform}/devicemodels -- "Any" was
// rejected, "Android"/"Desktop" were not), and -- msi only --
// deployment_options (all three sub-objects present; inner fields could
// stay default) and supported_processor_architecture.
//
// REAL-WIRE FINDING: InternalApplicationEntityV1.PushMode is typed string
// (matching swagger's declared enum ["Auto","OnDemand"]), but the live
// create response returns PushMode as a JSON number (0) on both platforms.
// This breaks CreateInternalApplication's typed decode against the REAL
// API even on a clean 200 (confirmed by recovering the orphaned app via a
// follow-up search); it does not surface here because the pre-existing
// shared mock fixture omits PushMode entirely, which is precisely why this
// defect was invisible until live capture.
//
// SEPARATE REAL-WIRE FINDING: on apk, the server silently overrode the
// caller-supplied bundle_id with one derived from the uploaded APK's own
// manifest (reproduced across two independent live create calls). On msi,
// no bundle_id was supplied at all and the server returned a real installer
// GUID instead (see the private ledger, tests/verified-endpoints.json, for
// provenance).
//
// POST-DELETE FINDING (both platforms): a follow-up GET after DELETE
// returned a consistent HTTP 401 with body {"errorCode":7000,"message":
// "Application not found or user does not have access to it."} rather than
// a plain 404 -- an unusual but repeatable not-found signature.
func TestGeneratedInternalAppsV1Lifecycle_APK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	runInternalAppsV1Lifecycle(t, "../testdata/mock-responses/internal-apps-v1/apk", 67001, "Android")
}

func TestGeneratedInternalAppsV1Lifecycle_MSI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	runInternalAppsV1Lifecycle(t, "../testdata/mock-responses/internal-apps-v1/msi", 67002, "WinRT")
}

// TestGeneratedInternalAppsV1Lifecycle_IPA is the iOS sibling of the apk/msi
// tests above, unblocked on 2026-08-01: a
// correctly-built HelloMDMTest.ipa (valid DTPlatformName + unexpired
// provisioning profile, see vendored-test-apps/ios/SPEC.md) completed the
// full create->get->delete lifecycle live against UEM 26.2.1614.12,
// unblocking the create that batch 5 above hit an expired-provisioning-
// profile 400 on. Fixtures are real, live-captured data (get_internal_app_ipa
// .json / delete_internal_app_ipa.json), anonymized before commit -- NOT
// synthetic, unlike the pkg/zip siblings below. The real live-captured id
// (65653) is used directly as appID here (not remapped to a fresh literal
// like the apk/msi/67001/67002 fixtures) since it was already anonymized at
// capture time and doesn't collide with any other literal id in this tree.
func TestGeneratedInternalAppsV1Lifecycle_IPA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	runInternalAppsV1Lifecycle(t, "../testdata/mock-responses/internal-apps-v1/ipa", 65653, "Apple")
}

// TestGeneratedInternalAppsV1Lifecycle_PKG exercises the PLAIN InternalAppsV1
// chunked-upload create path for macOS .pkg -- device_type AppleOsX, file_name
// ending .pkg -- which is DISTINCT from and does not touch the full-SLM
// MacOsAppsV1Service .pkg/.dmg coverage in generated_macosappsv1_lifecycle_test.go
// (that path uploads 3 independent blobs via BlobsV2Service, never uses
// chunked-upload at all). Fixtures in testdata/mock-responses/internal-apps-v1/pkg
// are REAL, live-captured (UEM 26.2.1614.12) via
// tools/internalapps-capture against the vendored MacOSPkgTestApp-1.0.pkg mock,
// then anonymized via mock-capture's Anonymize() -- see each fixture file's own
// metadata.description for details. This live run also confirmed the
// AppleOsX/"MacBook" device-model picklist value (see platform.go's
// knownPlatformDefaults) and, as a byproduct, the client.APIError numeric
// errorCode wire-shape defect (fixed in client/client.go's APIError.UnmarshalJSON).
func TestGeneratedInternalAppsV1Lifecycle_PKG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	runInternalAppsV1Lifecycle(t, "../testdata/mock-responses/internal-apps-v1/pkg", 65679, "AppleOsX")
}

// TestGeneratedInternalAppsV1Lifecycle_ZIP exercises the Windows .zip
// (Software Distribution) chunked-upload create path -- a separate, thinner
// path than .msi that additionally requires FilesOptions.ApplicationUnInstallProcess
// .CustomScript (mandatory for .zip/.exe, NOT for .msi -- vendored-test-apps/
// windows/SPEC.md section 9). Fixtures are SYNTHETIC / hand-authored
// PLACEHOLDERS, NOT live-captured. Live .zip capture is explicitly HELD
// pending org-group Software Distribution feature-flag confirmation -- do
// not attempt a live .zip capture until that flag's status is confirmed on
// -- so this is mock-only coverage by design, not an oversight. See each
// fixture file's own metadata.description for the full rationale.
//
// Unlike the apk/msi/ipa/pkg siblings above, this one is driven by
// runInternalAppsV1LifecycleBypass rather than runInternalAppsV1Lifecycle:
// the 3 fixtures under the vendored tree's internal-apps-v1/zip/ are
// themselves self-labeled synthetic placeholders (live .zip capture never
// happened, held pending the target org group's SFD flag being confirmed
// ON), so this test intentionally
// loads them via the bypass mechanism (newMockServerFromBypassFixturesOnly,
// see tests/bypass_fixture_helper_test.go) from their internal
// synthetic-placeholder location (internal/mockserver/testdata/
// synthetic-placeholders/internal-apps-v1/zip/), independent of
// MOCK_RESPONSES_DIR, for the same consistency reasons the other
// synthetic-placeholder twins in this corpus use that location.
func TestGeneratedInternalAppsV1Lifecycle_ZIP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	runInternalAppsV1LifecycleBypass(t, []string{
		"../internal/mockserver/testdata/synthetic-placeholders/internal-apps-v1/zip/internal_app_create_zip.json",
		"../internal/mockserver/testdata/synthetic-placeholders/internal-apps-v1/zip/get_internal_app_zip.json",
		"../internal/mockserver/testdata/synthetic-placeholders/internal-apps-v1/zip/delete_internal_app_zip.json",
	}, 67007, "WinRT")
}

// runInternalAppsV1Lifecycle drives upload -> create -> get -> delete
// against an isolated mock server built by scanning fixtureDir (the normal,
// override-aware directory scan). appID is a fixed, anonymized literal id
// (NOT the id the create step returns -- that step always returns 12345
// from the shared synthetic fixture per the architecture note above) baked
// into this platform's get/delete fixture paths as a literal path rather
// than the {applicationId}/{applicationid} template. This is deliberate:
// internal/mockserver's RequestMatcher scores an exact literal path higher
// than a template match, so a literal, distinct-per-platform id guarantees
// this fixture is never picked up by (or itself shadows) unrelated tests
// that load the full ../testdata/mock-responses tree and hit the generic
// {applicationId} template -- e.g. TestMamV1InternalAppGetByIdRenewalDateFixture,
// which this project hit as a real regression while this fixture set used
// the template form.
func runInternalAppsV1Lifecycle(t *testing.T, fixtureDir string, appID int, wantPlatform string) {
	t.Helper()
	ms := mockserver.LoadMockResponses(t, fixtureDir)
	t.Cleanup(ms.Close)
	runInternalAppsV1LifecycleWithServer(t, ms, appID, wantPlatform)
}

// runInternalAppsV1LifecycleBypass is runInternalAppsV1Lifecycle's sibling
// for a fixture set that must bypass MOCK_RESPONSES_DIR entirely (see
// TestGeneratedInternalAppsV1Lifecycle_ZIP's doc comment for why).
func runInternalAppsV1LifecycleBypass(t *testing.T, bypassPaths []string, appID int, wantPlatform string) {
	t.Helper()
	ms := newMockServerFromBypassFixturesOnly(t, bypassPaths)
	t.Cleanup(ms.Close)
	runInternalAppsV1LifecycleWithServer(t, ms, appID, wantPlatform)
}

// runInternalAppsV1LifecycleWithServer is the shared body both constructors
// above delegate to once ms is built: upload -> create -> get -> delete.
func runInternalAppsV1LifecycleWithServer(t *testing.T, ms *mockserver.MockServer, appID int, wantPlatform string) {
	t.Helper()
	c := mockserver.NewMockClient(t, ms)
	ctx := context.Background()

	// 1. Chunked upload — POST /api/mam/apps/internal/uploadchunk. Served
	// by mockserver's stateful ChunkHandler, wired into every MockServer
	// regardless of which fixture directory was loaded, so no static
	// fixture is needed for this op.
	uploader := sdk.NewChunkedUploader(c)
	const totalSize = 8 * 1024
	body := bytes.Repeat([]byte("A"), totalSize)
	session, err := uploader.Upload(ctx, bytes.NewReader(body), int64(totalSize), nil)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if session.TransactionID == "" {
		t.Fatal("session.TransactionID is empty after Upload")
	}

	// 2. Create — POST /api/mam/apps/internal/application. Always served by
	// ChunkHandler's hardcoded shared fixture (see the architecture note
	// above), so this step is platform-agnostic here by construction.
	app, err := uploader.CreateInternalApplication(ctx, session, &mamv1.InternalAppChunkTransactionV1Model{
		ApplicationName: "Forklift V1Model E2E",
	})
	if err != nil {
		t.Fatalf("CreateInternalApplication failed: %v", err)
	}
	if app.ID == nil || app.ID.Value == nil {
		t.Fatal("create response missing Id.Value")
	}
	// Deliberately NOT using the create response's id here -- see the
	// appID parameter doc comment above.

	svc := mamv1.NewInternalAppsV1Service(c)

	// 3. Get — GET /api/mam/apps/internal/{applicationId} → 200. This is
	// the platform-specific, newly-verified real capture: the fixture
	// loaded is unique to fixtureDir, so this assertion genuinely
	// distinguishes apk from msi.
	_, got, err := svc.GetInternalAppByIdAsync(ctx, appID)
	if err != nil {
		t.Fatalf("GetInternalAppByIdAsync failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetInternalAppByIdAsync returned nil app")
	}
	if got.Platform != wantPlatform {
		t.Errorf("Platform: got %q, want %q", got.Platform, wantPlatform)
	}
	if got.UUID == "" {
		t.Error("expected non-empty uuid in get response")
	}

	// 4. Delete — DELETE /api/mam/apps/internal/{applicationid} → 204
	// (teardown; this call also served as live-tenant cleanup during
	// capture, confirmed gone via a follow-up GET returning 401/errorCode
	// 7000 on the real tenant -- not reproduced here since the mock
	// matcher has no stateful "deleted" tracking for this endpoint).
	_, err = svc.DeleteInternalAppAsync(ctx, appID)
	if err != nil {
		t.Fatalf("DeleteInternalAppAsync failed: %v", err)
	}
}
