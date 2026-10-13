package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestEnterpriseAppRepositoryV2GetApplicationsDetails verifies
// EnterpriseAppRepositoryV2Service.GetApplicationsDetailsAsync against the
// genuine live capture (packageId "0-don.clippy", version "1.7.3"), and that
// the required query parameters (doctrine quirk 15) are actually sent and
// matched by the mock server, not just present in Go struct fields.
// Fixture: testdata/mock-responses/mam-apps/enterpriseapprepository_get_packages.json.
func TestEnterpriseAppRepositoryV2GetApplicationsDetails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/enterpriseapprepository_get_packages.json"))
	if err != nil {
		t.Fatalf("load enterpriseapprepository_get_packages fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewEnterpriseAppRepositoryV2Service(c)
	ctx := context.Background()

	_, result, err := svc.GetApplicationsDetailsAsync(ctx, &sdk.EnterpriseAppRepositoryV2GetApplicationsDetailsAsyncOptions{
		PackageID: "0-don.clippy",
		Version:   "1.7.3",
	})
	if err != nil {
		t.Fatalf("GetApplicationsDetailsAsync failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil response")
	}
	// Expected PackageIdentifier is read from the fixture's own body (the
	// genuine 26.4 recapture), not hardcoded.
	var want struct {
		PackageIdentifier string `json:"package_identifier"`
	}
	loadFixtureResponseBodyForTest(t, "mam-apps/enterpriseapprepository_get_packages.json", &want)
	// Without this the comparison below would pass on two empty identifiers.
	if want.PackageIdentifier == "" {
		t.Fatal("fixture invariant: enterpriseapprepository_get_packages.json package_identifier is empty, want a package identifier")
	}
	if result.PackageIdentifier != want.PackageIdentifier {
		t.Errorf("PackageIdentifier: got %q, want %s", result.PackageIdentifier, want.PackageIdentifier)
	}
	if result.PackageVersion != "1.7.3" {
		t.Errorf("PackageVersion: got %q, want 1.7.3", result.PackageVersion)
	}
	if len(result.Installers) != 1 {
		t.Fatalf("expected 1 installer, got %d", len(result.Installers))
	}

	// Control: doctrine quirk 15 requires packageId/version; the generated
	// client enforces this itself by rejecting a nil opts before ever
	// issuing the request.
	_, _, err = svc.GetApplicationsDetailsAsync(ctx, nil)
	if err == nil {
		t.Error("expected an error when opts (containing required packageId/version) is nil, got nil")
	}
}

// profileSearchFixtureRow is the structural slice of a v2 profile-search row
// that assertProfileSearchFixtureInvariants checks.
type profileSearchFixtureRow struct {
	ProfileID   int    `json:"ProfileId"`
	ProfileUUID string `json:"ProfileUuid"`
	ProfileName string `json:"ProfileName"`
	Platform    string `json:"Platform"`
}

// assertProfileSearchFixtureInvariants checks the fixed structure of a captured
// v2 profile search: a positive TotalResults covering every row returned, and
// rows that are identified and carry a known platform string.
func assertProfileSearchFixtureInvariants(t *testing.T, fixture string, rows []profileSearchFixtureRow, total *int) {
	t.Helper()
	if total == nil {
		t.Fatalf("fixture invariant: %s has no TotalResults", fixture)
	}
	if *total <= 0 || *total < len(rows) {
		t.Fatalf("fixture invariant: %s TotalResults = %d, want > 0 and >= the %d rows returned", fixture, *total, len(rows))
	}
	for i, r := range rows {
		if r.ProfileID <= 0 || r.ProfileUUID == "" || r.ProfileName == "" {
			t.Fatalf("fixture invariant: %s ProfileList[%d] has ProfileId %d, ProfileUuid %q, ProfileName %q, want all set", fixture, i, r.ProfileID, r.ProfileUUID, r.ProfileName)
		}
		switch r.Platform {
		case "Apple", "Apple iOS", "AppleOsX", "Android", "WinRT", "Windows 10", "Windows_Rugged", "Qnx", "To do":
		default:
			t.Fatalf("fixture invariant: %s ProfileList[%d].Platform = %q, want a known profile platform", fixture, i, r.Platform)
		}
	}
}

// TestProfilesV2SearchProfilesSecondCapture verifies SearchProfiles against
// a second, independently captured real v2 search response
// (profiles_get_search.json), distinct from the fixture covered by
// TestGeneratedProfilesV2SearchV2. On 26.4 this is the genuine 26.4.0.0
// capture; expected values are read from the fixture's own body. Loaded
// via an isolated single-fixture mock server so this specific fixture's
// shape is what gets exercised.
// Fixture: testdata/mock-responses/profiles/profiles_get_search.json.
func TestProfilesV2SearchProfilesSecondCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const fixture = "profiles/profiles_get_search.json"
	var want struct {
		ProfileList  []profileSearchFixtureRow `json:"ProfileList"`
		TotalResults *int                      `json:"TotalResults"`
	}
	loadFixtureResponseBodyForTest(t, fixture, &want)
	if len(want.ProfileList) == 0 || want.TotalResults == nil {
		t.Fatalf("%s: fixture has no ProfileList rows or no TotalResults", fixture)
	}
	assertProfileSearchFixtureInvariants(t, fixture, want.ProfileList, want.TotalResults)

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_get_search.json"))
	if err != nil {
		t.Fatalf("load %s: %v", fixture, err)
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
	if len(result.ProfileList) != len(want.ProfileList) {
		t.Fatalf("ProfileList: got %d profiles, want the fixture's %d", len(result.ProfileList), len(want.ProfileList))
	}
	if result.TotalResults == nil || *result.TotalResults != *want.TotalResults {
		t.Errorf("TotalResults: got %+v, want the fixture's %d", result.TotalResults, *want.TotalResults)
	}
	for i, p := range result.ProfileList {
		if p.Platform != want.ProfileList[i].Platform {
			t.Errorf("ProfileList[%d].Platform: got %q, want the fixture's %q", i, p.Platform, want.ProfileList[i].Platform)
		}
	}
}

// TestBlobsV1DeleteBlob verifies BlobsV1Service.DeleteBlobAsync against the
// genuine live capture confirming doctrine quirk 16 (BlobsV1's DELETE takes
// the numeric Value id, not a uuid).
// Fixture: testdata/mock-responses/blobs/delete_blob_v1.json.
func TestBlobsV1DeleteBlob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/blobs/delete_blob_v1.json"))
	if err != nil {
		t.Fatalf("load delete_blob_v1 fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewBlobsV1Service(c)
	ctx := context.Background()

	headers, err := svc.DeleteBlobAsync(ctx, 12345)
	if err != nil {
		t.Fatalf("DeleteBlobAsync failed: %v", err)
	}
	if headers == nil {
		t.Fatal("expected non-nil response headers")
	}
}
