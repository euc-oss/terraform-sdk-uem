package client

// Debug HTTP transport for logging request/response headers.
// Uses only packages already imported by the client package:
// fmt, net/http, strings — no new dependencies.

import (
	"fmt"
	"net/http"
	"strings"
)

// sensitiveHeaders lists headers that should be redacted in debug output.
var sensitiveHeaders = []string{
	"authorization",
	"cookie",
	"set-cookie",
	"x-api-key",
	"aw-tenant-code",
}

// isSensitiveHeader returns true if the header name should be redacted.
func isSensitiveHeader(key string) bool {
	lower := strings.ToLower(key)
	for _, h := range sensitiveHeaders {
		if lower == h {
			return true
		}
	}
	return false
}

// sanitizeURL returns a URL string with query parameters removed for safe logging.
func sanitizeURL(req *http.Request) string {
	if req.URL.RawQuery == "" {
		return req.URL.Path
	}
	return req.URL.Path + "?<redacted>"
}

// debugTransport wraps an http.RoundTripper and logs request/response details
// when the UEM_DEBUG environment variable is set to "true" or "1",
// or when TF_LOG is set to TRACE or DEBUG.
type debugTransport struct {
	wrapped http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
//
//nolint:errcheck // debug logging — write errors are intentionally ignored
func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Fprintf(logWriter, "\n[UEM-SDK] >>> %s %s\n", req.Method, sanitizeURL(req))
	fmt.Fprintf(logWriter, "[UEM-SDK] Request Headers:\n")
	for key, values := range req.Header {
		for _, v := range values {
			if isSensitiveHeader(key) {
				v = "***"
			}
			fmt.Fprintf(logWriter, "  %s: %s\n", key, v)
		}
	}

	resp, err := d.wrapped.RoundTrip(req)

	if err == nil {
		fmt.Fprintf(logWriter, "[UEM-SDK] <<< %s %s\n", resp.Status, req.URL.Path)
		fmt.Fprintf(logWriter, "[UEM-SDK] Response Headers:\n")
		for key, values := range resp.Header {
			for _, v := range values {
				if isSensitiveHeader(key) {
					v = "***"
				}
				fmt.Fprintf(logWriter, "  %s: %s\n", key, v)
			}
		}
		fmt.Fprintf(logWriter, "\n")
	} else {
		fmt.Fprintf(logWriter, "[UEM-SDK] <<< ERROR: %v\n\n", err)
	}

	return resp, err
}
