package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
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
	if result.PackageIdentifier != "com.example.pQIihbwj" {
		t.Errorf("PackageIdentifier: got %q, want com.example.pQIihbwj", result.PackageIdentifier)
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

// TestProfilesV2SearchProfilesSecondCapture verifies SearchProfiles against
// the second, independently-provenanced real v2 capture
// (profiles_get_search.json), distinct from the fixture already covered by
// TestGeneratedProfilesV2SearchV2 (profiles_search_v2.json/search_v2.json).
// Loaded via an isolated single-fixture mock server so this specific
// fixture's shape is what gets exercised.
// Fixture: testdata/mock-responses/profiles/profiles_get_search.json.
func TestProfilesV2SearchProfilesSecondCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_get_search.json"))
	if err != nil {
		t.Fatalf("load profiles_get_search fixture: %v", err)
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
	if len(result.ProfileList) != 2 {
		t.Fatalf("expected 2 profiles in ProfileList, got %d", len(result.ProfileList))
	}
	if result.TotalResults == nil || *result.TotalResults != 2 {
		t.Errorf("expected TotalResults=2, got %+v", result.TotalResults)
	}
	platforms := map[string]bool{}
	for _, p := range result.ProfileList {
		platforms[p.Platform] = true
	}
	if !platforms["Apple"] || !platforms["WinRT"] {
		t.Errorf("expected Apple and WinRT platforms in results, got %+v", platforms)
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
