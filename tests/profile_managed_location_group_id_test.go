package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
	"github.com/euc-oss/terraform-sdk-uem/v26/models"
	"github.com/euc-oss/terraform-sdk-uem/v26/resources"
)

// TestProfileServiceGet_ManagedLocationGroupIDFromGeneral exercises a
// live-captured (genuine, not a field-decode-shape test double) v2 GET
// /api/mdm/profiles/{id} fixture for an iOS ("apple") profile:
// profiles/get_apple_scep_12348.json, the genuine 26.4.0.0 capture that
// supersedes 26.2's profiles/profiles_get_apple_id.json for this scenario.
//
// The fixture's General section carries ManagedLocationGroupID as a nested
// integer, not at the top level. This asserts Layer-2
// resources.ProfileService.Get surfaces that value on the returned
// models.Profile.ManagedLocationGroupID. The expected value and the profile
// id are read from the fixture itself.
func TestProfileServiceGet_ManagedLocationGroupIDFromGeneral(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const fixture = "profiles/get_apple_scep_12348.json"
	var body struct {
		General struct {
			ManagedLocationGroupID json.Number `json:"ManagedLocationGroupID"`
			ProfileID              int         `json:"ProfileId"`
		} `json:"General"`
	}
	loadFixtureResponseBodyForTest(t, fixture, &body)
	wantManagedLocationGroupID := body.General.ManagedLocationGroupID.String()
	if wantManagedLocationGroupID == "" || body.General.ProfileID == 0 {
		t.Fatalf("%s: fixture lacks General.ManagedLocationGroupID/ProfileId", fixture)
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/get_apple_scep_12348.json"))
	if err != nil {
		t.Fatalf("load %s: %v", fixture, err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := resources.NewProfileService(c)
	ctx := context.Background()

	profile, err := svc.Get(ctx, body.General.ProfileID, models.PlatformAppleIOS)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	if profile.ManagedLocationGroupID != wantManagedLocationGroupID {
		t.Errorf("ManagedLocationGroupID = %q, want the fixture's %q", profile.ManagedLocationGroupID, wantManagedLocationGroupID)
	}
}
