// Code generated. DO NOT EDIT.

package mdmv4

import (
	"context"
	"fmt"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"net/http"
)

// ProfilesV4Service provides ProfilesV4 API operations.
type ProfilesV4Service struct {
	client *client.Client
}

// NewProfilesV4Service creates a new ProfilesV4Service.
func NewProfilesV4Service(c *client.Client) *ProfilesV4Service {
	return &ProfilesV4Service{client: c}
}

// UpdateLinuxDeviceProfileAsync — Updates an existing Linux Device Profile.
// Operation ID: ProfilesV4_UpdateLinuxDeviceProfileAsync
// HTTP: POST /api/mdm/profiles/platforms/linux/update
func (s *ProfilesV4Service) UpdateLinuxDeviceProfileAsync(
	ctx context.Context,
	request *LinuxDeviceProfileEntity1V4,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/profiles/platforms/linux/update"
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("ProfilesV4_UpdateLinuxDeviceProfileAsync: %w", err)
	}
	return headers, nil
}
