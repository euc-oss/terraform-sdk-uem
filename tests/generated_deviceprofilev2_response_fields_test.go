package tests

import (
	"encoding/json"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// ProfilesV2Service.GetDeviceProfileDetailsAsync
// (GET /api/mdm/profiles/{profileId}) decodes into sdk.DeviceProfileV2Entity,
// which upstream swagger (and the canonical C# DeviceProfileV2Entity base
// class) declares with only a General field. At runtime the controller
// actually returns one of the platform-specific subclasses
// (AppleDeviceProfileV2Entity, AppleOsXDeviceProfileEntity,
// WinRTDeviceProfileV2Entity, ...), so every payload field beyond General
// was silently dropped by json.Unmarshal regardless of what the profile
// actually contained -- no error, just missing data.
//
// These tests decode real live-captured fixture bodies directly into
// sdk.DeviceProfileV2Entity (bypassing the mock HTTP round-trip, since the
// bug is purely about the destination struct's shape) and assert specific
// non-General field values survive the decode. Before the
// mdmv2.profiles-details-response.overlay.yaml fix, this file does not
// compile: none of the asserted fields (Restrictions, NetworkList, Firewall,
// CredentialsList, etc.) exist on the pre-fix DeviceProfileV2Entity struct.
// That compile failure IS the proof of the bug -- a General-only struct
// can't even be asked about a field that was never declared.
func decodeDeviceProfileFixtureBody(t *testing.T, fixturePath string) *sdk.DeviceProfileV2Entity {
	t.Helper()

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir(fixturePath))
	if err != nil {
		t.Fatalf("load fixture %s: %v", fixturePath, err)
	}

	raw, err := json.Marshal(resp.Response.Body)
	if err != nil {
		t.Fatalf("re-marshal fixture body %s: %v", fixturePath, err)
	}

	var profile sdk.DeviceProfileV2Entity
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("unmarshal fixture body %s into DeviceProfileV2Entity: %v", fixturePath, err)
	}
	return &profile
}

// TestGeneratedDeviceProfileV2Response_AppleRestrictions verifies the Apple
// (iOS) GetDeviceProfileDetailsAsync response decodes its Restrictions
// payload, not just General.
// Fixture: testdata/mock-responses/profiles/profiles_apple_get_after_update.json.
func TestGeneratedDeviceProfileV2Response_AppleRestrictions(t *testing.T) {
	profile := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/profiles_apple_get_after_update.json")

	if profile.General == nil {
		t.Fatal("expected non-nil General")
	}
	if profile.Restrictions == nil {
		t.Fatal("expected non-nil Restrictions -- it was silently dropped")
	}
	allowAirDrop, ok := profile.Restrictions["AllowAirDrop"]
	if !ok {
		t.Fatal("expected Restrictions.AllowAirDrop to be present")
	}
	if allowAirDrop != true {
		t.Errorf("expected Restrictions.AllowAirDrop=true (fixture reflects the post-update state), got %v", allowAirDrop)
	}
}

// TestGeneratedDeviceProfileV2Response_AppleOsXRestrictionsAndNetwork verifies
// the AppleOsX (macOS) response decodes both Restrictions (nested object
// content) and the platform-specific NetworkList field.
// Fixtures: profiles_appleosx_get_after_update.json, get_appleosx_network_profile.json.
func TestGeneratedDeviceProfileV2Response_AppleOsXRestrictionsAndNetwork(t *testing.T) {
	profile := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/profiles_appleosx_get_after_update.json")

	if profile.Restrictions == nil {
		t.Fatal("expected non-nil Restrictions -- it was silently dropped")
	}
	applications, ok := profile.Restrictions["Applications"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Restrictions.Applications to be a nested object, got %T", profile.Restrictions["Applications"])
	}
	appStore, ok := applications["AppStore"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Restrictions.Applications.AppStore to be a nested object, got %T", applications["AppStore"])
	}
	if allow, _ := appStore["AllowAppStoreAppAdoption"].(bool); !allow {
		t.Errorf("expected Restrictions.Applications.AppStore.AllowAppStoreAppAdoption=true, got %v", appStore["AllowAppStoreAppAdoption"])
	}

	// NetworkList carries real configured network payloads for macOS profiles
	// and was entirely absent from the pre-fix DeviceProfileV2Entity.
	networkProfile := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/get_appleosx_network_profile.json")
	if len(networkProfile.NetworkList) != 1 {
		t.Fatalf("expected exactly one NetworkList entry, got %d -- it was silently dropped before the fix", len(networkProfile.NetworkList))
	}
	// The SSID is read from the fixture's own body (the genuine 26.4
	// recapture carries a different anonymized SSID than 26.2's capture).
	var wantNetwork struct {
		NetworkList []struct {
			ServiceSetIdentifier string `json:"ServiceSetIdentifier"`
		} `json:"NetworkList"`
	}
	loadFixtureResponseBodyForTest(t, "profiles/get_appleosx_network_profile.json", &wantNetwork)
	if len(wantNetwork.NetworkList) != 1 || wantNetwork.NetworkList[0].ServiceSetIdentifier == "" {
		t.Fatalf("profiles/get_appleosx_network_profile.json: want one NetworkList entry with an SSID, got %+v", wantNetwork.NetworkList)
	}
	if got, want := networkProfile.NetworkList[0].ServiceSetIdentifier, wantNetwork.NetworkList[0].ServiceSetIdentifier; got != want {
		t.Errorf("NetworkList[0].ServiceSetIdentifier: got %q, want the fixture's %q", got, want)
	}
}

// TestGeneratedDeviceProfileV2Response_WinRTFirewall verifies the WinRT
// (Windows 10) response decodes its Firewall payload, which has no bearing
// on General at all and was completely invisible before the fix.
// Fixture: testdata/mock-responses/profiles/profiles_winrt_get_after_update.json.
func TestGeneratedDeviceProfileV2Response_WinRTFirewall(t *testing.T) {
	profile := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/profiles_winrt_get_after_update.json")

	if profile.Firewall == nil {
		t.Fatal("expected non-nil Firewall -- it was silently dropped")
	}
	if profile.Firewall.EnableFirewallForPublicNetwork == nil || !*profile.Firewall.EnableFirewallForPublicNetwork {
		t.Errorf("expected Firewall.EnableFirewallForPublicNetwork=true, got %v", profile.Firewall.EnableFirewallForPublicNetwork)
	}
	if profile.Firewall.EnableFirewallForPrivateNetwork == nil || !*profile.Firewall.EnableFirewallForPrivateNetwork {
		t.Errorf("expected Firewall.EnableFirewallForPrivateNetwork=true, got %v", profile.Firewall.EnableFirewallForPrivateNetwork)
	}
}

// TestGeneratedDeviceProfileV2Response_AppleOsXDiskEncryption is a
// regression guard for internal-source/mdmv2.profiles-details-response.overlay.yaml's
// DiskEncryption gap-fill: DeviceProfileV2Entity (the base class swagger
// declares for GetDeviceProfileDetailsAsync) was missing DiskEncryption
// entirely, even though AppleOsXDeviceProfileEntity (the real runtime
// subclass for macOS profiles) declares it, and the referenced
// AppleOsXDiskEncryptionPayloadEntity + its three nested sub-schemas
// (DiskEncryptionAirWatch, DiskEncryptionFileVault2, DiskEncryptionMCX)
// were already complete in mdmv2.json. If the overlay entry regresses,
// DiskEncryption disappears from the generated struct and this file fails
// to compile -- same "compile failure is the proof" pattern as the rest
// of this file.
// Fixture: testdata/mock-responses/profiles/appleosx/disk-encryption/get-by-id-v2.json
// (live-captured against UEM 25.9.1134.9).
func TestGeneratedDeviceProfileV2Response_AppleOsXDiskEncryption(t *testing.T) {
	profile := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/appleosx/disk-encryption/get-by-id-v2.json")

	if profile.DiskEncryption == nil {
		t.Fatal("expected non-nil DiskEncryption -- it was silently dropped before the fix")
	}
	if profile.DiskEncryption.DiskEncryptionAirWatch == nil {
		t.Fatal("expected non-nil DiskEncryption.DiskEncryptionAirWatch")
	}
	if profile.DiskEncryption.DiskEncryptionAirWatch.EnableRecoveryKey == nil || !*profile.DiskEncryption.DiskEncryptionAirWatch.EnableRecoveryKey {
		t.Errorf("expected DiskEncryptionAirWatch.EnableRecoveryKey=true, got %v", profile.DiskEncryption.DiskEncryptionAirWatch.EnableRecoveryKey)
	}
	if profile.DiskEncryption.DiskEncryptionFileVault2 == nil {
		t.Fatal("expected non-nil DiskEncryption.DiskEncryptionFileVault2")
	}
	// FileVaultUser is read from the fixture's own body (the genuine 26.4
	// recapture carries a different anonymized value than 26.2's capture).
	var wantDiskEncryption struct {
		DiskEncryption struct {
			DiskEncryptionFileVault2 struct {
				FileVaultUser string `json:"FileVaultUser"`
			} `json:"DiskEncryptionFileVault2"`
		} `json:"DiskEncryption"`
	}
	loadFixtureResponseBodyForTest(t, "profiles/appleosx/disk-encryption/get-by-id-v2.json", &wantDiskEncryption)
	wantFileVaultUser := wantDiskEncryption.DiskEncryption.DiskEncryptionFileVault2.FileVaultUser
	if profile.DiskEncryption.DiskEncryptionFileVault2.FileVaultUser != wantFileVaultUser {
		t.Errorf("DiskEncryptionFileVault2.FileVaultUser: got %q, want %q", profile.DiskEncryption.DiskEncryptionFileVault2.FileVaultUser, wantFileVaultUser)
	}
	if profile.DiskEncryption.DiskEncryptionMCX == nil {
		t.Fatal("expected non-nil DiskEncryption.DiskEncryptionMCX")
	}
	if profile.DiskEncryption.DiskEncryptionMCX.DestroyFVKeyOnStandby == nil || *profile.DiskEncryption.DiskEncryptionMCX.DestroyFVKeyOnStandby {
		t.Errorf("expected DiskEncryptionMCX.DestroyFVKeyOnStandby=false, got %v", profile.DiskEncryption.DiskEncryptionMCX.DestroyFVKeyOnStandby)
	}
}

// TestGeneratedDeviceProfileV2Response_EmptyListsSurvive verifies that
// fields present-but-empty on the wire (e.g. CredentialsList: []) decode as
// empty, non-nil-vs-missing-distinguishable fields rather than not existing
// on the struct at all. This is a weaker assertion than the populated-field
// tests above, but it covers the remaining union fields the richer fixtures
// above don't happen to populate (CredentialsList, WifiList, VpnList,
// EmailList, ScepList, SsoExtensionList, CustomSettingsList, CustomAttributes,
// WebClipsList, GoogleAccount, AWMailCredentialList, EASNativeMailClientList,
// Passcode, AndroidForWorkCustomMessages) so a regression that removes any of
// them from the schema fails a compile, not a silent no-op.
func TestGeneratedDeviceProfileV2Response_EmptyListsSurvive(t *testing.T) {
	apple := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/profiles_apple_get_after_update.json")
	if apple.CredentialsList == nil {
		t.Error("expected CredentialsList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.WifiList == nil {
		t.Error("expected WifiList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.VpnList == nil {
		t.Error("expected VpnList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.EmailList == nil {
		t.Error("expected EmailList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.ScepList == nil {
		t.Error("expected ScepList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.SsoExtensionList == nil {
		t.Error("expected SsoExtensionList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.GoogleAccount == nil {
		t.Error("expected GoogleAccount to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.AWMailCredentialList == nil {
		t.Error("expected AWMailCredentialList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if apple.EASNativeMailClientList == nil {
		t.Error("expected EASNativeMailClientList to decode as an empty (non-nil-omitted) slice from `[]`")
	}

	appleOsX := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/profiles_appleosx_get_after_update.json")
	if appleOsX.CustomSettingsList == nil {
		t.Error("expected CustomSettingsList to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if appleOsX.CustomAttributes == nil {
		t.Error("expected CustomAttributes to decode as an empty (non-nil-omitted) slice from `[]`")
	}
	if appleOsX.WebClipsList == nil {
		t.Error("expected WebClipsList to decode as an empty (non-nil-omitted) slice from `[]`")
	}

	android := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/get_android_testdouble.json")
	if android.Passcode == nil {
		t.Error("expected Passcode to decode as a non-nil generic object")
	}

	dduiProfile := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/get_profile_12345.json")
	if dduiProfile.AndroidForWorkCustomMessages == nil {
		t.Error("expected AndroidForWorkCustomMessages to decode as non-nil -- it was silently dropped before the fix")
	}
}

// TestGeneratedDeviceProfileV2Response_GeneralOrganizationGroupId covers the
// org-group field of the v2 profile GET's General payload.
//
// Apple half: the genuine 26.4 iOS capture. Canonical 26.4 says the v2
// GeneralPayloadV2Entity has NO OrganizationGroupId; the org group is
// ManagedLocationGroupID, set on every v2 GET, so that is what is asserted.
//
// Android half: a hand-authored test double (no Android profile can be
// captured on the 26.4 tenant). It still carries OrganizationGroupId because
// internal-source/mdmv2.profiles-general.overlay.yaml gap-filled that field
// from the two test doubles; canonical 26.4 contradicts that basis. The
// overlay field stays in 26.4 (removing it would be a breaking change) and
// is queued for 26.7 removal; this half only guards that the generated
// struct still decodes it.
// Fixtures: profiles/get_android_testdouble.json, profiles/get_apple_ios_restrictions_live.json.
func TestGeneratedDeviceProfileV2Response_GeneralOrganizationGroupId(t *testing.T) {
	android := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/get_android_testdouble.json")
	if android.General == nil {
		t.Fatal("expected non-nil General")
	}
	if android.General.OrganizationGroupID == nil {
		t.Fatal("expected General.OrganizationGroupID to be present -- it was silently dropped before the fix")
	}

	// Apple: the GENUINE 26.4 iOS profile GET. Canonical 26.4 (api-doctrine
	// quirk 46): the v2 General has NO OrganizationGroupId; the org group is
	// RootLocationGroupId, serialized as ManagedLocationGroupID, a
	// non-nullable int set on every v2 GET. So assert that, and nothing
	// about OrganizationGroupId.
	apple := decodeDeviceProfileFixtureBody(t, "../testdata/mock-responses/profiles/get_apple_ios_restrictions_live.json")
	if apple.General == nil {
		t.Fatal("expected non-nil General")
	}
	if apple.General.ManagedLocationGroupID == nil || *apple.General.ManagedLocationGroupID <= 0 {
		t.Fatalf("expected General.ManagedLocationGroupID > 0 (the org group on a v2 GET), got %v", apple.General.ManagedLocationGroupID)
	}
}
