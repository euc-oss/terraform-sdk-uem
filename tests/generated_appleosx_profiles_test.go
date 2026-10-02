package tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
	"github.com/euc-oss/terraform-sdk-uem/internal/orgtest"
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// These tests exercise the generated macOS profile surface end-to-end through
// the mock server, using fixtures captured from a live UEM
// (UEM 25.9.1134.9) during a macOS CRUD sweep.
//
// Fixtures live under:
//
//   testdata/mock-responses/profiles/appleosx/{payload-type}/get-by-id-v2.json
//
// Each get fixture has a unique anonymized id baked into metadata.endpoint and
// body.General.ProfileId so the matcher routes each test to the right fixture.
//
// Layer 2 ProfileService is used for GETs because Layer 2 routes the
// response into the platform-specific AppleOsXDeviceProfileEntityV2, giving
// fully-typed sub-payloads. Layer 1 GetDeviceProfileDetailsAsync's
// DeviceProfileV2Entity (fixed via
// internal-source/mdmv2.profiles-details-response.overlay.yaml) now also
// exposes the sub-payloads beyond General, but fields whose shape differs by
// platform (Restrictions, CredentialsList, etc.) decode as generic objects
// there rather than the concrete platform types Layer 2 provides.

// setupAppleOsXGet returns a Layer 2 ProfileService pre-registered for the
// given macOS profile ids, wired onto the shared mock server. Using
// WithoutDiscovery + RegisterEntry avoids depending on a search fixture.
func setupAppleOsXGet(t *testing.T, ids ...int) *sdk.ProfileService {
	t.Helper()
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)
	svc := sdk.NewProfileServiceWithoutDiscovery(mockserver.NewMockClient(t, ms))
	for _, id := range ids {
		svc.RegisterEntry(id, "AppleOsX")
	}
	return svc
}

// TestAppleOsXCreateProfile verifies the macOS create endpoint returns a bare
// int profile id that the Layer 1 generated Create method deserializes.
// Fixture: testdata/mock-responses/profiles/appleosx/create-v2.json.
func TestAppleOsXCreateProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)
	l1 := sdk.NewProfilesV2Service(mockserver.NewMockClient(t, ms))

	_, id, err := l1.CreateAppleOsXDeviceProfileAsync(context.Background(), &sdk.AppleOsXDeviceProfileEntityV2{})
	if err != nil {
		t.Fatalf("CreateAppleOsXDeviceProfileAsync: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero profile id")
	}
}

// TestAppleOsXGetByIdDiskEncryption verifies the disk-encryption sub-payload
// round-trips, including the recently-fixed int→string enum fields from PR #21.
// Fixture: testdata/mock-responses/profiles/appleosx/disk-encryption/get-by-id-v2.json.
func TestAppleOsXGetByIdDiskEncryption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	result, err := setupAppleOsXGet(t, 12346).Get(context.Background(), 12346)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.AppleOsX == nil {
		t.Fatal("expected AppleOsX profile populated")
	}
	de := result.AppleOsX.DiskEncryption
	if de == nil {
		t.Fatal("expected DiskEncryption populated")
	}
	// Regression guard: these fields must deserialize as strings (not *int).
	if de.DiskEncryptionFileVault2 == nil {
		t.Fatal("expected FileVault2 populated")
	}
	if de.DiskEncryptionFileVault2.FileVaultUser == "" {
		t.Error("expected FileVaultUser string, got empty")
	}
	if de.DiskEncryptionFileVault2.PromptToEnableFileVaultAt == "" {
		t.Error("expected PromptToEnableFileVaultAt string, got empty")
	}
	if de.DiskEncryptionAirWatch == nil {
		t.Fatal("expected AirWatch populated")
	}
	if de.DiskEncryptionAirWatch.EncryptionActionAfterLastNotification == "" {
		t.Error("expected EncryptionActionAfterLastNotification string, got empty")
	}
}

// TestAppleOsXGetByIdCustomSettings verifies the custom-settings list sub-payload
// round-trips with the plist XML preserved verbatim.
// Fixture: testdata/mock-responses/profiles/appleosx/custom-settings/get-by-id-v2.json.
func TestAppleOsXGetByIdCustomSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	result, err := setupAppleOsXGet(t, 12347).Get(context.Background(), 12347)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.AppleOsX == nil {
		t.Fatal("expected AppleOsX profile populated")
	}
	if len(result.AppleOsX.CustomSettingsList) == 0 {
		t.Fatal("expected CustomSettingsList populated")
	}
	if result.AppleOsX.CustomSettingsList[0].CustomSettings == "" {
		t.Error("expected CustomSettings plist string, got empty")
	}
}

// TestAppleOsXGetByIdNetworkWifi verifies the network-wifi list sub-payload round-trips.
// Fixture: testdata/mock-responses/profiles/appleosx/network-wifi/get-by-id-v2.json.
func TestAppleOsXGetByIdNetworkWifi(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	result, err := setupAppleOsXGet(t, 12348).Get(context.Background(), 12348)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.AppleOsX == nil {
		t.Fatal("expected AppleOsX profile populated")
	}
	if len(result.AppleOsX.NetworkList) == 0 {
		t.Fatal("expected NetworkList populated")
	}
	net := result.AppleOsX.NetworkList[0]
	if net.NetworkInterface == "" {
		t.Error("expected NetworkInterface, got empty")
	}
	if net.SecurityType == "" {
		t.Error("expected SecurityType, got empty")
	}
}

// TestAppleOsXGetByIdRestrictions verifies the restrictions sub-payload with
// nested Sharing + Functionality entities round-trips.
// Fixture: testdata/mock-responses/profiles/appleosx/restrictions/get-by-id-v2.json.
func TestAppleOsXGetByIdRestrictions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	result, err := setupAppleOsXGet(t, 12349).Get(context.Background(), 12349)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.AppleOsX == nil {
		t.Fatal("expected AppleOsX profile populated")
	}
	r := result.AppleOsX.Restrictions
	if r == nil {
		t.Fatal("expected Restrictions populated")
	}
	if r.Sharing == nil {
		t.Fatal("expected Sharing sub-payload populated")
	}
	if r.Functionality == nil || r.Functionality.Spotlight == nil {
		t.Fatal("expected Functionality.Spotlight populated")
	}
}

// liveAppleOsXProfileCreate is the shared body for the four
// TestLive_AppleOsX*Create_FullLiveWithOGContainment tests below: reads the
// given request-template JSON (testdata/request-templates/profiles/appleosx/
// .../*-create-v2.json -- see testdata/request-templates/README.md), unmarshals
// it into the real generated AppleOsXDeviceProfileEntityV2 request type,
// overrides General.ManagedLocationGroupID to the disposable OG created for
// this test run, and posts it through the real generated
// CreateAppleOsXDeviceProfileAsync method (hand-written, not template-
// generated; see .claude/rules/api-doctrine.md and testdata/live-policy.yaml's
// ProfilesV2 entry).
func liveAppleOsXProfileCreate(t *testing.T, templatePath string) {
	t.Helper()
	skipUnlessLiveMutate(t)
	c := getTestClient(t, "basic")
	ctx := context.Background()
	ogID := orgtest.NewDisposableOrgGroupOrFatal(t, c, liveOrgGroupID(t), nil)

	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read request template: %v", err)
	}
	var req sdk.AppleOsXDeviceProfileEntityV2
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal request template: %v", err)
	}
	if req.General == nil {
		t.Fatalf("request template %s has no General payload", templatePath)
	}
	req.General.ManagedLocationGroupID = intPtr(ogID) // override to the disposable OG

	svc := sdk.NewProfilesV2Service(c)
	_, profileID, err := svc.CreateAppleOsXDeviceProfileAsync(ctx, &req)
	if err != nil {
		t.Fatalf("CreateAppleOsXDeviceProfileAsync: %v", err)
	}
	// Registered AFTER the OG create above, so LIFO fires this delete
	// BEFORE the OG's own delete cleanup -- do not depend on OG-delete
	// cascading to child profiles (unverified; only the documented
	// MacOsApps->Blobs cascade is confirmed).
	profileSvc := resources.NewProfileService(c)
	t.Cleanup(func() {
		if err := profileSvc.Delete(context.Background(), profileID); err != nil {
			t.Logf("cleanup: ProfileService.Delete(%d) failed: %v", profileID, err)
		}
	})
}

// TestLive_AppleOsXCustomSettingsCreate_FullLiveWithOGContainment exercises
// the appleosx custom-settings create request-template.
func TestLive_AppleOsXCustomSettingsCreate_FullLiveWithOGContainment(t *testing.T) {
	liveAppleOsXProfileCreate(t, "../testdata/request-templates/profiles/appleosx/custom-settings/custom-settings-create-v2.json")
}

// TestLive_AppleOsXDiskEncryptionCreate_FullLiveWithOGContainment exercises
// the appleosx disk-encryption create request-template.
func TestLive_AppleOsXDiskEncryptionCreate_FullLiveWithOGContainment(t *testing.T) {
	liveAppleOsXProfileCreate(t, "../testdata/request-templates/profiles/appleosx/disk-encryption/disk-encryption-create-v2.json")
}

// TestLive_AppleOsXNetworkWifiCreate_FullLiveWithOGContainment exercises the
// appleosx network-wifi create request-template.
func TestLive_AppleOsXNetworkWifiCreate_FullLiveWithOGContainment(t *testing.T) {
	liveAppleOsXProfileCreate(t, "../testdata/request-templates/profiles/appleosx/network-wifi/network-wifi-create-v2.json")
}

// TestLive_AppleOsXRestrictionsCreate_FullLiveWithOGContainment exercises the
// appleosx restrictions create request-template.
func TestLive_AppleOsXRestrictionsCreate_FullLiveWithOGContainment(t *testing.T) {
	liveAppleOsXProfileCreate(t, "../testdata/request-templates/profiles/appleosx/restrictions/restrictions-create-v2.json")
}
