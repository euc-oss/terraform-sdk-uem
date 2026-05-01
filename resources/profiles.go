package resources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/models"
)

const (
	profileAcceptV1    = "application/json;version=1"
	profileAcceptV2    = "application/json;version=2"
	profileAcceptV4    = "application/json;version=4"
	profileContentType = "application/json"
)

// ProfileService handles profile-related API operations.
type ProfileService struct {
	client *client.Client
}

// NewProfileService creates a new ProfileService.
func NewProfileService(c *client.Client) *ProfileService {
	return &ProfileService{client: c}
}

// SearchOptions contains options for searching profiles.
type SearchOptions struct {
	Page       int    // Page number (0-based)
	PageSize   int    // Number of results per page (default: 500)
	SearchText string // Search text to filter profiles
	Platform   string // Filter by platform
	Status     string // Filter by status
}

// Search searches for profiles with optional filters.
func (s *ProfileService) Search(ctx context.Context, opts *SearchOptions) (*models.ProfileSearchResponse, error) {
	// Build query parameters
	params := url.Values{}

	if opts != nil {
		if opts.PageSize > 0 {
			params.Set("pagesize", fmt.Sprintf("%d", opts.PageSize))
		} else {
			params.Set("pagesize", "500") // Default page size
		}

		if opts.Page > 0 {
			params.Set("page", fmt.Sprintf("%d", opts.Page))
		}

		if opts.SearchText != "" {
			params.Set("searchtext", opts.SearchText)
		}

		if opts.Platform != "" {
			params.Set("platform", opts.Platform)
		}

		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
	} else {
		params.Set("pagesize", "500") // Default page size
	}

	endpoint := "/api/mdm/profiles/search?" + params.Encode()

	var response models.ProfileSearchResponse
	_, err := s.client.DoRequest(ctx, "GET", endpoint, profileAcceptV1, profileContentType, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}

	return &response, nil
}

// Get retrieves a profile by ID.
// Platform is required to determine the correct API version (Linux requires v4, others use v2).
// Returns basic profile information without payloads. Use GetDetails() to get full profile with payloads.
// Returns an error if the profile is not found or if the API request fails.
func (s *ProfileService) Get(ctx context.Context, profileID int, platform string) (*models.Profile, error) {
	// Determine endpoint based on platform
	endpoint := fmt.Sprintf("/api/mdm/profiles/%d", profileID)

	// For Linux profiles, use the Linux-specific endpoint (requires v4)
	acceptHeader := profileAcceptV2
	if platform == models.PlatformLinux {
		endpoint = fmt.Sprintf("/api/mdm/profiles/linux/%d", profileID)
		acceptHeader = profileAcceptV4
	}

	var profile models.Profile
	_, err := s.client.DoRequest(ctx, "GET", endpoint, acceptHeader, profileContentType, nil, &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile %d: %w", profileID, err)
	}

	return &profile, nil
}

// GetDetails retrieves full profile details including payloads by UUID.
// This is a two-step operation:
// 1. Get the profile to retrieve the UUID (use Get() first)
// 2. Use this method with the UUID to get full details with payloads
// Returns an error if the profile is not found or if the API request fails.
func (s *ProfileService) GetDetails(ctx context.Context, profileUUID string) (*models.Profile, error) {
	endpoint := fmt.Sprintf("/api/v2/mdm/profile-payload-details/%s", profileUUID)

	var profile models.Profile
	// TODO: Upgrade to version=2 once models support v2 response format
	_, err := s.client.DoRequest(ctx, "GET", endpoint, profileAcceptV1, profileContentType, nil, &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile details for UUID %s: %w", profileUUID, err)
	}

	return &profile, nil
}

// Create creates a new profile
// Platform determines the API endpoint and version
// Returns a Profile with the ProfileID set (API returns just the ID as a number).
func (s *ProfileService) Create(ctx context.Context, platform string, request *models.ProfileCreateRequest) (*models.Profile, error) {
	// Determine endpoint based on platform
	endpoint := buildProfileEndpoint(platform, "create")

	// API returns just the profile ID as a number, not a full Profile object
	var profileID int
	_, err := s.client.DoRequest(ctx, "POST", endpoint, profileAcceptV2, profileContentType, request, &profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Return a Profile with just the ID set
	return &models.Profile{
		ProfileID: profileID,
	}, nil
}

// Update updates an existing profile
// Platform determines the API endpoint and HTTP method (Windows uses PUT, others use POST).
func (s *ProfileService) Update(ctx context.Context, platform string, profileID int, request *models.ProfileUpdateRequest) (*models.Profile, error) {
	// Determine endpoint based on platform
	endpoint := buildProfileEndpoint(platform, "update")

	// Determine HTTP method (Windows uses PUT, others use POST)
	method := "POST"
	if platform == models.PlatformWindows10 || platform == models.PlatformWindowsRugged {
		method = "PUT"
	}

	var profile models.Profile
	_, err := s.client.DoRequest(ctx, method, endpoint, profileAcceptV2, profileContentType, request, &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile %d: %w", profileID, err)
	}

	return &profile, nil
}

// Delete deletes a profile by its ID
// The API returns an empty response on success.
func (s *ProfileService) Delete(ctx context.Context, profileID int) error {
	endpoint := fmt.Sprintf("/api/mdm/profiles/%d", profileID)

	// API returns empty response on success, so pass nil for response
	_, err := s.client.DoRequest(ctx, "DELETE", endpoint, profileAcceptV2, profileContentType, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete profile %d: %w", profileID, err)
	}

	return nil
}

// buildProfileEndpoint builds the appropriate API endpoint based on platform and operation.
// This matches the forklift implementation in profile.controller.js buildUrl() function.
func buildProfileEndpoint(platform, operation string) string {
	// Map platform to API path segment (lowercase)
	var platformPath string
	switch platform {
	case models.PlatformAndroid:
		platformPath = "android"
	case models.PlatformAppleIOS:
		platformPath = "apple"
	case models.PlatformWindows10:
		platformPath = "winrt"
	case models.PlatformWindowsRugged:
		platformPath = "qnx"
	case models.PlatformAppleOSX:
		platformPath = "appleosx"
	case models.PlatformLinux:
		platformPath = "linux"
	default:
		// Fallback to generic endpoint
		return "/api/mdm/profiles"
	}

	// Build endpoint: /api/mdm/profiles/platforms/{platform}/{operation}
	return fmt.Sprintf("/api/mdm/profiles/platforms/%s/%s", platformPath, operation)
}
