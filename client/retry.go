package client

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

// workspaceOneRetryPolicy determines whether a request should be retried based on the response.
// This implements Workspace ONE-specific retry logic for transient errors.
func workspaceOneRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// Always retry on network errors
	if err != nil {
		return true, err
	}

	// Don't retry if context is canceled
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Retry on specific status codes
	switch resp.StatusCode {
	case http.StatusTooManyRequests: // 429
		return true, nil
	case http.StatusInternalServerError: // 500
		return true, nil
	case http.StatusBadGateway: // 502
		return true, nil
	case http.StatusServiceUnavailable: // 503
		return true, nil
	case http.StatusGatewayTimeout: // 504
		return true, nil
	}

	// Don't retry on client errors (4xx except 429) or success (2xx, 3xx)
	return false, nil
}

// DefaultRetryPolicy is an alias for workspaceOneRetryPolicy for external use.
var DefaultRetryPolicy = retryablehttp.CheckRetry(workspaceOneRetryPolicy)
