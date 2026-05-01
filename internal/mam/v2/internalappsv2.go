// Code generated. DO NOT EDIT.

package mamv2

import (
	"context"
	"fmt"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"net/http"
	"net/url"
)

// InternalAppsV2Service provides InternalAppsV2 API operations.
type InternalAppsV2Service struct {
	client *client.Client
}

// NewInternalAppsV2Service creates a new InternalAppsV2Service.
func NewInternalAppsV2Service(c *client.Client) *InternalAppsV2Service {
	return &InternalAppsV2Service{client: c}
}

// InternalAppsV2GetApplicationBranchCacheStatisticsAsyncOptions holds optional query parameters for GetApplicationBranchCacheStatisticsAsync.
type InternalAppsV2GetApplicationBranchCacheStatisticsAsyncOptions struct {
	SummaryOnly           *bool   // Only the summary of the BranchCache statistics for the application deployments is returned. This is the default behavior if not specified.
	ApplicationUUIDs      *string // Comma separated list of application UUIDs for the BranchCache statistics of the applications deployed to devices. At least one application must be part of the application bundle.
	DeviceUUIDs           *string // Comma separated list of device UUIDs for the BranchCache statistics of the applications deployed to those devices. Must not be specified with smart_group_uuids.
	SmartGroupUUIDs       *string // Comma separated list of smart group UUIDs for the BranchCache statistics of the applications deployed to those smart groups. Must not be specified with device_uuids.
	OrganizationGroupUUID *string // The organization group to limit the BranchCache statistics query to.
}

// GetApplicationBranchCacheStatisticsAsync — New - Gets the BranchCache statistics for the application bundle identifier.
// Operation ID: InternalAppsV2_GetApplicationBranchCacheStatisticsAsync
// HTTP: GET /api/mam/apps/internal/peerdistribution/branchcache/{bundleid}
func (s *InternalAppsV2Service) GetApplicationBranchCacheStatisticsAsync(
	ctx context.Context,
	Bundleid string,
	opts *InternalAppsV2GetApplicationBranchCacheStatisticsAsyncOptions,
) (http.Header, *ApplicationBranchCacheStatisticsModelV2, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/apps/internal/peerdistribution/branchcache/%s", Bundleid)
	// Build query parameters
	if opts != nil {
		query := url.Values{}
		if opts.SummaryOnly != nil {
			query.Set("summary_only", fmt.Sprintf("%t", *opts.SummaryOnly))
		}
		if opts.ApplicationUUIDs != nil {
			query.Set("application_uuids", fmt.Sprintf("%s", *opts.ApplicationUUIDs))
		}
		if opts.DeviceUUIDs != nil {
			query.Set("device_uuids", fmt.Sprintf("%s", *opts.DeviceUUIDs))
		}
		if opts.SmartGroupUUIDs != nil {
			query.Set("smart_group_uuids", fmt.Sprintf("%s", *opts.SmartGroupUUIDs))
		}
		if opts.OrganizationGroupUUID != nil {
			query.Set("organization_group_uuid", fmt.Sprintf("%s", *opts.OrganizationGroupUUID))
		}
		if len(query) > 0 {
			endpoint = endpoint + "?" + query.Encode()
		}
	}
	var response ApplicationBranchCacheStatisticsModelV2
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("InternalAppsV2_GetApplicationBranchCacheStatisticsAsync: %w", err)
	}
	return headers, &response, nil
}

// GetApplicationList — Gets all applications which are using the given provisioning profile.
// Operation ID: InternalAppsV2_GetApplicationList
// HTTP: GET /api/mam/apps/internal/provisionings/{uuid}
func (s *InternalAppsV2Service) GetApplicationList(
	ctx context.Context,
	UUID string,
) (http.Header, *AppListUsingProvisioningProfileModelV2, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/apps/internal/provisionings/%s", UUID)
	var response AppListUsingProvisioningProfileModelV2
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("InternalAppsV2_GetApplicationList: %w", err)
	}
	return headers, &response, nil
}

// GetInternalAppByUuid — New - Details of an internal app identified by UUID.
// Operation ID: InternalAppsV2_GetInternalAppByUuid
// HTTP: GET /api/mam/apps/internal/{uuid}
func (s *InternalAppsV2Service) GetInternalAppByUuid(
	ctx context.Context,
	UUID string,
) (http.Header, *InternalAppModelV2, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/apps/internal/%s", UUID)
	var response InternalAppModelV2
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("InternalAppsV2_GetInternalAppByUuid: %w", err)
	}
	return headers, &response, nil
}

// RenewProvisioningProfile — New - Renew the provisioning profile of all the applications using the given provisioning profile.
// Operation ID: InternalAppsV2_RenewProvisioningProfile
// HTTP: PUT /api/mam/apps/internal/provisionings/{uuid}
func (s *InternalAppsV2Service) RenewProvisioningProfile(
	ctx context.Context,
	UUID string,
) (http.Header, *ApplicationsProvisionProfileModelV2, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/apps/internal/provisionings/%s", UUID)
	var response ApplicationsProvisionProfileModelV2
	headers, err := s.client.DoRequest(ctx, "PUT", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("InternalAppsV2_RenewProvisioningProfile: %w", err)
	}
	return headers, &response, nil
}
