// Code generated. DO NOT EDIT.

package mdmv1

import (
	"context"
	"fmt"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"net/http"
	"net/url"
)

// ProfilesV1Service provides ProfilesV1 API operations.
type ProfilesV1Service struct {
	client *client.Client
}

// NewProfilesV1Service creates a new ProfilesV1Service.
func NewProfilesV1Service(c *client.Client) *ProfilesV1Service {
	return &ProfilesV1Service{client: c}
}

// ProfilesV1SearchOptions holds optional query parameters for Search.
type ProfilesV1SearchOptions struct {
	Type                *string         // Assignment Type.
	ProfileName         *string         // Profile Name.
	OrganizationGroupID *int            // Organization Group ID.
	Platform            *string         // Platform name.
	Status              *string         // Smart Group Identifier.
	Ownership           *string         // Ownership Type.
	ModifiedFrom        *client.UEMTime // DateTime, Filters the result where Profile modified date is greater than or equal to modifiedfrom value.
	ModifiedTill        *client.UEMTime // DateTime, Filters the result where Profile modified date is less than or equal to modifiedtill value.
	OrderBy             *string         // Orderby parameter name.
	SortOrder           *string         // Sorting order. Values ASC or DESC. Defaults to ASC.
	Page                *int            // Page number.
	PageSize            *int            // Maximum results which should be returned in each page.
}

// Search — Searches for all profiles applicable using the query information provided.
// Operation ID: ProfilesV1_Search
// HTTP: GET /api/mdm/profiles/search
func (s *ProfilesV1Service) Search(
	ctx context.Context,
	opts *ProfilesV1SearchOptions,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/search"
	// Build query parameters
	if opts != nil {
		query := url.Values{}
		if opts.Type != nil {
			query.Set("type", fmt.Sprintf("%s", *opts.Type))
		}
		if opts.ProfileName != nil {
			query.Set("profilename", fmt.Sprintf("%s", *opts.ProfileName))
		}
		if opts.OrganizationGroupID != nil {
			query.Set("organizationgroupid", fmt.Sprintf("%d", *opts.OrganizationGroupID))
		}
		if opts.Platform != nil {
			query.Set("platform", fmt.Sprintf("%s", *opts.Platform))
		}
		if opts.Status != nil {
			query.Set("status", fmt.Sprintf("%s", *opts.Status))
		}
		if opts.Ownership != nil {
			query.Set("ownership", fmt.Sprintf("%s", *opts.Ownership))
		}
		if opts.ModifiedFrom != nil {
			query.Set("modifiedfrom", fmt.Sprintf("%v", *opts.ModifiedFrom))
		}
		if opts.ModifiedTill != nil {
			query.Set("modifiedtill", fmt.Sprintf("%v", *opts.ModifiedTill))
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
		if len(query) > 0 {
			endpoint = endpoint + "?" + query.Encode()
		}
	}
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV1_Search: %w", err)
	}
	return headers, nil
}

// UploadCertificate — Uploads certificate into Airwatch.
// Operation ID: ProfilesV1_UploadCertificate
// HTTP: POST /api/mdm/profiles/uploadcertificate
func (s *ProfilesV1Service) UploadCertificate(
	ctx context.Context,
	request *CertificateV1,
) (http.Header, *EntityIdV1, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/uploadcertificate"
	var response EntityIdV1
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("ProfilesV1_UploadCertificate: %w", err)
	}
	return headers, &response, nil
}
