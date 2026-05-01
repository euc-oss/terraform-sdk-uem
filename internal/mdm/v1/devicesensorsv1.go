// Code generated. DO NOT EDIT.

package mdmv1

import (
	"context"
	"fmt"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"net/http"
	"net/url"
)

// DeviceSensorsV1Service provides DeviceSensorsV1 API operations.
type DeviceSensorsV1Service struct {
	client *client.Client
}

// NewDeviceSensorsV1Service creates a new DeviceSensorsV1Service.
func NewDeviceSensorsV1Service(c *client.Client) *DeviceSensorsV1Service {
	return &DeviceSensorsV1Service{client: c}
}

// BulkDeleteDeviceSensors — New - Deletes the list of device sensors based on the identifiers provided.
// Operation ID: DeviceSensorsV1_BulkDeleteDeviceSensors
// HTTP: POST /api/mdm/devicesensors/bulkdelete
func (s *DeviceSensorsV1Service) BulkDeleteDeviceSensors(
	ctx context.Context,
	request *DeviceSensorsBulkDeleteRequestV1Model,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/devicesensors/bulkdelete"
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("DeviceSensorsV1_BulkDeleteDeviceSensors: %w", err)
	}
	return headers, nil
}

// CreateDeviceSensor — New - Create a device sensor.
// Operation ID: DeviceSensorsV1_CreateDeviceSensor
// HTTP: POST /api/mdm/devicesensors
func (s *DeviceSensorsV1Service) CreateDeviceSensor(
	ctx context.Context,
	request *DeviceSensorRequestV1Model,
) (http.Header, error) {
	// Build endpoint path
	endpoint := "/api/mdm/devicesensors"
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("DeviceSensorsV1_CreateDeviceSensor: %w", err)
	}
	return headers, nil
}

// GetDeviceSensor — New - Gets the device sensor information.
// Operation ID: DeviceSensorsV1_GetDeviceSensor
// HTTP: GET /api/mdm/devicesensors/{sensorUuid}
func (s *DeviceSensorsV1Service) GetDeviceSensor(
	ctx context.Context,
	SensorUUID string,
) (http.Header, *DeviceSensorResponseV1Model, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mdm/devicesensors/%s", SensorUUID)
	var response DeviceSensorResponseV1Model
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("DeviceSensorsV1_GetDeviceSensor: %w", err)
	}
	return headers, &response, nil
}

// DeviceSensorsV1GetDeviceSensorsOptions holds optional query parameters for GetDeviceSensors.
type DeviceSensorsV1GetDeviceSensorsOptions struct {
	Page     *int // Specific page number to get. 0 based index
	PageSize *int // Maximum records per page. Default 500
}

// GetDeviceSensors — New - Gets the list of all the device sensors for the Organization Group.
// Operation ID: DeviceSensorsV1_GetDeviceSensors
// HTTP: GET /api/mdm/devicesensors/list/{organizationGroupUuid}
func (s *DeviceSensorsV1Service) GetDeviceSensors(
	ctx context.Context,
	OrganizationGroupUUID string,
	opts *DeviceSensorsV1GetDeviceSensorsOptions,
) (http.Header, *DeviceSensorListResponseV1Model, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mdm/devicesensors/list/%s", OrganizationGroupUUID)
	// Build query parameters
	if opts != nil {
		query := url.Values{}
		if opts.Page != nil {
			query.Set("page", fmt.Sprintf("%d", *opts.Page))
		}
		if opts.PageSize != nil {
			query.Set("pageSize", fmt.Sprintf("%d", *opts.PageSize))
		}
		if len(query) > 0 {
			endpoint = endpoint + "?" + query.Encode()
		}
	}
	var response DeviceSensorListResponseV1Model
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("DeviceSensorsV1_GetDeviceSensors: %w", err)
	}
	return headers, &response, nil
}

// UpdateDeviceSensor — New - Update the device sensor.
// Operation ID: DeviceSensorsV1_UpdateDeviceSensor
// HTTP: PUT /api/mdm/devicesensors/{sensorUuid}
func (s *DeviceSensorsV1Service) UpdateDeviceSensor(
	ctx context.Context,
	SensorUUID string,
	request *DeviceSensorUpdateV1Model,
) (http.Header, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mdm/devicesensors/%s", SensorUUID)
	headers, err := s.client.DoRequest(ctx, "PUT", endpoint, AcceptHeader, "application/json", request, nil)
	if err != nil {
		return nil, fmt.Errorf("DeviceSensorsV1_UpdateDeviceSensor: %w", err)
	}
	return headers, nil
}
