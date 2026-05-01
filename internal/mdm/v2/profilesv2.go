// Code generated. DO NOT EDIT.

package mdmv2

import (
	"context"
	"fmt"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"net/http"
	"net/url"
)

// ProfilesV2Service provides ProfilesV2 API operations.
type ProfilesV2Service struct {
	client *client.Client
}

// NewProfilesV2Service creates a new ProfilesV2Service.
func NewProfilesV2Service(c *client.Client) *ProfilesV2Service {
	return &ProfilesV2Service{client: c}
}

// CreateAndroidDeviceProfileAsync — Creates an ANDROID Device Profile.
// Operation ID: ProfilesV2_CreateAndroidDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/android/create
func (s *ProfilesV2Service) CreateAndroidDeviceProfileAsync(
	ctx context.Context,
	request *AndroidDeviceProfileV2Entity,
) (http.Header, int, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/android/create"
	var response int
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("ProfilesV2_CreateAndroidDeviceProfileAsync: %w", err)
	}
	return headers, response, nil
}

// CreateAppleDeviceProfileAsync — Creates an Apple iOS device profile.
// Operation ID: ProfilesV2_CreateAppleDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/apple/create
func (s *ProfilesV2Service) CreateAppleDeviceProfileAsync(
	ctx context.Context,
	request *AppleDeviceProfileV2Entity,
) (http.Header, int, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/apple/create"
	var response int
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("ProfilesV2_CreateAppleDeviceProfileAsync: %w", err)
	}
	return headers, response, nil
}

// CreateAppleOsXDeviceProfileAsync — Creates an Apple macOS device Profile.
// Operation ID: ProfilesV2_CreateAppleOsXDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/appleosx/create
func (s *ProfilesV2Service) CreateAppleOsXDeviceProfileAsync(
	ctx context.Context,
	request *AppleOsXDeviceProfileEntityV2,
) (http.Header, int, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/appleosx/create"
	var response int
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("ProfilesV2_CreateAppleOsXDeviceProfileAsync: %w", err)
	}
	return headers, response, nil
}

// CreateQnxDeviceProfileAsync — Creates a QNX (Windows Rugged) Device Profile.
// Operation ID: ProfilesV2_CreateQnxDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/qnx/create
func (s *ProfilesV2Service) CreateQnxDeviceProfileAsync(
	ctx context.Context,
	request *QnxDeviceProfileEntityV2,
) (http.Header, int, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/qnx/create"
	var response int
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("ProfilesV2_CreateQnxDeviceProfileAsync: %w", err)
	}
	return headers, response, nil
}

// CreateWinRTDeviceProfileAsync — Creates a WinRT (Windows 10) Device Profile.
// Operation ID: ProfilesV2_CreateWinRTDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/winrt/create
func (s *ProfilesV2Service) CreateWinRTDeviceProfileAsync(
	ctx context.Context,
	request *WinRTDeviceProfileV2Entity,
) (http.Header, int, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/winrt/create"
	var response int
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("ProfilesV2_CreateWinRTDeviceProfileAsync: %w", err)
	}
	return headers, response, nil
}

// DeleteProfileAsync — Deletes the specified profile.
// Operation ID: ProfilesV2_DeleteProfileAsync
// HTTP: DELETE /api/mdm/profiles/{profileId}
func (s *ProfilesV2Service) DeleteProfileAsync(
	ctx context.Context,
	ProfileID int,
) (http.Header, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mdm/profiles/%d", ProfileID)
	headers, err := s.client.DoRequest(ctx, "DELETE", endpoint, AcceptHeader, "application/json", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV2_DeleteProfileAsync: %w", err)
	}
	return headers, nil
}

// GetDeviceProfileDetailsAsync — Gets Device Profile.
// Operation ID: ProfilesV2_GetDeviceProfileDetailsAsync
// HTTP: GET /api/mdm/profiles/{profileId}
func (s *ProfilesV2Service) GetDeviceProfileDetailsAsync(
	ctx context.Context,
	ProfileID int,
) (http.Header, *DeviceProfileV2Entity, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mdm/profiles/%d", ProfileID)
	var response DeviceProfileV2Entity
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("ProfilesV2_GetDeviceProfileDetailsAsync: %w", err)
	}
	return headers, &response, nil
}

// ProfilesV2SearchProfilesOptions holds optional query parameters for SearchProfiles.
type ProfilesV2SearchProfilesOptions struct {
	OrganizationGroupID         *int    // Organization Group ID.
	OrganizationGroupUUID       *string // Organization group uuid, based on we search the profiles, organizationgroupid will be ignored if valid organizationgroupuuid is Passed.
	Platform                    *string // Platform name.
	ProfileType                 *string // Profile Type.
	Status                      *string // Profile status (Active or Inactive).
	SearchText                  *string // search text.
	OrderBy                     *string // Orderby parameter name.
	SortOrder                   *string // Sorting order. Values ASC or DESC. Defaults to ASC.
	Page                        *int    // Page number.
	PageSize                    *int    // Maximum results which should be returned in each page.
	IncludeAndroidForWork       *bool   // It will include android for work profiles.
	IncludeAndroidmanagementapi *bool   // It will include android management api profiles.
	PayloadName                 *string // search with anyone of the payload name :Passcode, Email, Wi-Fi, Restriction, Vpn, CustomSetting, CustomAttribute, ExchangeActiveSync, ExchangeWebServices, Device, SharedDevice, Notifications, HomeScreenLayout, GoogleAccount, ManagedDomains, WebClips, BookmarkSettings, SingleAppMode, SingleSignOn, Permissions, PublicAppAutoUpdate, CustomMessages, ApplicationControl, NetworkSharePoint, DiskEncryption, KernelExtension, PrivacyPreferences, SmartCard, ConferenceRoomDisplay, WindowsLicensing, OemUpdates, WindowsAutomaticUpdates, Encryption, BIOS, UserData, Customization, PassportForWork, Scep, Firewall, Proxy, Windows10Kiosk, Antivirus, P2PBranchCacheSettings, UnifiedWriteFilter, AssignedAccess, ShortcutSettings, Certificate.
}

// SearchProfiles — Gets List of profiles based on the search criteria.
// Operation ID: ProfilesV2_SearchProfiles
// HTTP: GET /api/mdm/profiles/search
func (s *ProfilesV2Service) SearchProfiles(
	ctx context.Context,
	opts *ProfilesV2SearchProfilesOptions,
) (http.Header, *ProfileSearchResultV2Entity, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/search"
	// Build query parameters
	if opts != nil {
		query := url.Values{}
		if opts.OrganizationGroupID != nil {
			query.Set("organizationgroupid", fmt.Sprintf("%d", *opts.OrganizationGroupID))
		}
		if opts.OrganizationGroupUUID != nil {
			query.Set("organizationgroupuuid", fmt.Sprintf("%s", *opts.OrganizationGroupUUID))
		}
		if opts.Platform != nil {
			query.Set("platform", fmt.Sprintf("%s", *opts.Platform))
		}
		if opts.ProfileType != nil {
			query.Set("profiletype", fmt.Sprintf("%s", *opts.ProfileType))
		}
		if opts.Status != nil {
			query.Set("status", fmt.Sprintf("%s", *opts.Status))
		}
		if opts.SearchText != nil {
			query.Set("searchtext", fmt.Sprintf("%s", *opts.SearchText))
		}
		if opts.OrderBy != nil {
			query.Set("orderby", fmt.Sprintf("%s", *opts.OrderBy))
		}
		if opts.SortOrder != nil {
			query.Set("sortorder", fmt.Sprintf("%s", *opts.SortOrder))
		}
		if opts.Page != nil {
			query.Set("page", fmt.Sprintf("%d", *opts.Page))
		}
		if opts.PageSize != nil {
			query.Set("pagesize", fmt.Sprintf("%d", *opts.PageSize))
		}
		if opts.IncludeAndroidForWork != nil {
			query.Set("includeandroidforwork", fmt.Sprintf("%t", *opts.IncludeAndroidForWork))
		}
		if opts.IncludeAndroidmanagementapi != nil {
			query.Set("include_androidmanagementapi", fmt.Sprintf("%t", *opts.IncludeAndroidmanagementapi))
		}
		if opts.PayloadName != nil {
			query.Set("payloadName", fmt.Sprintf("%s", *opts.PayloadName))
		}
		if len(query) > 0 {
			endpoint = endpoint + "?" + query.Encode()
		}
	}
	var response ProfileSearchResultV2Entity
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("ProfilesV2_SearchProfiles: %w", err)
	}
	return headers, &response, nil
}

// UpdateAndroidDeviceProfileAsync — Updates an ANDROID Device Profile. If the CreateNewVersion key is empty or false, a new Profile version will not be created but AssignedSmartGroups, RootLocationGroup, AssignedGeofenceArea and AssignedSchedule will be saved and published. Else if it's true, new version of the profile will be created and published.
// Operation ID: ProfilesV2_UpdateAndroidDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/android/update
func (s *ProfilesV2Service) UpdateAndroidDeviceProfileAsync(
	ctx context.Context,
	request *AndroidDeviceProfileV2Entity,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/android/update"
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV2_UpdateAndroidDeviceProfileAsync: %w", err)
	}
	return headers, nil
}

// UpdateAppleDeviceProfileAsync — Updates an Apple iOS device profile.
// Operation ID: ProfilesV2_UpdateAppleDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/apple/update
func (s *ProfilesV2Service) UpdateAppleDeviceProfileAsync(
	ctx context.Context,
	request *AppleDeviceProfileV2Entity,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/apple/update"
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV2_UpdateAppleDeviceProfileAsync: %w", err)
	}
	return headers, nil
}

// UpdateAppleOsXDeviceProfileAsync — Updates an Apple macOS device Profile.
// Operation ID: ProfilesV2_UpdateAppleOsXDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/appleosx/update
func (s *ProfilesV2Service) UpdateAppleOsXDeviceProfileAsync(
	ctx context.Context,
	request *AppleOsXDeviceProfileEntityV2,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/appleosx/update"
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV2_UpdateAppleOsXDeviceProfileAsync: %w", err)
	}
	return headers, nil
}

// UpdateQnxDeviceProfileAsync — Updates an existing QNX (Windows Rugged) Device Profile.
// Operation ID: ProfilesV2_UpdateQnxDeviceProfileAsync
// HTTP: PUT /api/mdm/profiles/platforms/qnx/update
func (s *ProfilesV2Service) UpdateQnxDeviceProfileAsync(
	ctx context.Context,
	request *QnxDeviceProfileEntityV2,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/qnx/update"
	headers, err := s.client.DoRequest(ctx, "PUT", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV2_UpdateQnxDeviceProfileAsync: %w", err)
	}
	return headers, nil
}

// UpdateWinRTDeviceProfileAsync — Updates an existing WinRT (Windows 10) Device Profile.
// Operation ID: ProfilesV2_UpdateWinRTDeviceProfileAsync
// HTTP: PUT /api/mdm/profiles/platforms/winrt/update
func (s *ProfilesV2Service) UpdateWinRTDeviceProfileAsync(
	ctx context.Context,
	request *WinRTDeviceProfileV2Entity,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/winrt/update"
	headers, err := s.client.DoRequest(ctx, "PUT", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV2_UpdateWinRTDeviceProfileAsync: %w", err)
	}
	return headers, nil
}
