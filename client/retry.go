package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

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

// attemptsHeader carries the retry attempt count from exhaustedRetryHandler
// to handleResponse. It is set only on the SDK's own copy of the response
// and removed before the headers are returned to the caller.
const attemptsHeader = "X-Uem-Sdk-Attempts"

// exhaustedRetryHandler runs when the retry policy gives up. When the last
// attempt produced an HTTP response (e.g. a persistent 500), that response
// is passed through so handleResponse turns it into an *APIError carrying
// the server's status, errorCode and message; the attempt count travels
// with it. Without a response (a network error) or when the retry check
// itself failed (e.g. a canceled context), it returns an error as the
// library's default handler does.
func exhaustedRetryHandler(resp *http.Response, err error, numTries int) (*http.Response, error) {
	if err == nil && resp != nil {
		resp.Header.Set(attemptsHeader, strconv.Itoa(numTries))
		return resp, nil
	}
	if resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	if err == nil {
		return nil, fmt.Errorf("giving up after %d attempt(s)", numTries)
	}
	return nil, fmt.Errorf("giving up after %d attempt(s): %w", numTries, err)
}
