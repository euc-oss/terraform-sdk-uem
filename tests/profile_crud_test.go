package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/euc-oss/terraform-sdk-uem/models"
	"github.com/euc-oss/terraform-sdk-uem/resources"
	"github.com/joho/godotenv"
)

// TestIntegration_ProfileService_FullCRUD tests the complete CRUD lifecycle
// Pattern: Create → Verify → Update → Verify → Delete → Verify.
func TestIntegration_ProfileService_FullCRUD(t *testing.T) {
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
				"Description":            "Test profile created by go-wsone-sdk integration tests",
				"AssignmentType":         "Auto",
				"ProfileScope":           "Production",
				"ManagedLocationGroupID": getEnvOrDefault("ORG_GROUP_ID", "14165"),
				"IsActive":               true,
			},
			"AndroidForWorkCustomMessages": map[string]interface{}{
				"LockScreenMessage": "SDK Test Profile",
			},
		}

		profile, err := profileService.Create(ctx, models.PlatformAndroid, &createRequest)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		createdProfileID = profile.GetProfileID()
		if createdProfileID == 0 {
			t.Fatal("Created profile has ID 0")
		}

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
				"ManagedLocationGroupID": getEnvOrDefault("ORG_GROUP_ID", "14165"),
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
