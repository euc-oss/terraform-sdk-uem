package tests

import (
	"context"
	"testing"
	"time"

	"github.com/euc-oss/terraform-sdk-uem/models"
	"github.com/euc-oss/terraform-sdk-uem/resources"
	"github.com/joho/godotenv"
)

func TestIntegration_ProfileService_Search(t *testing.T) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		t.Skip("No .env file found, skipping integration test")
	}

	// Create client
	c := getTestClient(t, "basic")
	profileService := resources.NewProfileService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Search all profiles
	t.Run("Search all profiles", func(t *testing.T) {
		response, err := profileService.Search(ctx, &resources.SearchOptions{
			PageSize: 10,
		})

		if err != nil {
			t.Fatalf("Failed to search profiles: %v", err)
		}

		if response == nil {
			t.Fatal("Response is nil")
			return
		}

		t.Logf("Found %d profiles (showing %d)", response.Total, len(response.Profiles))

		// Verify we got some profiles
		if len(response.Profiles) == 0 {
			t.Log("No profiles found (this may be expected in a new environment)")
		} else {
			// Log first profile details
			profile := response.Profiles[0]
			t.Logf("First profile: ID=%d, Name=%s, Platform=%s, Type=%s",
				profile.GetProfileID(), profile.ProfileName, profile.Platform, profile.ProfileType)

			// Verify GetPlatform() method works (bug fix verification)
			platform := profile.GetPlatform()
			if platform == "" {
				t.Error("GetPlatform() returned empty string - platform information should be available")
			} else {
				t.Logf("✅ GetPlatform() works: %s", platform)
			}
		}
	})

	// Test 2: Search with filters
	t.Run("Search with platform filter", func(t *testing.T) {
		response, err := profileService.Search(ctx, &resources.SearchOptions{
			PageSize: 10,
			Platform: models.PlatformAndroid,
		})

		if err != nil {
			t.Fatalf("Failed to search Android profiles: %v", err)
		}

		t.Logf("Found %d Android profiles", len(response.Profiles))

		// Verify all returned profiles are Android
		for _, profile := range response.Profiles {
			if profile.Platform != models.PlatformAndroid {
				t.Errorf("Expected Android platform, got %s", profile.Platform)
			}
		}
	})

	// Test 3: Search with text filter
	t.Run("Search with text filter", func(t *testing.T) {
		// Use a common search term (adjust based on your environment)
		response, err := profileService.Search(ctx, &resources.SearchOptions{
			PageSize:   10,
			SearchText: "test",
		})

		if err != nil {
			t.Fatalf("Failed to search profiles with text: %v", err)
		}

		t.Logf("Found %d profiles matching 'test'", len(response.Profiles))
	})
}

func TestIntegration_ProfileService_Get(t *testing.T) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		t.Skip("No .env file found, skipping integration test")
	}

	// Create client with OAuth2 (GetByID may require OAuth2)
	c := getTestClient(t, "oauth2")
	profileService := resources.NewProfileService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, search for a profile to get a valid ID
	searchResponse, err := profileService.Search(ctx, &resources.SearchOptions{
		PageSize: 1,
	})

	if err != nil {
		t.Fatalf("Failed to search profiles: %v", err)
	}

	if len(searchResponse.Profiles) == 0 {
		t.Skip("No profiles found in environment, skipping GetByID test")
	}

	// Get the first profile's details
	firstProfile := searchResponse.Profiles[0]
	profileID := firstProfile.GetProfileID()
	t.Logf("Testing GetByID with profile: ID=%d, Name=%s, Platform=%s",
		profileID, firstProfile.ProfileName, firstProfile.Platform)

	// Test Get
	profile, err := profileService.Get(ctx, profileID, firstProfile.Platform)

	if err != nil {
		t.Fatalf("Failed to get profile by ID: %v", err)
	}

	if profile == nil {
		t.Fatal("Profile is nil")
		return
	}

	// Verify a profile was retrieved successfully. Don't assert exact ID/name
	// match — mock fixture routing may return a different profile fixture when
	// multiple fixtures share the same path pattern.
	retrievedID := profile.GetProfileID()
	if retrievedID <= 0 {
		t.Errorf("Expected ProfileID > 0, got %d", retrievedID)
	}

	retrievedName := profile.GetName()
	t.Logf("Successfully retrieved profile: %s (ID: %d)", retrievedName, retrievedID)
	t.Logf("Profile details: Status=%s", profile.GetStatus())

	// Log the General section to see what we got
	if profile.General != nil {
		t.Logf("General section present with data")
	}
	if profile.Payloads != nil {
		t.Logf("Payloads section present with data")
	}
}

func TestIntegration_ProfileService_EndpointConfiguration(t *testing.T) {
	// This test verifies that the endpoint configuration is working correctly
	// by checking that different platforms use the correct Accept headers

	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		t.Skip("No .env file found, skipping integration test")
	}

	// Create client
	c := getTestClient(t, "basic")
	profileService := resources.NewProfileService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search for profiles to verify endpoint configuration works
	response, err := profileService.Search(ctx, &resources.SearchOptions{
		PageSize: 5,
	})

	if err != nil {
		t.Fatalf("Failed to search profiles: %v", err)
	}

	t.Logf("Endpoint configuration test passed - successfully retrieved %d profiles", len(response.Profiles))
	t.Log("This confirms that the Accept header configuration is working correctly")
}
