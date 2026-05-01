package client

// IsRetryable reports whether the SDK's automatic retry layer would retry
// this error. The set of retried codes mirrors workspaceOneRetryPolicy in
// retry.go: 429 (rate limit), 500 (internal server error), 502 (bad gateway),
// 503 (service unavailable), and 504 (gateway timeout). Other 5xx codes
// (501 Not Implemented, 505+ protocol/policy errors) are treated as
// non-transient and are not retried.
//
// This method exists for callers building their own retry loops on top of
// the SDK; it returns the same answer the SDK uses internally.
func (e *APIError) IsRetryable() bool {
	if e == nil {
		return false
	}
	switch e.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}
