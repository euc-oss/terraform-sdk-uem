// Code generated. DO NOT EDIT.

package mamv2

import (
	"context"
	"fmt"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"net/http"
	"net/url"
)

// BlobsV2Service provides BlobsV2 API operations.
type BlobsV2Service struct {
	client *client.Client
}

// NewBlobsV2Service creates a new BlobsV2Service.
func NewBlobsV2Service(c *client.Client) *BlobsV2Service {
	return &BlobsV2Service{client: c}
}

// Delete — New - Deletes a blob by Guid
// Operation ID: BlobsV2_Delete
// HTTP: DELETE /api/mam/blobs/{blobId}
func (s *BlobsV2Service) Delete(
	ctx context.Context,
	BlobID string,
) (http.Header, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/blobs/%s", BlobID)
	headers, err := s.client.DoRequest(ctx, "DELETE", endpoint, AcceptHeader, "application/json", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("BlobsV2_Delete: %w", err)
	}
	return headers, nil
}

// Get — Gets a blob by the Guid.
// Operation ID: BlobsV2_Get
// HTTP: GET /api/mam/blobs/{blobId}
func (s *BlobsV2Service) Get(
	ctx context.Context,
	BlobID string,
) (http.Header, []byte, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/blobs/%s", BlobID)
	var response []byte
	headers, err := s.client.DoRequest(ctx, "GET", endpoint, AcceptHeader, "application/json", nil, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("BlobsV2_Get: %w", err)
	}
	return headers, response, nil
}

// Head — New - Gets a blob contents information by the Guid
// Operation ID: BlobsV2_Head
// HTTP: HEAD /api/mam/blobs/{blobId}
func (s *BlobsV2Service) Head(
	ctx context.Context,
	BlobID string,
) (http.Header, error) {
	// Build endpoint path
	endpoint := fmt.Sprintf("/api/mam/blobs/%s", BlobID)
	headers, err := s.client.DoRequest(ctx, "HEAD", endpoint, AcceptHeader, "application/json", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("BlobsV2_Head: %w", err)
	}
	return headers, nil
}

// BlobsV2UploadBlobAsyncOptions holds optional query parameters for UploadBlobAsync.
type BlobsV2UploadBlobAsyncOptions struct {
	FileName             string  // Name of the file being uploaded(Required)
	OrganizationGroupID  int     // Organization Group ID integer identifying the customer or container(Required)
	ModuleType           *string // Module type of the blob. For application blobs, module type is required and should be set as Application.
	FileLink             *string // Path of the file to upload. Required if blob file is not submitted
	AccessVia            *string // Access type. If EIS, content gateway ID is required and validated for the Organization Group Id
	ContentGatewayID     *int    // Content Gateway ID of the repository to save to. Required if accessVia is EIS.
	DownloadFileFromLink *bool   // Set to true if application needs to be downloaded from the link.
	Username             *string // required when accessVia is EIS and downloadfilefromlink is true
	Password             *string // required when accessVia is EIS and downloadfilefromlink is true
	SHA256FileHash       *string // SHA-256 of the file to validate the integrity of the downloaded file from the link.
}

// UploadBlobAsync — New - Create a new blob with attached file
// Operation ID: BlobsV2_UploadBlobAsync
// HTTP: POST /api/mam/blobs/uploadblob
func (s *BlobsV2Service) UploadBlobAsync(
	ctx context.Context,
	request []byte,
	opts *BlobsV2UploadBlobAsyncOptions,
) (http.Header, *EntityV1ModelV2, error) {
	// Build endpoint path
	endpoint := "/api/mam/blobs/uploadblob"
	// Validate required query parameters
	if opts == nil {
		return nil, nil, fmt.Errorf("BlobsV2_UploadBlobAsync: opts must not be nil (contains required parameters)")
	}
	// Build query parameters
	if opts != nil {
		query := url.Values{}
		query.Set("fileName", fmt.Sprintf("%s", opts.FileName))
		query.Set("organizationGroupId", fmt.Sprintf("%d", opts.OrganizationGroupID))
		if opts.ModuleType != nil {
			query.Set("moduleType", fmt.Sprintf("%s", *opts.ModuleType))
		}
		if opts.FileLink != nil {
			query.Set("fileLink", fmt.Sprintf("%s", *opts.FileLink))
		}
		if opts.AccessVia != nil {
			query.Set("accessVia", fmt.Sprintf("%s", *opts.AccessVia))
		}
		if opts.ContentGatewayID != nil {
			query.Set("contentGatewayId", fmt.Sprintf("%d", *opts.ContentGatewayID))
		}
		if opts.DownloadFileFromLink != nil {
			query.Set("downloadfilefromlink", fmt.Sprintf("%t", *opts.DownloadFileFromLink))
		}
		if opts.Username != nil {
			query.Set("username", fmt.Sprintf("%s", *opts.Username))
		}
		if opts.Password != nil {
			query.Set("password", fmt.Sprintf("%s", *opts.Password))
		}
		if opts.SHA256FileHash != nil {
			query.Set("SHA256FileHash", fmt.Sprintf("%s", *opts.SHA256FileHash))
		}
		if len(query) > 0 {
			endpoint = endpoint + "?" + query.Encode()
		}
	}
	var response EntityV1ModelV2
	headers, err := s.client.DoRequest(ctx, "POST", endpoint, AcceptHeader, "application/octet-stream", request, &response)
	if err != nil {
		return nil, nil, fmt.Errorf("BlobsV2_UploadBlobAsync: %w", err)
	}
	return headers, &response, nil
}
