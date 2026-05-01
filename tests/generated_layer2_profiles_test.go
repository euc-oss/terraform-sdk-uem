package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

func newProfileService(t *testing.T) (*sdk.ProfileService, *mockserver.MockServer) {
	t.Helper()
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	ctx := context.Background()
	svc, err := sdk.NewProfileService(ctx, c)
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	return svc, ms
}

// TestLayer2ProfileServiceGet verifies typed GET deserialization through
// Layer 2's ProfileService using the mock server.
func TestLayer2ProfileServiceGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _ := newProfileService(t)
	ctx := context.Background()

	// Get Android profile (from search_v2.json discovery + get_android_68748.json)
	result, err := svc.Get(ctx, 68748)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result.Platform != "Android" {
		t.Errorf("expected Android, got %s", result.Platform)
	}
	if result.Android == nil {
		t.Fatal("expected Android field to be non-nil")
	}
	if result.Android.General.Name != "Test Android Profile" {
		t.Errorf("unexpected name: %s", result.Android.General.Name)
	}

	// Get Apple iOS profile
	result, err = svc.Get(ctx, 68749)
	if err != nil {
		t.Fatalf("Get Apple: %v", err)
	}
	if result.Platform != "Apple iOS" {
		t.Errorf("expected Apple iOS, got %s", result.Platform)
	}
	if result.AppleiOS == nil {
		t.Fatal("expected AppleiOS field to be non-nil")
	}
}

// TestLayer2ProfileCRUDLifecycle tests the full Create -> Get -> Update ->
// Delete lifecycle using the mock server's stateful CRUD handling.
func TestLayer2ProfileCRUDLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _ := newProfileService(t)
	ctx := context.Background()

	// Create Android profile
	id, err := svc.Create(ctx, "Android", &sdk.AndroidDeviceProfileV2Entity{
		General: &sdk.GeneralPayloadV2Entity{
			Name: "Test Create",
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID from create")
	}

	// Get (exercises auto-refresh since create only added a partial entry)
	result, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get after create: %v", err)
	}
	if result.Android == nil {
		t.Error("expected Android result after create")
	}

	// Update (mock server requires General.ProfileId in body for state lookup)
	err = svc.Update(ctx, id, &sdk.AndroidDeviceProfileV2Entity{
		General: &sdk.GeneralPayloadV2Entity{
			Name:      "Updated Name",
			ProfileID: sdk.IntPtr(id),
		},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Delete
	err = svc.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// TestLayer2CreateWrongType verifies that passing the wrong platform type
// returns a descriptive error.
func TestLayer2CreateWrongType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _ := newProfileService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "Android", &sdk.AppleDeviceProfileV2Entity{})
	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

// TestLayer2CreateUnsupportedPlatform verifies that an unknown platform
// returns a descriptive error.
func TestLayer2CreateUnsupportedPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _ := newProfileService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "InvalidPlatform", nil)
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}
