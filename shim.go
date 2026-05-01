// Package sdk provides a clean, idiomatic public API for the Workspace ONE UEM SDK.
// This file is hand-written and must NOT carry a "Code generated. DO NOT EDIT." header.
package sdk

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/models"
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// OAuth2Config holds the fields needed to create an OAuth2 auth provider.
// Pass to NewOAuth2Auth to obtain an AuthProvider.
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
}

// Config is the ergonomic public-surface configuration for the SDK client.
// It maps cleanly to the lower-level client configuration without exposing
// internal auth-method strings. Use with NewClient to create a *client.Client.
type Config struct {
	// BaseURL is the base URL of the Workspace ONE UEM instance
	// (e.g., "https://your-instance.awmdm.com").
	BaseURL string

	// Auth is the authentication provider. Construct one with
	// NewOAuth2Auth or sdk.NewBasicAuth.
	Auth AuthProvider

	// TenantCode is the aw-tenant-code header value required on all requests.
	TenantCode string

	// HTTPClient is an optional custom *http.Client to use as the inner
	// transport. When nil, a default client with sensible timeouts is used.
	HTTPClient *http.Client

	// MaxRetries is the maximum number of retry attempts for transient errors.
	// Zero means use the SDK default (3 retries).
	MaxRetries int

	// RateLimit is the maximum number of requests per minute. Zero means use
	// the SDK default (1000 requests per minute).
	RateLimit int

	// Timeout is the per-request HTTP timeout. Zero means use the SDK default
	// (30 seconds).
	Timeout time.Duration
}

// NewOAuth2Auth constructs an OAuth2 auth provider from an OAuth2Config.
// Returns an error if any required field is empty.
func NewOAuth2Auth(cfg OAuth2Config) (AuthProvider, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TokenURL == "" {
		return nil, errors.New("OAuth2Config requires ClientID, ClientSecret, and TokenURL")
	}
	return client.NewOAuth2Auth(cfg.TokenURL, cfg.ClientID, cfg.ClientSecret), nil
}

// NewClient constructs a new SDK client from a public Config. This is the
// recommended entry point; it wraps the lower-level client.NewClient with a
// cleaner, ergonomic surface.
func NewClient(cfg Config) (*client.Client, error) {
	inner := client.Config{
		InstanceURL: cfg.BaseURL,
		TenantCode:  cfg.TenantCode,
		Auth:        cfg.Auth,
		HTTPClient:  cfg.HTTPClient,
		MaxRetries:  cfg.MaxRetries,
		RateLimit:   cfg.RateLimit,
		Timeout:     cfg.Timeout,
	}
	return client.NewClient(&inner)
}

// ListOptions controls pagination for list calls.
type ListOptions struct {
	// Page is the zero-based page number (default: 0).
	Page int
	// PageSize is the number of results per page (default: 500).
	PageSize int
}

// ListProfiles returns profiles as a flat slice, using the supplied pagination
// options. Pass nil to use defaults (page 0, page size 500).
func ListProfiles(ctx context.Context, c *client.Client, opts *ListOptions) ([]*models.Profile, error) {
	svc := resources.NewProfileService(c)

	var searchOpts *resources.SearchOptions
	if opts != nil {
		searchOpts = &resources.SearchOptions{
			Page:     opts.Page,
			PageSize: opts.PageSize,
		}
	}

	resp, err := svc.Search(ctx, searchOpts)
	if err != nil {
		return nil, err
	}

	profiles := make([]*models.Profile, len(resp.Profiles))
	for i := range resp.Profiles {
		p := resp.Profiles[i]
		profiles[i] = &p
	}
	return profiles, nil
}

// GetProfile fetches a profile by ID and platform.
//
// Platform must be one of the wsone.Platform* constants (PlatformAppleiOS,
// PlatformAppleOsX, PlatformAndroid, PlatformWindows10, PlatformWindowsRugged,
// PlatformLinux). The platform is required because the API endpoint is
// platform-specific.
func GetProfile(ctx context.Context, c *client.Client, profileID int, platform string) (*models.Profile, error) {
	return resources.NewProfileService(c).Get(ctx, profileID, platform)
}

// CreateProfile creates a new profile for the given platform.
//
// Platform must be one of the wsone.Platform* constants. The request body
// shape is platform-specific; see models.ProfileCreateRequest for the field set.
func CreateProfile(ctx context.Context, c *client.Client, platform string, request *models.ProfileCreateRequest) (*models.Profile, error) {
	return resources.NewProfileService(c).Create(ctx, platform, request)
}

// UpdateProfile updates an existing profile.
//
// Platform must be one of the wsone.Platform* constants. The profileID
// parameter identifies the existing profile; request carries the new
// payload. Note: Windows updates use HTTP PUT; all other platforms use
// POST. This is handled internally — callers do not need to choose.
func UpdateProfile(ctx context.Context, c *client.Client, platform string, profileID int, request *models.ProfileUpdateRequest) (*models.Profile, error) {
	return resources.NewProfileService(c).Update(ctx, platform, profileID, request)
}

// DeleteProfile deletes the profile with the given ID.
//
// Unlike Get, Create, and Update, the Delete endpoint is not platform-specific —
// the API derives the platform from the stored profile metadata.
func DeleteProfile(ctx context.Context, c *client.Client, profileID int) error {
	return resources.NewProfileService(c).Delete(ctx, profileID)
}
