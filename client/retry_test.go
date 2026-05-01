package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// TestWorkspaceOneRetryPolicy tests the retry policy for various scenarios.
func TestWorkspaceOneRetryPolicy(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		wantRetry  bool
	}{
		{
			name:       "network error - should retry",
			statusCode: 0,
			err:        errors.New("network error"),
			wantRetry:  true,
		},
		{
			name:       "429 Too Many Requests - should retry",
			statusCode: http.StatusTooManyRequests,
			err:        nil,
			wantRetry:  true,
		},
		{
			name:       "500 Internal Server Error - should retry",
			statusCode: http.StatusInternalServerError,
			err:        nil,
			wantRetry:  true,
		},
		{
			name:       "502 Bad Gateway - should retry",
			statusCode: http.StatusBadGateway,
			err:        nil,
			wantRetry:  true,
		},
		{
			name:       "503 Service Unavailable - should retry",
			statusCode: http.StatusServiceUnavailable,
			err:        nil,
			wantRetry:  true,
		},
		{
			name:       "504 Gateway Timeout - should retry",
			statusCode: http.StatusGatewayTimeout,
			err:        nil,
			wantRetry:  true,
		},
		{
			name:       "200 OK - should not retry",
			statusCode: http.StatusOK,
			err:        nil,
			wantRetry:  false,
		},
		{
			name:       "201 Created - should not retry",
			statusCode: http.StatusCreated,
			err:        nil,
			wantRetry:  false,
		},
		{
			name:       "400 Bad Request - should not retry",
			statusCode: http.StatusBadRequest,
			err:        nil,
			wantRetry:  false,
		},
		{
			name:       "401 Unauthorized - should not retry",
			statusCode: http.StatusUnauthorized,
			err:        nil,
			wantRetry:  false,
		},
		{
			name:       "403 Forbidden - should not retry",
			statusCode: http.StatusForbidden,
			err:        nil,
			wantRetry:  false,
		},
		{
			name:       "404 Not Found - should not retry",
			statusCode: http.StatusNotFound,
			err:        nil,
			wantRetry:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.statusCode > 0 {
				resp = &http.Response{
					StatusCode: tt.statusCode,
				}
			}

			ctx := context.Background()
			shouldRetry, err := workspaceOneRetryPolicy(ctx, resp, tt.err)

			if shouldRetry != tt.wantRetry {
				t.Errorf("workspaceOneRetryPolicy() shouldRetry = %v, want %v", shouldRetry, tt.wantRetry)
			}

			// If there was an input error, it should be returned
			if tt.err != nil && err != tt.err {
				t.Errorf("workspaceOneRetryPolicy() err = %v, want %v", err, tt.err)
			}
		})
	}
}

// TestWorkspaceOneRetryPolicyWithCanceledContext tests that canceled contexts are not retried.
func TestWorkspaceOneRetryPolicyWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError, // Normally would retry
	}

	shouldRetry, err := workspaceOneRetryPolicy(ctx, resp, nil)

	if shouldRetry {
		t.Error("workspaceOneRetryPolicy() should not retry with canceled context")
	}

	if err != context.Canceled {
		t.Errorf("workspaceOneRetryPolicy() err = %v, want %v", err, context.Canceled)
	}
}
