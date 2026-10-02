package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	mamv1 "github.com/euc-oss/terraform-sdk-uem/v26/internal/mam/v1"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/orgtest"
	"github.com/euc-oss/terraform-sdk-uem/v26/models"
)

// This file is the FIRST live-mode ("TEST_MODE=live") variant of a
// resource lifecycle test in this repo -- every other lifecycle test
// (generated_macosappsv1_lifecycle_test.go, generated_internalappsv1_
// lifecycle_test.go, internal_apps_chunked_upload_test.go) is mock-only.
// It is meant to be the harness's reference/proof artifact for future
// live-gated resource lifecycle tests: it reuses integration_test.go's
// getLiveClient*/env-var-reading machinery rather than duplicating it, and
// documents every real-wire behavior it depends on.
//
// Both tests below are silent no-ops in any environment that has not
// explicitly opted into TEST_MODE=live: they t.Skip() (never t.Fail()) the
// instant that gate is missing, before touching the network in any way.
// They also respect `go test -short` like every other integration-style
// test in this package.

// liveLifecycleTimeoutEnvVar overrides the generous default HTTP timeout
// used by the live resource lifecycle tests in this file. Real file
// uploads and multi-step create/get/delete sequences against a live tenant
// can legitimately take much longer than the short, read-only checks in
// integration_test.go (which use getLiveClient's 30s default) -- so these
// tests get their own, larger default and their own override knob rather
// than sharing that one.
const liveLifecycleTimeoutEnvVar = "LIVE_LIFECYCLE_TIMEOUT_SECONDS"

// defaultLiveLifecycleTimeout is a deliberately generous, round default --
// NOT a value pulled from any specific past live-capture run -- chosen to
// give real uploads plus a multi-step lifecycle (upload(s) -> create -> get
// -> delete -> verify) comfortable headroom over a live network. Set
// LIVE_LIFECYCLE_TIMEOUT_SECONDS if a given environment needs more (e.g. a
// slow link or a much larger real asset than the tiny synthetic payloads
// used here).
const defaultLiveLifecycleTimeout = 10 * time.Minute

// liveLifecycleTimeout resolves the HTTP client timeout for this file's
// live tests from LIVE_LIFECYCLE_TIMEOUT_SECONDS, falling back to
// defaultLiveLifecycleTimeout.
func liveLifecycleTimeout(t *testing.T) time.Duration {
	t.Helper()
	v := os.Getenv(liveLifecycleTimeoutEnvVar)
	if v == "" {
		return defaultLiveLifecycleTimeout
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		t.Fatalf("%s=%q must be a positive integer number of seconds", liveLifecycleTimeoutEnvVar, v)
	}
	return time.Duration(secs) * time.Second
}

// liveOrgGroupID resolves the organization group id these live lifecycle
// tests create resources under, from the same ORG_GROUP_ID env var
// convention already established by TestIntegration_ProfileService_FullCRUD
// (profile_crud_test.go) -- reusing getEnvOrDefault from that file rather
// than introducing a second env-var-reading path for the same setting.
func liveOrgGroupID(t *testing.T) int {
	t.Helper()
	raw := getEnvOrDefault("ORG_GROUP_ID", "12345")
	id, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("ORG_GROUP_ID=%q is not a valid integer: %v", raw, err)
	}
	return id
}

// TestAppsDirEnv is the environment variable that overrides the vendored
// synthetic test-app binaries directory used by
// TestLiveInternalAppsV1ChunkedUploadLifecycle below. Mirrors internal/
// mockserver's MOCK_RESPONSES_DIR/ResolveResponsesDir pattern exactly
// (internal/mockserver/loader.go:16) -- overridable, defaults to a fixed
// local directory if unset -- so CI, or a differently-laid-out local
// checkout, can point elsewhere without editing this file.
const TestAppsDirEnv = "TEST_APPS_DIR"

// defaultTestAppsDir points at `make vendor-test-apps`'s own TEST_APPS_STAGING
// staging directory (see Makefile's vendor-test-apps target), which is
// created relative to the REPO ROOT. `go test` runs with its working
// directory set to the package under test (this file's package is `tests`,
// i.e. <repo-root>/tests), not the repo root or the caller's shell cwd --
// so this default must be "../vendored-test-apps", not "vendored-test-apps",
// or every unset-env-var run fails with a spurious "no such file or
// directory" regardless of whether `make vendor-test-apps` was actually run.
const defaultTestAppsDir = "../vendored-test-apps"

// resolveTestAppsDir returns the directory the live internal-app lifecycle
// test reads its per-platform binaries from. Not needed by (and does not
// affect) any mock-mode test: those use static JSON fixtures under
// testdata/mock-responses, never real binaries, so CI's mock-only test run
// never touches this directory or this env var.
func resolveTestAppsDir() string {
	if override := strings.TrimSpace(os.Getenv(TestAppsDirEnv)); override != "" {
		return override
	}
	return defaultTestAppsDir
}

// pickLiveAuthMethod chooses which of getLiveClient's two supported auth
// methods to use, based solely on which credential pair is present --
// CLIENT_ID/CLIENT_SECRET selects "oauth2", otherwise "basic" is requested
// (getLiveClientWithTimeout itself still does the actual required-field
// checking and graceful t.Skip if neither pair is fully populated; this
// helper does not duplicate that logic, it only picks which branch to ask
// for).
func pickLiveAuthMethod() string {
	if os.Getenv("CLIENT_ID") != "" && os.Getenv("CLIENT_SECRET") != "" {
		return "oauth2"
	}
	return "basic"
}

// skipUnlessLive is the shared gate for every test in this file: skip
// (never fail) in short mode, and skip (never fail) unless the caller has
// explicitly opted in with TEST_MODE=live. Actual credential presence
// (INSTANCE_URL, TENANT_CODE, and either USERNAME/PASSWORD or
// CLIENT_ID/CLIENT_SECRET) is still checked, and gracefully skipped on,
// inside getLiveClientWithTimeout -- this function only handles the
// TEST_MODE switch itself, since getTestClient's default ("mock") would
// otherwise silently run these tests against the mock server instead of
// skipping them, which is never what we want for a live-only test.
func skipUnlessLive(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping live integration test in short mode")
	}
	if os.Getenv("TEST_MODE") != "live" {
		t.Skip("TEST_MODE=live not set; skipping live resource lifecycle test")
	}
}

// liveMacOsFullSlmCase describes one full-SLM asset pair for
// TestLiveMacOsAppsV1Lifecycle: which vendored application binary + companion
// Admin-Assistant pkginfo plist to upload. Both fields in every case are
// REAL, vendored, synthetic files (chore/vendored-sdk-test-apps via
// resolveTestAppsDir()) -- NOT real vendor software and NOT inline
// throwaway bytes. This mirrors generated_macosappsv1_lifecycle_test.go's
// two mock subtests (Chrome-style .dmg, WS1-Assist-style .pkg) but with
// fully synthetic sources on both the live and mock side, to remove
// real-vendor-name residue (com.google.Chrome / Workspace ONE Assist)
// from the committed fixture.
type liveMacOsFullSlmCase struct {
	name           string // subtest name, e.g. "DMG"
	appRelPath     string // relative to resolveTestAppsDir(), the application binary
	pkginfoRelPath string // relative to resolveTestAppsDir(), the companion plist
}

var liveMacOsFullSlmCases = []liveMacOsFullSlmCase{
	{
		name:           "DMG",
		appRelPath:     filepath.Join("macos", "MacOSDMGTestApp.dmg"),
		pkginfoRelPath: filepath.Join("macos", "MacOSDMGTestApp-1.0.plist"),
	},
	{
		// PKGStyle mirrors generated_macosappsv1_lifecycle_test.go's
		// WS1-Assist subtest -- same full-SLM endpoint, same "native
		// pkg-install, no installer_type key" Munki style, but both the
		// binary (vendored MacOSPkgTestApp-1.0.pkg) and the companion plist
		// (MacOSPkgStyleTestApp-1.0.plist) are synthetic, replacing the real
		// WS1 Assist product name/version.
		name:           "PKGStyle",
		appRelPath:     filepath.Join("macos", "MacOSPkgTestApp-1.0.pkg"),
		pkginfoRelPath: filepath.Join("macos", "MacOSPkgStyleTestApp-1.0.plist"),
	},
}

// TestLiveMacOsAppsV1Lifecycle is the TEST_MODE=live counterpart to
// TestGeneratedMacOsAppsV1Lifecycle (generated_macosappsv1_lifecycle_test.go).
// It drives the real macOS internal-app create lifecycle end to end against
// a live Workspace ONE UEM tenant, once per liveMacOsFullSlmCases entry:
//
//  1. 3 independent BlobsV2Service.UploadBlobAsync calls (application
//     binary, icon, pkginfo) -- macOS apps do NOT use the chunked-upload
//     flow at all, unlike TestLiveInternalAppsV1ChunkedUploadLifecycle below.
//  2. MacOsAppsV1Service.CreateMacOSApplication.
//  3. InternalAppsV1Service.GetInternalAppByIdAsync to confirm the app
//     round-trips with macOS-specific fields populated.
//  4. Teardown: InternalAppsV1Service.DeleteInternalAppAsync, plus an
//     explicit BlobsV2Service.Delete of the pkginfo blob.
//
// The application binary and pkginfo plist are REAL vendored synthetic
// files (see liveMacOsFullSlmCase's doc comment) -- confirmed live-verified
// per platform (UEM 26.2.1614.12) before this comment was
// written; the icon stays a tiny inline placeholder since icon content is
// never validated (vendored-test-apps/macos/SPEC.md).
//
// REAL-WIRE BEHAVIORS this test's assertions are pinned to (see
// api-doctrine.md and generated_macosappsv1_lifecycle_test.go's doc
// comments for the fuller writeups from the live sessions that discovered
// them):
//
//   - CreateMacOSApplication's response body is EMPTY on success
//     (Content-Length: 0) -- the new application id is only
//     available via the response's Location header. Do not expect a typed
//     body here; that overlay defect was removed at the source, and this
//     endpoint's Go signature reflects that (returns (http.Header, error),
//     no response struct).
//   - Deleting the created app cascade-deletes its application-binary and
//     icon blobs automatically. It does NOT cascade-delete the pkginfo
//     blob, which needs its own explicit BlobsV2Service.Delete call --
//     included below as teardown, registered via t.Cleanup so it still
//     runs if an earlier assertion in this test fails.
func TestLiveMacOsAppsV1Lifecycle(t *testing.T) {
	skipUnlessLiveMutate(t)

	for _, tc := range liveMacOsFullSlmCases {
		t.Run(tc.name, func(t *testing.T) {
			runLiveMacOsAppsV1Lifecycle(t, tc)
		})
	}
}

func runLiveMacOsAppsV1Lifecycle(t *testing.T, tc liveMacOsFullSlmCase) {
	c := getLiveClientWithTimeout(t, pickLiveAuthMethod(), liveLifecycleTimeout(t))
	ctx := context.Background()

	// orgGroupID is a disposable child OG created under liveOrgGroupID(t),
	// not the shared tenant-wide OG itself: this test mutates a real tenant (blob uploads + a macOS app
	// create), so it must never run scoped to the shared/real OG. Blobs are
	// NOT OG-tree-parented (they don't cascade-delete on OG delete, unlike
	// the app they get attached to -- see the explicit pkginfo-blob
	// t.Cleanup below), so containment here is about keeping the app itself
	// out of the shared OG's tree, not about blob cleanup, which is already
	// handled explicitly regardless of which OG is used.
	orgGroupID := orgtest.NewDisposableOrgGroupOrFatal(t, c, liveOrgGroupID(t), nil)
	stamp := time.Now().UnixNano()

	blobSvc := sdk.NewBlobsV2Service(c)
	macSvc := sdk.NewMacOsAppsV1Service(c)
	internalSvc := mamv1.NewInternalAppsV1Service(c)

	testAppsDir := resolveTestAppsDir()
	appPath := filepath.Join(testAppsDir, tc.appRelPath)
	appContent, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatalf("open vendored test-app binary %s: %v (set %s to override the default %q, or run `make vendor-test-apps` first)", appPath, err, TestAppsDirEnv, defaultTestAppsDir)
	}
	pkginfoPath := filepath.Join(testAppsDir, tc.pkginfoRelPath)
	pkginfoContent, err := os.ReadFile(pkginfoPath)
	if err != nil {
		t.Fatalf("open vendored pkginfo plist %s: %v (set %s to override the default %q, or run `make vendor-test-apps` first)", pkginfoPath, err, TestAppsDirEnv, defaultTestAppsDir)
	}
	// Icon content is never validated (vendored-test-apps/macos/SPEC.md --
	// icon resolution is wrapped in its own try/catch, only logs on
	// failure) -- a tiny inline placeholder is sufficient, no vendored icon
	// file is needed.
	iconContent := []byte(fmt.Sprintf("synthetic-icon-bytes-%d", stamp))

	appFileName := fmt.Sprintf("sdk-live-lifecycle-%s-%d%s", tc.name, stamp, filepath.Ext(tc.appRelPath))
	iconFileName := fmt.Sprintf("sdk-live-lifecycle-%s-%d.png", tc.name, stamp)
	pkginfoFileName := fmt.Sprintf("sdk-live-lifecycle-%s-%d.plist", tc.name, stamp)

	// 1. Upload the 3 required blobs.
	_, appBlob, err := blobSvc.UploadBlobAsync(ctx, appContent, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            appFileName,
		OrganizationGroupID: orgGroupID,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync (application binary) failed: %v", err)
	}
	if appBlob == nil || appBlob.Value == nil {
		t.Fatal("UploadBlobAsync (application binary): expected non-nil Value (int blob id) -- CreateMacOSApplication's request requires it")
	}

	_, iconBlob, err := blobSvc.UploadBlobAsync(ctx, iconContent, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            iconFileName,
		OrganizationGroupID: orgGroupID,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync (icon) failed: %v", err)
	}
	if iconBlob == nil || iconBlob.Value == nil {
		t.Fatal("UploadBlobAsync (icon): expected non-nil Value (int blob id)")
	}

	_, pkginfoBlob, err := blobSvc.UploadBlobAsync(ctx, pkginfoContent, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            pkginfoFileName,
		OrganizationGroupID: orgGroupID,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync (pkginfo) failed: %v", err)
	}
	if pkginfoBlob == nil || pkginfoBlob.Value == nil {
		t.Fatal("UploadBlobAsync (pkginfo): expected non-nil Value (int blob id)")
	}

	// Teardown for the pkginfo blob -- registered as soon as we have its
	// uuid, BEFORE the app-delete cleanup below is registered, so t.Cleanup's
	// LIFO ordering runs app-delete (which cascades away the app+icon blobs)
	// first and this pkginfo-blob delete second, matching the real
	// dependency order confirmed live (see the REAL-WIRE BEHAVIORS doc
	// comment above the test).
	if pkginfoBlob.UUID == "" {
		t.Fatal("UploadBlobAsync (pkginfo): expected non-empty uuid (Get/Head/Delete key off uuid, not Value)")
	}
	t.Cleanup(func() {
		if _, err := blobSvc.Delete(context.Background(), pkginfoBlob.UUID); err != nil {
			t.Logf("cleanup: BlobsV2Service.Delete(pkginfo blob %s) failed: %v", pkginfoBlob.UUID, err)
		}
	})

	// 2. Create the macOS application. Real response body is EMPTY; the new
	// app id is only available via the Location header.
	headers, err := macSvc.CreateMacOSApplication(ctx, orgGroupID, &sdk.MacOsCreateApplicationRequestV1Model{
		ApplicationBlobID: appBlob.Value,
		ApplicationIconID: iconBlob.Value,
		PkgInfoBlobID:     pkginfoBlob.Value,
		Version:           "1.0.0",
	})
	if err != nil {
		t.Fatalf("CreateMacOSApplication failed: %v", err)
	}
	if headers == nil {
		t.Fatal("CreateMacOSApplication: expected non-nil headers")
	}
	location := headers.Get("Location")
	if location == "" {
		t.Fatal("CreateMacOSApplication: expected a non-empty Location header carrying the new app id -- real response body is empty, so this is the ONLY place the id is available")
	}
	locID, err := client.ParseLocationID(location)
	if err != nil {
		t.Fatalf("CreateMacOSApplication: could not extract an id from Location header %q: %v", location, err)
	}
	appID, err := strconv.Atoi(locID)
	if err != nil {
		t.Fatalf("CreateMacOSApplication: Location header %q's id segment %q is not an integer: %v", location, locID, err)
	}
	if appID <= 0 {
		t.Fatalf("CreateMacOSApplication: parsed a non-positive app id %d from Location header %q", appID, location)
	}

	// Teardown for the app itself -- registered as soon as we have appID,
	// AFTER the pkginfo cleanup above. t.Cleanup runs LIFO, so this
	// app-delete runs FIRST (cascading away the app+icon blobs), then the
	// pkginfo-blob delete registered above runs second -- see that
	// registration's comment for why this order matters.
	t.Cleanup(func() {
		if _, err := internalSvc.DeleteInternalAppAsync(context.Background(), appID); err != nil {
			t.Logf("cleanup: DeleteInternalAppAsync(%d) failed: %v", appID, err)
		}
	})

	// 3. Confirm the app round-trips via GET, with macOS-specific fields
	// populated.
	_, got, err := internalSvc.GetInternalAppByIdAsync(ctx, appID)
	if err != nil {
		t.Fatalf("GetInternalAppByIdAsync(%d) failed: %v", appID, err)
	}
	if got == nil {
		t.Fatal("GetInternalAppByIdAsync returned nil app")
	}
	if got.Platform != models.PlatformAppleOSX {
		t.Errorf("Platform: got %q, want %q (models.PlatformAppleOSX -- mixed case is the real wire value, not \"AppleOSX\")", got.Platform, models.PlatformAppleOSX)
	}
	if got.UUID == "" {
		t.Error("expected non-empty uuid in GetInternalAppByIdAsync response")
	}
	if got.MacOsSoftwareDeploymentSummary == nil {
		t.Fatal("expected non-nil MacOsSoftwareDeploymentSummary -- macOS-specific field, absent for other platforms")
	}
}

// liveInternalAppPlatformCase describes one platform's worth of input to
// TestLiveInternalAppsV1ChunkedUploadLifecycle: which vendored binary to
// upload, and the platform-specific request metadata the live API is
// empirically known (or, where noted, assumed) to require.
type liveInternalAppPlatformCase struct {
	// name is the subtest name (t.Run("APK", ...) etc).
	name string
	// testAppRelPath is relative to resolveTestAppsDir(), e.g.
	// "android/AndroidApkTestApp-1.0.apk".
	testAppRelPath string
	// deviceType is the mamv1.json device_type wire value -- NOT the
	// unrelated models.Platform* constants used by the profiles/devices
	// APIs, which use a different string set (e.g. models.PlatformAppleIOS
	// = "Apple iOS", where this enum uses plain "Apple"). See platform.go's
	// doc comment in tools/internalapps-capture for the same caution.
	deviceType string
	// modelName is the supported_models[].model_name device-model picklist
	// value. "Any" is empirically rejected by the live API for every
	// platform checked so far (apk/msi/ipa) -- see the EMPIRICAL
	// REQUIRED-FIELD FINDINGS doc comment on the apk/msi mock lifecycle
	// tests in generated_internalappsv1_lifecycle_test.go.
	modelName string
	// wantPlatform is the literal string InternalAppModelV1.Platform is
	// expected to report on GET (a different, string-typed field from the
	// integer-typed InternalApplicationEntityV1.Platform on create).
	wantPlatform string
	// requiresSoftwareDistributionDeployment gates whether
	// SupportedProcessorArchitecture and a full DeploymentOptions block
	// must be populated -- mandatory for msi (and zip, per
	// tools/internalapps-capture/platform.go's requiresMSIDeployment; zip
	// itself is intentionally NOT a case here, see the doc comment on
	// TestLiveInternalAppsV1ChunkedUploadLifecycle below), NOT required for
	// apk/ipa/pkg.
	requiresSoftwareDistributionDeployment bool
}

// liveInternalAppPlatformCases is the full platform table for
// TestLiveInternalAppsV1ChunkedUploadLifecycle. The zip is deliberately absent:
// live .zip capture is HELD pending org-group Software Distribution
// feature-flag confirmation -- do not attempt a live .zip capture until
// that flag's status is confirmed on -- zip has mock-only
// coverage (TestGeneratedInternalAppsV1Lifecycle_ZIP) until that flag is
// confirmed and this table is extended.
//
// The deviceType/modelName confidence varies by platform:
//   - apk ("Android"/"Android") and msi ("WinRT"/"Desktop") are empirically
//     confirmed live (2026-07-30, documented on the mock
//     lifecycle tests).
//   - ipa ("Apple"/"iPhone") is empirically confirmed live
//     (2026-08-01; "iPhone" matches
//     scripts/capture-internal-apps-chunked.sh's own documented working
//     example for this platform).
//   - pkg ("AppleOsX"/"Mac") is UNCONFIRMED for modelName specifically:
//     vendored-test-apps/macos/SPEC.md section 6 confirms SupportedModels.
//     Model must be non-empty but does not confirm any live-validated
//     per-platform picklist value the way apk/msi/ipa's sessions did (no
//     "Any"-was-rejected-for-AppleOsX finding exists yet). "Mac" is a
//     plausible placeholder, not a verified value -- if the live run this
//     table is meant for rejects it with an invalid-model-name error,
//     that is expected until a picklist lookup against
//     GET /api/mdm/picklists/platforms/appleosx/devicemodels confirms the
//     real value.
var liveInternalAppPlatformCases = []liveInternalAppPlatformCase{
	{
		name:           "APK",
		testAppRelPath: filepath.Join("android", "AndroidApkTestApp-1.0.apk"),
		deviceType:     "Android",
		modelName:      "Android",
		wantPlatform:   "Android",
	},
	{
		name:                                   "MSI",
		testAppRelPath:                         filepath.Join("windows", "WindowsMsiTestApp-1.0.msi"),
		deviceType:                             "WinRT",
		modelName:                              "Desktop",
		wantPlatform:                           "WinRT",
		requiresSoftwareDistributionDeployment: true,
	},
	{
		name:           "IPA",
		testAppRelPath: filepath.Join("ios", "HelloMDMTest.ipa"),
		deviceType:     "Apple",
		modelName:      "iPhone",
		wantPlatform:   "Apple",
	},
	{
		name:           "PKG",
		testAppRelPath: filepath.Join("macos", "MacOSPkgTestApp-1.0.pkg"),
		deviceType:     "AppleOsX",
		// modelName: live-queried via GET
		// /api/mdm/picklists/platforms/AppleOsX/devicemodels --
		// real values are "Any", "MacBook Pro", "Mac Studio", "Virtual
		// Machine", "iMac Pro", "MacBook Air", "Mac Mini", "MacBook", "iMac",
		// "Mac Pro". "Mac" (the prior placeholder) is not a valid entry and
		// was rejected live ("Device Model specified is invalid."). "MacBook"
		// mirrors the plain-word convention already established for
		// apk/msi ("Android"/"Desktop").
		modelName:    "MacBook",
		wantPlatform: "AppleOsX",
	},
}

// TestLiveInternalAppsV1ChunkedUploadLifecycle is the TEST_MODE=live
// counterpart to the apk/msi/ipa/pkg lifecycle tests in
// generated_internalappsv1_lifecycle_test.go and the mock-only end-to-end
// coverage in internal_apps_chunked_upload_test.go. It drives the real
// chunked-upload internal-app create lifecycle end to end against a live
// Workspace ONE UEM tenant, once per platform in liveInternalAppPlatformCases,
// reusing resources.ChunkedUploader (via the sdk.NewChunkedUploader
// constructor) rather than re-implementing any chunking logic. The zip is
// intentionally NOT one of the platforms exercised here -- see
// liveInternalAppPlatformCases's doc comment.
//
// Unlike the single-platform version this replaced, each subtest uploads a
// REAL vendored synthetic binary (assets/test-apps/<platform>/… on the
// chore/vendored-sdk-test-apps orphan branch, staged locally by
// `make vendor-test-apps` into TEST_APPS_DIR, default "vendored-test-apps")
// rather than an arbitrary byte stream with a plausible-looking extension --
// several platforms' create validation depends on genuine file structure
// (e.g. ipa's provisioning-profile/DTPlatformName checks, msi's real
// Property-table read), which a fake byte stream cannot satisfy. See each
// vendored-test-apps/<platform>/SPEC.md for why that platform's binary is
// shaped the way it is.
//
// REAL-WIRE BEHAVIORS this test's assertions are pinned to (apk/msi/ipa;
// pkg is new and unverified, see liveInternalAppPlatformCases):
//
//   - InternalApplicationEntityV1.PushMode: the live create response
//     returns PushMode as a JSON number, which used to break this call's
//     typed decode entirely. That fix is already merged into
//     this branch's base -- this test
//     does not need any special-case handling for it; CreateInternalApplication
//     should simply succeed and decode cleanly. If it does NOT, that is a
//     regression of that fix, not a new problem to work around here.
//   - Post-delete GET returns HTTP 401 with body errorCode 7000
//     ("Application not found or user does not have access to it"), NOT a
//     plain 404. Asserted explicitly below via errors.As against
//     *client.APIError rather than assuming any particular status code.
func TestLiveInternalAppsV1ChunkedUploadLifecycle(t *testing.T) {
	skipUnlessLiveMutate(t)

	for _, tc := range liveInternalAppPlatformCases {
		t.Run(tc.name, func(t *testing.T) {
			runLiveInternalAppsV1ChunkedUploadLifecycle(t, tc)
		})
	}
}

func runLiveInternalAppsV1ChunkedUploadLifecycle(t *testing.T, tc liveInternalAppPlatformCase) {
	t.Helper()

	c := getLiveClientWithTimeout(t, pickLiveAuthMethod(), liveLifecycleTimeout(t))
	ctx := context.Background()

	// ogUUID scopes the created internal application to a disposable child
	// OG rather than whatever OG the live token's own default happens to be:
	// this test previously left
	// InternalAppChunkTransactionV1Model.OrganizationGroupUUID unset
	// entirely, so LIVE_MUTATE=1 alone could create+delete a real app under
	// the shared tenant-wide OG with no containment at all. The chunked
	// upload itself (uploader.Upload, step 1 below) goes to
	// /api/mam/apps/internal/uploadchunk, not BlobsV2 -- it does not create
	// a Blobs-table row, so unlike TestLiveMacOsAppsV1Lifecycle above there
	// is no separate non-cascading blob to delete before OG teardown; the
	// app delete in step 4 below is the only resource this test creates.
	_, ogUUID := orgtest.NewDisposableOrgGroupWithUUIDOrFatal(t, c, liveOrgGroupID(t), nil)

	stamp := time.Now().UnixNano()

	uploader := sdk.NewChunkedUploader(c)
	svc := mamv1.NewInternalAppsV1Service(c)

	// 1. Upload the platform's real vendored synthetic binary through
	// /api/mam/apps/internal/uploadchunk. Streamed directly from disk (not
	// read into memory first) -- ChunkedUploader.Upload only needs an
	// io.Reader plus the exact total size.
	testAppPath := filepath.Join(resolveTestAppsDir(), tc.testAppRelPath)
	f, err := os.Open(testAppPath)
	if err != nil {
		t.Fatalf("open vendored test-app binary %s: %v (set %s to override the default %q, or run `make vendor-test-apps` first)", testAppPath, err, TestAppsDirEnv, defaultTestAppsDir)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("stat %s: %v", testAppPath, err)
	}
	totalSize := info.Size()

	session, err := uploader.Upload(ctx, f, totalSize, nil)
	if err != nil {
		t.Fatalf("ChunkedUploader.Upload failed: %v", err)
	}
	if session.TransactionID == "" {
		t.Fatal("session.TransactionID is empty after Upload")
	}

	// 2. Create the internal application from the uploaded chunks.
	// SupportedModels uses the platform's real picklist value -- the
	// generic "Any" is rejected by the live API on every platform checked
	// so far (empirical finding from the apk/msi live-capture pass, see
	// liveInternalAppPlatformCases's doc comment).
	fileName := fmt.Sprintf("sdk-live-lifecycle-%d%s", stamp, filepath.Ext(tc.testAppRelPath))
	metadata := &mamv1.InternalAppChunkTransactionV1Model{
		ApplicationName:       fmt.Sprintf("SDK Live Lifecycle Test %s %d", tc.name, stamp),
		FileName:              fileName,
		DeviceType:            tc.deviceType,
		OrganizationGroupUUID: ogUUID,
		SupportedModels: &mamv1.ApplicationSupportedV1Model{
			Model: []mamv1.AppSupportedV1Model{
				{ModelName: tc.modelName},
			},
		},
	}
	if tc.requiresSoftwareDistributionDeployment {
		// Mandatory for msi (and zip, held): "Deployment options
		// WhenToInstall, HowToInstall, WhenToCallInstallComplete cannot be
		// null or empty" and "SupportedProcessorArchitecture cannot be null
		// or empty" -- see the EMPIRICAL REQUIRED-FIELD FINDINGS doc
		// comment on the mock msi lifecycle test.
		//
		// Live-verified against UEM 26.2.1614.12:
		// the prior finding's "inner fields may stay default" claim is WRONG
		// for at least two HowToInstall sub-fields, despite both being
		// `omitempty` in the Go model (internal/mam/v1/models.go:789,793) --
		// leaving either at its zero value ("") is REJECTED live, not
		// silently omitted/defaulted server-side:
		//   - InstallContext: "InstallContext is invalid. Supported values:
		//     Device, User" -- "Device" used (system/device-context install,
		//     the standard choice for an unattended MDM-pushed install;
		//     "User" would require an interactively logged-in user context).
		//   - DeviceRestart: "DeviceRestart option is invalid. Supported
		//     values: DoNotRestart, ForceRestart, RestartIfNeeded" --
		//     "DoNotRestart" used (safest choice for an unattended test
		//     lifecycle; forcing a real device restart is never appropriate
		//     here).
		//   - InstallCommand: "InstallCommand cannot be null or empty." --
		//     "/quiet" used, the model's own doc-comment example
		//     (internal/mam/v1/models.go:790) and the standard silent-install
		//     flag for msiexec.
		metadata.SupportedProcessorArchitecture = "x64"
		metadata.DeploymentOptions = &mamv1.ApplicationDeploymentOptionsV1Model{
			HowToInstall: &mamv1.HowToInstallV1Model{
				InstallContext: "Device",
				DeviceRestart:  "DoNotRestart",
				InstallCommand: "/quiet",
			},
			WhenToCallInstallComplete: &mamv1.WhenToCallInstallCompleteV1Model{},
			WhenToInstall:             &mamv1.WhenToInstallV1Model{},
		}
	}

	app, err := uploader.CreateInternalApplication(ctx, session, metadata)
	if err != nil {
		t.Fatalf("CreateInternalApplication failed (if this is a JSON decode error on PushMode, the PushMode wire-type fix has regressed): %v", err)
	}
	if app == nil {
		t.Fatal("CreateInternalApplication returned nil application")
	}
	if app.ID == nil || app.ID.Value == nil {
		t.Fatal("CreateInternalApplication: response missing Id.Value")
	}
	appID := int(*app.ID.Value)
	if appID <= 0 {
		t.Fatalf("CreateInternalApplication: non-positive app id %d in response", appID)
	}

	// Safety-net teardown: re-attempts DeleteInternalAppAsync even if the
	// explicit delete step below never runs because an earlier assertion in
	// this test failed first. A delete of an already-deleted app just
	// errors, which is only logged, not failed -- the primary, intentional
	// teardown is the explicit Delete + post-delete verification later in
	// this test.
	t.Cleanup(func() {
		if _, err := svc.DeleteInternalAppAsync(context.Background(), appID); err != nil {
			t.Logf("cleanup: DeleteInternalAppAsync(%d) (safety net) failed: %v", appID, err)
		}
	})

	// 3. Confirm the app round-trips via GET.
	_, got, err := svc.GetInternalAppByIdAsync(ctx, appID)
	if err != nil {
		t.Fatalf("GetInternalAppByIdAsync(%d) failed: %v", appID, err)
	}
	if got == nil {
		t.Fatal("GetInternalAppByIdAsync returned nil app")
	}
	if got.Platform != tc.wantPlatform {
		t.Errorf("Platform: got %q, want %q", got.Platform, tc.wantPlatform)
	}
	if got.UUID == "" {
		t.Error("expected non-empty uuid in GetInternalAppByIdAsync response")
	}

	// 4. Delete (primary, intentional teardown).
	if _, err := svc.DeleteInternalAppAsync(ctx, appID); err != nil {
		t.Fatalf("DeleteInternalAppAsync(%d) failed: %v", appID, err)
	}

	// 5. Verify the REAL post-delete signature: HTTP 401 with errorCode
	// 7000, not a plain 404. Assert via errors.As against *client.APIError
	// (the service method wraps the underlying error with fmt.Errorf's %w,
	// so a direct type assertion on err would not match).
	_, _, err = svc.GetInternalAppByIdAsync(ctx, appID)
	if err == nil {
		t.Fatal("expected an error from GetInternalAppByIdAsync after delete, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected a *client.APIError somewhere in the post-delete GET error chain, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("post-delete GET status: got %d, want 401 (real-wire not-found signature for this endpoint, not a plain 404)", apiErr.StatusCode)
	}
	if apiErr.ErrorCode != "7000" {
		t.Errorf("post-delete GET apiErr.ErrorCode: got %q, want \"7000\" (expected real-wire signature)", apiErr.ErrorCode)
	}
}
