// Package client provides the HTTP client for the Workspace ONE UEM API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

// logWriter is where debug output goes. Set to io.Discard unless debug is enabled.
var logWriter io.Writer = io.Discard

func init() {
	uemDebug := strings.ToLower(os.Getenv("UEM_DEBUG"))
	tfLog := strings.ToUpper(os.Getenv("TF_LOG"))
	if uemDebug == "true" || uemDebug == "1" || tfLog == "TRACE" || tfLog == "DEBUG" {
		logWriter = os.Stderr
	}
}

// isDebugEnabled returns true when debug logging is active.
func isDebugEnabled() bool {
	return logWriter != io.Discard
}

// Config holds the configuration for the Workspace ONE client.
type Config struct {
	// InstanceURL is the base URL of the Workspace ONE instance (e.g., "https://your-instance.awmdm.com")
	InstanceURL string

	// TenantCode is the aw-tenant-code header value (REQUIRED on all requests)
	TenantCode string

	// Auth is a pre-built authentication provider. When set, AuthMethod and the
	// per-method credential fields (ClientID, ClientSecret, Username, Password) are
	// ignored and this provider is used directly. Prefer this field for new code;
	// the credential fields remain available for backward compatibility.
	Auth AuthProvider

	// AuthMethod specifies the authentication method: "oauth2" or "basic"
	AuthMethod string

	// OAuth2 credentials (required if AuthMethod is "oauth2")
	ClientID     string
	ClientSecret string
	// OAuth2TokenURL is the OAuth2 token endpoint (optional, defaults to InstanceURL/oauth/token)
	// For Workspace ONE cloud, use: https://na.uemauth.workspaceone.com/connect/token
	OAuth2TokenURL string

	// Basic Auth credentials (required if AuthMethod is "basic")
	Username string
	Password string

	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int

	// RateLimit is the maximum number of requests per minute (default: 1000)
	RateLimit int

	// Timeout is the HTTP request timeout (default: 30 seconds)
	Timeout time.Duration

	// HTTPClient is an optional custom *http.Client to use as the inner transport.
	// When nil, a default client with sensible timeouts is used.
	HTTPClient *http.Client
}

// Client is the main client for interacting with the Workspace ONE UEM API.
type Client struct {
	config      *Config
	httpClient  *retryablehttp.Client
	auth        AuthProvider
	rateLimiter *rate.Limiter
	baseURL     *url.URL
}

// NewClient creates a new Workspace ONE client with the given configuration.
func NewClient(config *Config) (*Client, error) {
	// Validate required configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RateLimit == 0 {
		config.RateLimit = 1000
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Parse base URL
	baseURL, err := url.Parse(config.InstanceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid instance URL: %w", err)
	}

	// Create retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = config.MaxRetries
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.CheckRetry = workspaceOneRetryPolicy
	retryClient.Backoff = retryablehttp.DefaultBackoff
	retryClient.HTTPClient.Timeout = config.Timeout

	// Swap in a caller-supplied inner HTTP client when provided.
	if config.HTTPClient != nil {
		retryClient.HTTPClient = config.HTTPClient
		retryClient.HTTPClient.Timeout = config.Timeout
	}

	// Wrap the transport with debug logging when UEM_DEBUG=true or TF_LOG=TRACE/DEBUG
	if isDebugEnabled() {
		wrappedTransport := retryClient.HTTPClient.Transport
		if wrappedTransport == nil {
			wrappedTransport = http.DefaultTransport
		}
		retryClient.HTTPClient.Transport = &debugTransport{
			wrapped: wrappedTransport,
		}
	}

	// Create auth provider — use injected provider when available.
	var auth AuthProvider
	if config.Auth != nil {
		auth = config.Auth
		// Infer AuthMethod from the concrete type so addHeaders picks the right prefix.
		if config.AuthMethod == "" {
			switch config.Auth.(type) {
			case *OAuth2Auth:
				config.AuthMethod = "oauth2"
			default:
				config.AuthMethod = "basic"
			}
		}
	} else if config.AuthMethod == "oauth2" {
		tokenURL := config.OAuth2TokenURL
		if tokenURL == "" {
			// Default to instance URL + /oauth/token
			tokenURL = config.InstanceURL
		}
		auth = NewOAuth2Auth(tokenURL, config.ClientID, config.ClientSecret)
	} else {
		auth = NewBasicAuth(config.Username, config.Password)
	}

	// Create rate limiter (requests per minute to requests per second)
	ratePerSecond := float64(config.RateLimit) / 60.0
	rateLimiter := rate.NewLimiter(rate.Limit(ratePerSecond), config.RateLimit)

	client := &Client{
		config:      config,
		httpClient:  retryClient,
		auth:        auth,
		rateLimiter: rateLimiter,
		baseURL:     baseURL,
	}

	return client, nil
}

// DoRequest executes an HTTP request with authentication, rate limiting, and retry logic.
// The acceptHeader and contentType are set directly on the request; callers must supply the correct values.
// Returns the response headers and any error. Response headers are available even on successful requests,
// enabling callers to read Location, X-Api-Version, and other response metadata.
func (c *Client) DoRequest(ctx context.Context, method, endpoint, acceptHeader, contentType string, body interface{}, out interface{}) (http.Header, error) {
	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build full URL
	fullURL := c.buildURL(endpoint)

	// Marshal request body if provided
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			var bodyBytes []byte
			var marshalErr error
			if strings.Contains(contentType, "xml") {
				bodyBytes, marshalErr = xml.Marshal(body)
			} else {
				bodyBytes, marshalErr = json.Marshal(body)
			}
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", marshalErr)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}
	}

	// Create retryable request
	req, err := retryablehttp.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers using the caller-supplied accept and content-type values
	if err := c.addHeaders(req.Request, acceptHeader, contentType); err != nil {
		return nil, fmt.Errorf("failed to add headers: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log but don't override the main error
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Capture response headers before handling the body
	respHeaders := resp.Header.Clone()

	// Handle response
	if err := c.handleResponse(resp, out); err != nil {
		return respHeaders, err
	}
	return respHeaders, nil
}

// buildURL constructs the full URL for an API endpoint.
func (c *Client) buildURL(endpoint string) string {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// If endpoint contains query parameters, handle them separately
	baseURL := c.baseURL.String()
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(endpoint, "/") {
		baseURL += "/"
	}

	// Simply concatenate - don't use url.Parse to avoid double-encoding query strings
	return strings.TrimSuffix(baseURL, "/") + endpoint
}

// addHeaders adds required headers to the request.
func (c *Client) addHeaders(req *http.Request, acceptHeader, contentType string) error {
	// Get authentication token
	token, err := c.auth.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	// Add authentication header
	if c.config.AuthMethod == "oauth2" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		req.Header.Set("Authorization", "Basic "+token)
	}

	// Add headers from caller-supplied parameters
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("aw-tenant-code", c.config.TenantCode) // CRITICAL: Required on ALL requests

	return nil
}

// handleResponse processes the HTTP response and unmarshals the result.
func (c *Client) handleResponse(resp *http.Response, out interface{}) error {
	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.handleErrorResponse(resp.StatusCode, bodyBytes)
	}

	// Unmarshal response if output is provided
	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// handleErrorResponse creates an appropriate error from an error response.
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	// Try to parse error response
	var apiError APIError
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Message != "" {
		apiError.StatusCode = statusCode
		return &apiError
	}

	// Fallback to generic error
	return &APIError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf("API request failed with status %d: %s", statusCode, string(body)),
	}
}

// validateConfig validates the client configuration.
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.InstanceURL == "" {
		return fmt.Errorf("InstanceURL is required")
	}

	if config.TenantCode == "" {
		return fmt.Errorf("TenantCode is required")
	}

	// When an external AuthProvider is supplied the per-method credential fields
	// are not required — skip credential validation entirely.
	if config.Auth != nil {
		return nil
	}

	if config.AuthMethod == "" {
		config.AuthMethod = "oauth2" // Default to OAuth2
	}

	if config.AuthMethod != "oauth2" && config.AuthMethod != "basic" {
		return fmt.Errorf("AuthMethod must be 'oauth2' or 'basic'")
	}

	if config.AuthMethod == "oauth2" {
		if config.ClientID == "" {
			return fmt.Errorf("ClientID is required for OAuth2 authentication")
		}
		if config.ClientSecret == "" {
			return fmt.Errorf("ClientSecret is required for OAuth2 authentication")
		}
	}

	if config.AuthMethod == "basic" {
		if config.Username == "" {
			return fmt.Errorf("username is required for Basic authentication")
		}
		if config.Password == "" {
			return fmt.Errorf("password is required for Basic authentication")
		}
	}

	return nil
}

// APIError represents an error response from the Workspace ONE API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	ErrorCode  string `json:"errorCode"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// GetHTTPTransport returns the current HTTP transport.
func (c *Client) GetHTTPTransport() http.RoundTripper {
	if c.httpClient.HTTPClient.Transport != nil {
		return c.httpClient.HTTPClient.Transport
	}
	return http.DefaultTransport
}

// SetHTTPTransport replaces the HTTP transport (used for instrumentation).
func (c *Client) SetHTTPTransport(rt http.RoundTripper) {
	c.httpClient.HTTPClient.Transport = rt
}
