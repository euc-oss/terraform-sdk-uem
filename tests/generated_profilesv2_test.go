package tests

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// boolPtr and intPtr are small helpers for building request bodies whose
// fields are optional pointers (e.g. GeneralPayloadV2Entity).
func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// Test profile IDs used across the Apple/AppleOsX/WinRT platform lifecycle
// tests below. These are anonymized placeholders baked into the fixture
// files' metadata.endpoint as literal (not "{id}") paths -- the mockserver's
// exact-path match (score 200) beats any {id}-template fixture (score 100),
// so each platform's own literal id is collision-free against every other
// fixture in the corpus. The get_profile_12345.json, get_android_68748.json,
// and get_apple_68749.json were fixed to the same literal-endpoint pattern
// -- they previously all
// declared the generic {id} template and genuinely tied for score against
// each other and against the canonical profiles_get_apple_id.json capture,
// resolved silently by directory-walk load order rather than by test
// intent. They do NOT need to match the real profile ids used during live
// capture (those were anonymized out of the fixtures, same as
// testSmartGroupID in tests/generated_smartgroupsv1_test.go).
const (
	testApplePlatformProfileID    = 68910
	testAppleOsXPlatformProfileID = 68911
	testWinRTPlatformProfileID    = 68912
)

// TestGeneratedGetProfile verifies GetDeviceProfileDetailsAsync produces
// the correct HTTP request and parses the mock response.
// Fixture: testdata/mock-responses/profiles/get_profile_12345.json.
func TestGeneratedGetProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)

	ctx := context.Background()
	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, 12345)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil {
		t.Fatal("expected non-nil profile response")
	}
}

// TestGeneratedSearchProfiles verifies SearchProfiles produces the correct
// HTTP request and parses the mock response.
// Fixture: testdata/mock-responses/profiles/profiles_get_search.json (the
// actual full-tree-auto-scan winner among several same-route v2 fixtures,
// confirmed by response-content inspection; this test only asserts
// non-nil, so it does not depend on which one wins).
func TestGeneratedSearchProfiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)

	ctx := context.Background()
	_, result, err := svc.SearchProfiles(ctx, nil)
	if err != nil {
		t.Fatalf("SearchProfiles failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil search response")
	}
}

// TestGeneratedCreateProfile verifies CreateAndroidDeviceProfileAsync produces
// the correct HTTP request and parses the scalar int response.
// Fixture: testdata/mock-responses/profiles/create_profile_success.json.
func TestGeneratedCreateProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)

	ctx := context.Background()
	request := &sdk.AndroidDeviceProfileV2Entity{}
	_, profileID, err := svc.CreateAndroidDeviceProfileAsync(ctx, request)
	if err != nil {
		t.Fatalf("CreateAndroidDeviceProfileAsync failed: %v", err)
	}
	if profileID == 0 {
		t.Fatal("expected non-zero profile ID")
	}
}

// ---------------------------------------------------------------------
// Apple (iOS) platform create/update/delete lifecycle.
//
// Live-captured against UEM 26.2.0.0, re-verified with a
// clean v2 Accept header, re-verified after the Accept-header fix (the previous "unserveable" 404
// conclusion was caused entirely by mock-capture silently sending
// version=1 for this v2-only route family; the real API returns a genuine
// 422 validation error, then a genuine success, once the header is right).
//
// FIXED: GetDeviceProfileDetailsAsync's
// response type, DeviceProfileV2Entity (internal/mdm/v2/models.go), used to
// only have a General field -- it did NOT expose Restrictions,
// CredentialsList, WifiList, or any other platform-specific payload, even
// though the real wire response includes all of them (json.Unmarshal
// silently dropped the extra fields). internal-source/mdmv2.profiles-details-response.overlay.yaml
// now back-fills the union of top-level fields observed across every
// platform's live-captured fixtures. Fields with one unambiguous shape
// across platforms reuse that platform's existing named schema, same as
// General. Fields whose shape genuinely differs by platform (Restrictions,
// CredentialsList, CustomSettingsList, EmailList, ScepList, SsoExtensionList,
// VpnList, WifiList, Passcode -- codegen has no oneOf/discriminated-union
// support) decode as generic objects so no field is silently dropped for any
// platform. See tests/generated_deviceprofilev2_response_fields_test.go for
// coverage of these fields. The tests below still only assert on
// profile.General.* to keep the CRUD-lifecycle focus narrow, but
// Restrictions/Firewall content (AllowAirDrop, enableFirewallForPublicNetwork,
// etc.) is now reachable through the generated type if a future test needs it.
// ---------------------------------------------------------------------

// TestGeneratedProfilesV2AppleLifecycle exercises the live-captured
// create/get/update/delete Apple (iOS) lifecycle:
//   - CREATE POST /api/mdm/profiles/platforms/apple/create   -> 200, bare int ProfileId
//   - GET    GET  /api/mdm/profiles/{profileId}               -> 200, initial state
//   - UPDATE POST /api/mdm/profiles/platforms/apple/update    -> 200, empty body
//   - DELETE DELETE /api/mdm/profiles/{profileId}             -> 200, empty body
//
// The get-after-update and get-after-delete states are NOT checked in this
// test (profiles_apple_get_after_create.json and profiles_apple_get_after_update.json
// both use the literal endpoint /api/mdm/profiles/68910 in a full-corpus
// load, so only one of them can win the request); they are verified in
// their own isolated-mock-server tests below, mirroring the pattern in
// TestGeneratedSmartGroupsV1GetAfterUpdate / GetAfterDelete.
func TestGeneratedProfilesV2AppleLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	// CREATE. Fixture: profiles/profiles_apple_create.json
	createReq := &sdk.AppleDeviceProfileV2Entity{
		General: &sdk.GeneralPayloadV2Entity{
			Name:                   "sdk-test-apple",
			Description:            "SDK CRUD validation - apple create",
			ManagedLocationGroupID: intPtr(12345),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
		},
		Restrictions: &sdk.AppleRestrictionsPayloadV2Entity{
			AllowAirDrop: boolPtr(false),
		},
	}
	_, profileID, err := svc.CreateAppleDeviceProfileAsync(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateAppleDeviceProfileAsync failed: %v", err)
	}
	// NOTE: internal/mockserver/server.go's handleStatefulRequest intercepts
	// every POST to /platforms/{x}/create BEFORE the static fixture matcher
	// ever runs, always returning a hardcoded id (99999) so the rest of the
	// shared CRUD simulation (used by Layer 2's ProfileService tests) has
	// something to key off of. This means profiles_apple_create.json's own
	// response body (68910) is not reachable through this test -- it exists
	// for fixture-corpus completeness / verified-endpoints provenance, and
	// was confirmed directly against the live API during capture. Only non-zero is asserted here, matching the
	// existing convention in TestGeneratedCreateProfile (Android).
	if profileID == 0 {
		t.Fatal("expected non-zero profile id")
	}

	// GET (after create). Fixture: profiles/profiles_apple_get_after_create.json
	// This uses the fixed test id (not the id returned by Create above)
	// because the stateful simulator only intercepts GETs for ids it
	// created itself (99999) -- a GET for any other id falls through to the
	// static matcher, which is what exercises our real captured fixture.
	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, testApplePlatformProfileID)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil || profile.General == nil {
		t.Fatal("expected non-nil profile with General")
	}
	if profile.General.Version == nil || *profile.General.Version != 1 {
		t.Errorf("expected initial Version=1, got %+v", profile.General.Version)
	}

	// UPDATE, with CreateNewVersion=true (without it, this is a real-wire
	// SILENT NO-OP -- see profiles_apple_update.json). Fixture: profiles/profiles_apple_update.json
	// Targets the id actually returned by Create above (not the fixed test
	// id) so the shared stateful simulator's existence check passes.
	updateReq := &sdk.AppleDeviceProfileV2Entity{
		General: &sdk.GeneralPayloadV2Entity{
			ProfileID:              intPtr(profileID),
			Name:                   "sdk-test-apple",
			Description:            "SDK CRUD validation - apple UPDATED",
			ManagedLocationGroupID: intPtr(12345),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
			CreateNewVersion:       boolPtr(true),
		},
		Restrictions: &sdk.AppleRestrictionsPayloadV2Entity{
			AllowAirDrop: boolPtr(true),
		},
	}
	if _, err := svc.UpdateAppleDeviceProfileAsync(ctx, updateReq); err != nil {
		t.Fatalf("UpdateAppleDeviceProfileAsync failed: %v", err)
	}

	// DELETE. Fixture: profiles/profiles_apple_delete.json
	// DELETE by int id is registered only at v1 (quirk 1).
	if _, err := sdk.NewProfilesV1Service(c).DeleteDeviceProfileAsync(ctx, profileID); err != nil {
		t.Fatalf("DeleteDeviceProfileAsync failed: %v", err)
	}
}

// TestGeneratedProfilesV2AppleGetAfterUpdate verifies the
// GetDeviceProfileDetailsAsync response shape immediately after
// UpdateAppleDeviceProfileAsync (with CreateNewVersion=true) -- confirmed
// live that the update actually persisted (Version incremented 1->2,
// Description changed), not a silent no-op.
func TestGeneratedProfilesV2AppleGetAfterUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_apple_get_after_update.json"))
	if err != nil {
		t.Fatalf("load get_after_update fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, testApplePlatformProfileID)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil || profile.General == nil {
		t.Fatal("expected non-nil profile with General")
	}
	if profile.General.Version == nil || *profile.General.Version != 2 {
		t.Errorf("expected post-update Version=2, got %+v", profile.General.Version)
	}
	if profile.General.Description == "" {
		t.Error("expected non-empty post-update description (anonymized fixture content)")
	}
}

// TestGeneratedProfilesV2AppleGetAfterDelete verifies the
// GetDeviceProfileDetailsAsync error path immediately after
// DeleteDeviceProfileAsync (v1). Real-wire finding: the live API returns HTTP 400
// with errorCode 400 and message "Invalid Profile <id>." -- not a literal
// HTTP 404 -- confirming the delete was real (the profile is genuinely
// gone), not silently ignored.
func TestGeneratedProfilesV2AppleGetAfterDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_apple_get_after_delete.json"))
	if err != nil {
		t.Fatalf("load get_after_delete fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, _, err = svc.GetDeviceProfileDetailsAsync(ctx, testApplePlatformProfileID)
	if err == nil {
		t.Fatal("expected error for deleted profile, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to mention status 400, got: %v", err)
	}
}

// ---------------------------------------------------------------------
// AppleOsX (macOS) platform create/update/delete lifecycle.
// Same provenance and GetDeviceProfileDetailsAsync caveat as the Apple
// (iOS) lifecycle above.
// ---------------------------------------------------------------------

// TestGeneratedProfilesV2AppleOsXLifecycle exercises the live-captured
// create/get/update/delete AppleOsX lifecycle. See
// TestGeneratedProfilesV2AppleLifecycle for the shadowing note on
// get-after-update/get-after-delete.
//
// Real-wire finding (matches the request-template notes):
// Restrictions.Sharing.Mail was requested as true on both create and
// update, but persisted as false both times -- Restrictions.Sharing's
// individual toggles only take effect once the parent
// RestrictWhichSharingServicesAreEnabled flag is also set true.
func TestGeneratedProfilesV2AppleOsXLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	// CREATE. Fixture: profiles/profiles_appleosx_create.json
	createReq := &sdk.AppleOsXDeviceProfileEntityV2{
		General: &sdk.GeneralPayloadV2Entity{
			Name:                   "sdk-test-appleosx",
			Description:            "SDK CRUD validation - appleosx create",
			ManagedLocationGroupID: intPtr(12345),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
		},
		Restrictions: &sdk.AppleOsXRestrictionsPayloadEntityV2{
			Functionality: &sdk.AppleOsXRestrictionFunctionalityPayloadEntityV2{
				Spotlight: &sdk.AppleOsXRestrictionSpotlightPayloadEntityV2{
					AllowSpotlightSuggestions: boolPtr(false),
				},
			},
			Sharing: &sdk.AppleOsXRestrictionSharingPayloadEntityV2{
				AirDrop:  boolPtr(false),
				Facebook: boolPtr(false),
				Mail:     boolPtr(true),
			},
		},
	}
	_, profileID, err := svc.CreateAppleOsXDeviceProfileAsync(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateAppleOsXDeviceProfileAsync failed: %v", err)
	}
	// See the comment on the equivalent assertion in
	// TestGeneratedProfilesV2AppleLifecycle for why this only checks
	// non-zero rather than the fixture's own captured id.
	if profileID == 0 {
		t.Fatal("expected non-zero profile id")
	}

	// GET (after create). Fixture: profiles/profiles_appleosx_get_after_create.json
	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, testAppleOsXPlatformProfileID)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil || profile.General == nil {
		t.Fatal("expected non-nil profile with General")
	}
	if profile.General.Version == nil || *profile.General.Version != 1 {
		t.Errorf("expected initial Version=1, got %+v", profile.General.Version)
	}

	// UPDATE, with CreateNewVersion=true. Fixture: profiles/profiles_appleosx_update.json
	updateReq := &sdk.AppleOsXDeviceProfileEntityV2{
		General: &sdk.GeneralPayloadV2Entity{
			ProfileID:              intPtr(profileID),
			Name:                   "sdk-test-appleosx",
			Description:            "SDK CRUD validation - appleosx UPDATED",
			ManagedLocationGroupID: intPtr(12345),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
			CreateNewVersion:       boolPtr(true),
		},
		Restrictions: &sdk.AppleOsXRestrictionsPayloadEntityV2{
			Functionality: &sdk.AppleOsXRestrictionFunctionalityPayloadEntityV2{
				Spotlight: &sdk.AppleOsXRestrictionSpotlightPayloadEntityV2{
					AllowSpotlightSuggestions: boolPtr(true),
				},
			},
			Sharing: &sdk.AppleOsXRestrictionSharingPayloadEntityV2{
				AirDrop:  boolPtr(false),
				Facebook: boolPtr(false),
				Mail:     boolPtr(true),
			},
		},
	}
	if _, err := svc.UpdateAppleOsXDeviceProfileAsync(ctx, updateReq); err != nil {
		t.Fatalf("UpdateAppleOsXDeviceProfileAsync failed: %v", err)
	}

	// DELETE. Fixture: profiles/profiles_appleosx_delete.json
	// DELETE by int id is registered only at v1 (quirk 1).
	if _, err := sdk.NewProfilesV1Service(c).DeleteDeviceProfileAsync(ctx, profileID); err != nil {
		t.Fatalf("DeleteDeviceProfileAsync failed: %v", err)
	}
}

// TestGeneratedProfilesV2AppleOsXGetAfterUpdate verifies the
// GetDeviceProfileDetailsAsync response shape immediately after
// UpdateAppleOsXDeviceProfileAsync (with CreateNewVersion=true).
func TestGeneratedProfilesV2AppleOsXGetAfterUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_appleosx_get_after_update.json"))
	if err != nil {
		t.Fatalf("load get_after_update fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, testAppleOsXPlatformProfileID)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil || profile.General == nil {
		t.Fatal("expected non-nil profile with General")
	}
	if profile.General.Version == nil || *profile.General.Version != 2 {
		t.Errorf("expected post-update Version=2, got %+v", profile.General.Version)
	}
	if profile.General.Description == "" {
		t.Error("expected non-empty post-update description (anonymized fixture content)")
	}
}

// TestGeneratedProfilesV2AppleOsXGetAfterDelete verifies the
// GetDeviceProfileDetailsAsync error path immediately after
// DeleteDeviceProfileAsync (v1) for the AppleOsX platform.
func TestGeneratedProfilesV2AppleOsXGetAfterDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_appleosx_get_after_delete.json"))
	if err != nil {
		t.Fatalf("load get_after_delete fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, _, err = svc.GetDeviceProfileDetailsAsync(ctx, testAppleOsXPlatformProfileID)
	if err == nil {
		t.Fatal("expected error for deleted profile, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to mention status 400, got: %v", err)
	}
}

// ---------------------------------------------------------------------
// WinRT (Windows 10) platform create/update/delete lifecycle.
// Same provenance and GetDeviceProfileDetailsAsync caveat as above.
// Note the update verb: WinRT update is PUT (internal codegen config verb
// override), unlike apple/appleosx/android which are all POST -- confirmed
// correct on the wire during this capture.
// ---------------------------------------------------------------------

// TestGeneratedProfilesV2WinRTLifecycle exercises the live-captured
// create/get/update/delete WinRT lifecycle. See
// TestGeneratedProfilesV2AppleLifecycle for the shadowing note on
// get-after-update/get-after-delete.
//
// Real-wire finding: CreateWinRTDeviceProfileAsync returns HTTP 201, unlike
// CreateAppleDeviceProfileAsync/CreateAppleOsXDeviceProfileAsync which both
// return HTTP 200 for the same bare-int response shape -- status code is
// not uniform across platforms for this operation family.
func TestGeneratedProfilesV2WinRTLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	// CREATE. Fixture: profiles/profiles_winrt_create.json
	createReq := &sdk.WinRTDeviceProfileV2Entity{
		General: &sdk.GeneralPayloadV2Entity{
			Name:                   "sdk-test-winrt",
			Description:            "SDK CRUD validation - winrt create",
			ManagedLocationGroupID: intPtr(12345),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
		},
		Firewall: &sdk.WindowsDesktopFirewallPayloadEntityV2{
			EnableFirewallForPrivateNetwork: boolPtr(true),
		},
	}
	_, profileID, err := svc.CreateWinRTDeviceProfileAsync(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateWinRTDeviceProfileAsync failed: %v", err)
	}
	// See the comment on the equivalent assertion in
	// TestGeneratedProfilesV2AppleLifecycle for why this only checks
	// non-zero rather than the fixture's own captured id.
	if profileID == 0 {
		t.Fatal("expected non-zero profile id")
	}

	// GET (after create). Fixture: profiles/profiles_winrt_get_after_create.json
	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, testWinRTPlatformProfileID)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil || profile.General == nil {
		t.Fatal("expected non-nil profile with General")
	}
	if profile.General.Version == nil || *profile.General.Version != 1 {
		t.Errorf("expected initial Version=1, got %+v", profile.General.Version)
	}

	// UPDATE (PUT, not POST), with CreateNewVersion=true. Fixture: profiles/profiles_winrt_update.json
	updateReq := &sdk.WinRTDeviceProfileV2Entity{
		General: &sdk.GeneralPayloadV2Entity{
			ProfileID:              intPtr(profileID),
			Name:                   "sdk-test-winrt",
			Description:            "SDK CRUD validation - winrt UPDATED",
			ManagedLocationGroupID: intPtr(12345),
			AssignmentType:         "Auto",
			ProfileContext:         "Device",
			CreateNewVersion:       boolPtr(true),
		},
		Firewall: &sdk.WindowsDesktopFirewallPayloadEntityV2{
			EnableFirewallForPrivateNetwork: boolPtr(true),
			EnableFirewallForPublicNetwork:  boolPtr(true),
		},
	}
	if _, err := svc.UpdateWinRTDeviceProfileAsync(ctx, updateReq); err != nil {
		t.Fatalf("UpdateWinRTDeviceProfileAsync failed: %v", err)
	}

	// DELETE. Fixture: profiles/profiles_winrt_delete.json
	// DELETE by int id is registered only at v1 (quirk 1).
	if _, err := sdk.NewProfilesV1Service(c).DeleteDeviceProfileAsync(ctx, profileID); err != nil {
		t.Fatalf("DeleteDeviceProfileAsync failed: %v", err)
	}
}

// TestGeneratedProfilesV2WinRTGetAfterUpdate verifies the
// GetDeviceProfileDetailsAsync response shape immediately after
// UpdateWinRTDeviceProfileAsync (with CreateNewVersion=true).
func TestGeneratedProfilesV2WinRTGetAfterUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_winrt_get_after_update.json"))
	if err != nil {
		t.Fatalf("load get_after_update fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, profile, err := svc.GetDeviceProfileDetailsAsync(ctx, testWinRTPlatformProfileID)
	if err != nil {
		t.Fatalf("GetDeviceProfileDetailsAsync failed: %v", err)
	}
	if profile == nil || profile.General == nil {
		t.Fatal("expected non-nil profile with General")
	}
	if profile.General.Version == nil || *profile.General.Version != 2 {
		t.Errorf("expected post-update Version=2, got %+v", profile.General.Version)
	}
	if profile.General.Description == "" {
		t.Error("expected non-empty post-update description (anonymized fixture content)")
	}
}

// TestGeneratedProfilesV2WinRTGetAfterDelete verifies the
// GetDeviceProfileDetailsAsync error path immediately after
// DeleteDeviceProfileAsync (v1) for the WinRT platform.
func TestGeneratedProfilesV2WinRTGetAfterDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_winrt_get_after_delete.json"))
	if err != nil {
		t.Fatalf("load get_after_delete fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, _, err = svc.GetDeviceProfileDetailsAsync(ctx, testWinRTPlatformProfileID)
	if err == nil {
		t.Fatal("expected error for deleted profile, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to mention status 400, got: %v", err)
	}
}

// TestGeneratedProfilesV2SearchV2 verifies SearchProfiles against the
// re-captured v2 fixture (profiles_search_v2.json), which correctly uses
// the current wire shape (AdditionalInfo + ProfileList + TotalResults),
// unlike the older search_android_success.json (a v1-shaped Profiles/Page/
// PageSize body that does not match ProfileSearchResultV2Entity at all).
// Loaded via an isolated single-fixture mock server so this specific
// fixture's shape is what gets exercised, regardless of tie-break ordering
// among the several pre-existing (and, in places, hand-fabricated) search
// fixtures already in the corpus.
func TestGeneratedProfilesV2SearchV2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_search_v2.json"))
	if err != nil {
		t.Fatalf("load search_v2 fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV2Service(c)
	ctx := context.Background()

	_, result, err := svc.SearchProfiles(ctx, nil)
	if err != nil {
		t.Fatalf("SearchProfiles failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil search response")
	}
	// Field-verification: the generated ProfileSearchResultV2Entity already
	// has a ProfileList field (not "Profiles") -- matches the live wire
	// response field-for-field, confirmed by this fixture.
	if len(result.ProfileList) != 3 {
		t.Fatalf("expected 3 profiles in ProfileList, got %d", len(result.ProfileList))
	}
	if result.TotalResults == nil || *result.TotalResults != 3 {
		t.Errorf("expected TotalResults=3, got %+v", result.TotalResults)
	}
	platforms := map[string]bool{}
	for _, p := range result.ProfileList {
		platforms[p.Platform] = true
	}
	if !platforms["Apple"] || !platforms["WinRT"] {
		t.Errorf("expected Apple and WinRT platforms in results, got %+v", platforms)
	}
}
