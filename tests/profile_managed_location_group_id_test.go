package tests

import (
	"context"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
	"github.com/euc-oss/terraform-sdk-uem/models"
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// TestProfileServiceGet_ManagedLocationGroupIDFromGeneral exercises the
// live-captured (and genuine, not a field-decode-shape test double) v2 GET
// /api/mdm/profiles/{id} fixture: profiles/profiles_get_apple_id.json. Its
// verified-endpoints.json ledger row records it as a "Genuine live capture
// (commit e7dc544bd), real UUID and name (not placeholder), strict-decoded
// against mdmv2.DeviceProfileV2Entity in fixture_schema_validation_test.go."
//
// The fixture's General section carries ManagedLocationGroupID as a nested
// integer (General.ManagedLocationGroupID = 12347), not at the top level.
// This asserts Layer-2 resources.ProfileService.Get surfaces that value on
// the returned models.Profile.ManagedLocationGroupID.
func TestProfileServiceGet_ManagedLocationGroupIDFromGeneral(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const fixturePath = "../testdata/mock-responses/profiles/profiles_get_apple_id.json"
	const wantManagedLocationGroupID = "12347" // profiles_get_apple_id.json General.ManagedLocationGroupID

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir(fixturePath))
	if err != nil {
		t.Fatalf("load profiles_get_apple_id fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := resources.NewProfileService(c)
	ctx := context.Background()

	profile, err := svc.Get(ctx, 68910, models.PlatformAppleIOS)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	if profile.ManagedLocationGroupID != wantManagedLocationGroupID {
		t.Errorf("ManagedLocationGroupID = %q, want %q", profile.ManagedLocationGroupID, wantManagedLocationGroupID)
	}
}
