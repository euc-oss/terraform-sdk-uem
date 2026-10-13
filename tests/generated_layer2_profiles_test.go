package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

func newProfileService(t *testing.T) *sdk.ProfileService {
	t.Helper()
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	ctx := context.Background()
	svc, err := sdk.NewProfileService(ctx, c)
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	return svc
}

// isolatedProfileService builds a Layer-2 ProfileService over a mock server
// holding only the given fixtures. Callers load each fixture with a literal
// LoadResponseFromFile path so the bypass-fixture governance scan sees it.
func isolatedProfileService(t *testing.T, responses ...*mockserver.MockResponse) *sdk.ProfileService {
	t.Helper()
	ms := mockserver.NewMockServer(responses)
	t.Cleanup(ms.Close)
	svc, err := sdk.NewProfileService(context.Background(), mockserver.NewMockClient(t, ms))
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	return svc
}

// mustFixture returns a func that fails the test on a fixture load error,
// so a literal LoadResponseFromFile call can be passed straight in.
func mustFixture(t *testing.T) func(*mockserver.MockResponse, error) *mockserver.MockResponse {
	return func(r *mockserver.MockResponse, err error) *mockserver.MockResponse {
		t.Helper()
		if err != nil {
			t.Fatalf("load fixture: %v", err)
		}
		return r
	}
}

// TestLayer2ProfileServiceGet verifies typed GET deserialization through
// Layer 2's ProfileService: discovery via the v2 profile search, then a
// platform-routed detail GET.
//
// Each platform runs against an isolated mock server holding only its own
// search and detail inputs, so the result does not depend on which of the
// several v2 search fixtures a full-corpus load happens to serve to
// Discover's paged request.
//
// Apple iOS: the genuine 26.4.0.0 captures, the search
// profiles/profiles_get_search.json (its first "Apple" row) and the iOS
// detail profiles/get_apple_scep_12348.json.
//
// Android: profiles/search_v2.json and profiles/get_android_testdouble.json.
// The 26.4 test tenant cannot hold an Android profile (Android create is
// blocked tenant-wide; api-doctrine quirk 26), so both are PORTED-FROM-26.2
// hand-authored test doubles.
func TestLayer2ProfileServiceGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()

	t.Run("AppleIOS", func(t *testing.T) {
		var search struct {
			ProfileList  []profileSearchFixtureRow `json:"ProfileList"`
			TotalResults *int                      `json:"TotalResults"`
		}
		loadFixtureResponseBodyForTest(t, "profiles/profiles_get_search.json", &search)
		assertProfileSearchFixtureInvariants(t, "profiles/profiles_get_search.json", search.ProfileList, search.TotalResults)
		appleID := 0
		for _, p := range search.ProfileList {
			if p.Platform == "Apple" {
				appleID = p.ProfileID
				break
			}
		}
		if appleID == 0 {
			t.Fatal("profiles/profiles_get_search.json has no Apple row")
		}

		svc := isolatedProfileService(t,
			mustFixture(t)(mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/profiles_get_search.json"))),
			mustFixture(t)(mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/get_apple_scep_12348.json"))))
		result, err := svc.Get(ctx, appleID)
		if err != nil {
			t.Fatalf("Get Apple %d: %v", appleID, err)
		}
		if result.Platform != "Apple iOS" {
			t.Errorf("expected Apple iOS, got %s", result.Platform)
		}
		if result.AppleiOS == nil {
			t.Fatal("expected AppleiOS field to be non-nil")
		}
	})

	t.Run("Android", func(t *testing.T) {
		svc := isolatedProfileService(t,
			mustFixture(t)(mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/search_v2.json"))),
			mustFixture(t)(mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/get_android_testdouble.json"))))
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
	})
}

// TestLayer2ProfileCRUDLifecycle tests the full Create -> Get -> Update ->
// Delete lifecycle using the mock server's stateful CRUD handling.
func TestLayer2ProfileCRUDLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc := newProfileService(t)
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

	svc := newProfileService(t)
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

	svc := newProfileService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "InvalidPlatform", nil)
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}
