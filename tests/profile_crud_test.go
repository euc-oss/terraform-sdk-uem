package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/orgtest"
	"github.com/euc-oss/terraform-sdk-uem/v26/models"
	"github.com/euc-oss/terraform-sdk-uem/v26/resources"
	"github.com/joho/godotenv"
)

// TestIntegration_ProfileService_FullCRUD tests the complete CRUD lifecycle
// Pattern: Create → Verify → Update → Verify → Delete → Verify.
func TestIntegration_ProfileService_FullCRUD(t *testing.T) {
	// Unlike skipUnlessLiveMutate (tests/live_mutate_test.go), this check is
	// conditional on TEST_MODE, not unconditional: this test ALSO runs in
	// mock mode via getTestClient's own internal dispatch (below), and a
	// bare skipUnlessLiveMutate(t) call here would always skip under
	// TEST_MODE=mock (the default), silently deleting this test's existing
	// mock-mode CRUD coverage. This closes the exact same gap
	// skipUnlessLiveMutate closes for the OTHER live-only files (a real
	// mutation must never fire on TEST_MODE=live alone) without touching
	// mock-mode behavior at all.
	if os.Getenv("TEST_MODE") == "live" && getEnvOrDefault("LIVE_MUTATE", "") != "1" {
		t.Skip("TEST_MODE=live but LIVE_MUTATE=1 not set; skipping mutating live test")
	}

	// outerT is this test's own *testing.T, captured before any t.Run
	// subtest shadows the "t" identifier. registerProfileDeleteCleanup MUST
	// use this, not the subtest's t: a subtest's t.Cleanup fires the moment
	// that subtest returns, which would delete the profile after
	// Create_Profile but BEFORE Verify_Created_Profile/Update_Profile/etc.
	// run -- breaking the CRUD flow on the success path, not just guarding
	// an early return.
	outerT := t

	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		t.Skip("No .env file found, skipping integration test")
	}

	// Create client with OAuth2
	c := getTestClient(t, "oauth2")
	profileService := resources.NewProfileService(c)
	ctx := context.Background()

	// Get test records tracker
	records := GetTestRecords()

	// Generate unique profile name with timestamp
	timestamp := time.Now().Unix()
	profileName := fmt.Sprintf("SDK-Test-Profile-%d", timestamp)

	var createdProfileID int

	// ogID is a disposable child OG created under liveOrgGroupID(outerT),
	// used for both Create_Profile and Update_Profile's
	// ManagedLocationGroupID below (Task 2.4.3 / D5: this profile lives
	// under a throwaway OG instead of the shared-fixed ORG_GROUP_ID, so a
	// failed cleanup here never accumulates junk under a real, long-lived
	// org group). orgtest.NewDisposableOrgGroupOrFatal registers its own OG-level
	// delete cleanup on outerT; since t.Cleanup fires LIFO and this call
	// happens before registerProfileDeleteCleanup (below, once
	// createdProfileID is known), the profile delete fires BEFORE the OG
	// delete -- the correct dependency order.
	ogID := orgtest.NewDisposableOrgGroupOrFatal(outerT, c, liveOrgGroupID(outerT), nil)

	// ========================================================================
	// STEP 1: CREATE - Create a new test profile
	// ========================================================================
	t.Run("Create_Profile", func(t *testing.T) {
		t.Logf("Creating test profile: %s", profileName)

		// Profile structure is flat - General and payloads at same level
		// Use AndroidForWorkCustomMessages payload (simplest Android payload)
		createRequest := models.ProfileCreateRequest{
			"General": map[string]interface{}{
				"Name":                   profileName,
				"Description":            "Test profile created by terraform-sdk-uem integration tests",
				"AssignmentType":         "Auto",
				"ProfileScope":           "Production",
				"ManagedLocationGroupID": fmt.Sprintf("%d", ogID),
				"IsActive":               true,
			},
			"AndroidForWorkCustomMessages": map[string]interface{}{
				"LockScreenMessage": "SDK Test Profile",
			},
		}

		profile, err := profileService.Create(ctx, models.PlatformAndroid, &createRequest)
		if err != nil {
			if isAndroidEMMPreconditionError(err) {
				t.Skip(androidEMMPreconditionSkipReason)
			}
			t.Fatalf("Failed to create profile: %v", err)
		}

		createdProfileID = profile.GetProfileID()
		if createdProfileID == 0 {
			t.Fatal("Created profile has ID 0")
		}

		registerProfileDeleteCleanup(outerT, profileService, createdProfileID)

		// Track this profile for cleanup
		if err := records.AddProfile(createdProfileID); err != nil {
			t.Logf("Warning: Failed to track profile %d: %v", createdProfileID, err)
		}

		t.Logf("✅ Profile created successfully: ID=%d, Name=%s", createdProfileID, profile.GetName())
	})

	// ========================================================================
	// STEP 2: VERIFY CREATE - Retrieve the profile and verify it exists
	// ========================================================================
	t.Run("Verify_Created_Profile", func(t *testing.T) {
		if createdProfileID == 0 {
			t.Skip("Profile creation failed, skipping verification")
		}

		t.Logf("Verifying created profile: ID=%d", createdProfileID)

		profile, err := profileService.Get(ctx, createdProfileID, models.PlatformAndroid)
		if err != nil {
			t.Fatalf("Failed to retrieve created profile: %v", err)
		}
		if profile == nil {
			t.Fatal("Get returned nil profile with nil error")
		}

		retrievedID := profile.GetProfileID()
		if retrievedID != createdProfileID {
			t.Errorf("Expected ProfileID %d, got %d", createdProfileID, retrievedID)
		}

		retrievedName := profile.GetName()
		if retrievedName != profileName {
			t.Errorf("Expected ProfileName %s, got %s", profileName, retrievedName)
		}

		t.Logf("✅ Profile verified: ID=%d, Name=%s, Status=%s",
			retrievedID, retrievedName, profile.GetStatus())
	})

	// ========================================================================
	// STEP 3: UPDATE - Update the profile
	// ========================================================================
	t.Run("Update_Profile", func(t *testing.T) {
		if createdProfileID == 0 {
			t.Skip("Profile creation failed, skipping update")
		}

		updatedDescription := fmt.Sprintf("Updated by SDK test at %s", time.Now().Format(time.RFC3339))
		t.Logf("Updating profile %d with new description", createdProfileID)

		// Profile structure is flat - General and payloads at same level
		// For Update, ProfileId MUST be included in General section
		updateRequest := models.ProfileUpdateRequest{
			"General": map[string]interface{}{
				"ProfileId":              createdProfileID, // REQUIRED for update
				"Name":                   profileName,
				"Description":            updatedDescription,
				"AssignmentType":         "Auto",
				"ProfileScope":           "Production",
				"ManagedLocationGroupID": fmt.Sprintf("%d", ogID),
				"IsActive":               true,
			},
			"AndroidForWorkCustomMessages": map[string]interface{}{
				"LockScreenMessage": "SDK Test Profile - Updated",
			},
		}

		profile, err := profileService.Update(ctx, models.PlatformAndroid, createdProfileID, &updateRequest)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		t.Logf("✅ Profile updated successfully: ID=%d", profile.GetProfileID())
	})

	// ========================================================================
	// STEP 4: VERIFY UPDATE - Retrieve the profile and verify the update
	// ========================================================================
	t.Run("Verify_Updated_Profile", func(t *testing.T) {
		if createdProfileID == 0 {
			t.Skip("Profile creation failed, skipping verification")
		}

		t.Logf("Verifying updated profile: ID=%d", createdProfileID)

		profile, err := profileService.Get(ctx, createdProfileID, models.PlatformAndroid)
		if err != nil {
			t.Fatalf("Failed to retrieve updated profile: %v", err)
		}

		// Verify the profile still exists and has correct ID
		retrievedID := profile.GetProfileID()
		if retrievedID != createdProfileID {
			t.Errorf("Expected ProfileID %d, got %d", createdProfileID, retrievedID)
		}

		// Note: Description verification would require parsing General section
		// For now, just verify the profile is still accessible
		t.Logf("✅ Updated profile verified: ID=%d, Name=%s", retrievedID, profile.GetName())
	})

	// ========================================================================
	// STEP 5: DELETE - Delete the profile
	// ========================================================================
	t.Run("Delete_Profile", func(t *testing.T) {
		if createdProfileID == 0 {
			t.Skip("Profile creation failed, skipping delete")
		}

		t.Logf("Deleting profile: ID=%d", createdProfileID)

		err := profileService.Delete(ctx, createdProfileID)
		if err != nil {
			t.Fatalf("Failed to delete profile: %v", err)
		}

		// Remove from tracking
		if err := records.RemoveProfile(createdProfileID); err != nil {
			t.Logf("Warning: Failed to untrack profile %d: %v", createdProfileID, err)
		}

		t.Logf("✅ Profile deleted successfully: ID=%d", createdProfileID)
	})

	// ========================================================================
	// STEP 6: VERIFY DELETE - Verify the profile no longer exists
	// ========================================================================
	t.Run("Verify_Deleted_Profile", func(t *testing.T) {
		if createdProfileID == 0 {
			t.Skip("Profile creation failed, skipping verification")
		}

		t.Logf("Verifying profile deletion: ID=%d", createdProfileID)

		_, err := profileService.Get(ctx, createdProfileID, models.PlatformAndroid)
		if err == nil {
			t.Error("Expected error when retrieving deleted profile, got nil")
		} else {
			t.Logf("✅ Profile deletion verified: Get returned error as expected: %v", err)
		}
	})
}

// getEnvOrDefault returns environment variable value or default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// androidEMMPreconditionSkipReason documents the exact tenant-precondition
// carve-out this file's Create_Profile step skips on, matching the
// unverified[] reason recorded for the equivalent Android profile-update
// capture gap in tests/verified-endpoints.json (around line 1204).
const androidEMMPreconditionSkipReason = "BLOCKED on a UEM 26.2.0.0 test tenant -- not a request-shape defect. Android profile CREATE (a precondition for capturing update against a real profile) is refused tenant-wide with a live 412 (errorCode 5163) 'Android EMM Registration should be completed prior to profile setup.' Confirmed across multiple General.ProfileScope values (0/1/3/4/5 all blocked; only ProfileScope=2 'Staging' bypasses this specific error but still requires an EMM-registered tenant for any subsequent step) and multiple payload shapes (Restrictions, WifiList) -- the 412 is a tenant Android-Enterprise-registration gap, not a malformed request. No Android profile could be created to then update, so the update route itself remains genuinely uncaptured. Re-attempt once this tenant (or an equivalent one) has Android EMM/Enterprise registration completed."

// isAndroidEMMPreconditionError reports whether err is precisely the
// documented tenant Android-EMM-registration precondition failure (HTTP 412,
// errorCode 5163) -- matched on both status and error code together so a
// different 412 (or a different errorCode under some other status) still
// fails the test loudly instead of being masked as this known carve-out.
func isAndroidEMMPreconditionError(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == 412 && apiErr.ErrorCode == "5163"
}
