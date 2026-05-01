package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedGetProfile verifies GetDeviceProfileDetailsAsync produces
// the correct HTTP request and parses the mock response.
// Fixture: testdata/mock-responses/profiles/get_profile_12345.json
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
// Fixture: testdata/mock-responses/profiles/search_android_success.json
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
// Fixture: testdata/mock-responses/profiles/create_profile_success.json
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
