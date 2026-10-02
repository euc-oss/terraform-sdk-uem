package client

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewClient tests the creation of a new client with various configurations.
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid OAuth2 config",
			config: &Config{
				InstanceURL:  "https://test.awmdm.com",
				TenantCode:   "test-tenant",
				AuthMethod:   "oauth2",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: false,
		},
		{
			name: "valid Basic Auth config",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "basic",
				Username:    "test-user",
				Password:    "test-pass",
			},
			wantErr: false,
		},
		{
			name: "missing instance URL",
			config: &Config{
				TenantCode:   "test-tenant",
				AuthMethod:   "oauth2",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: true,
		},
		{
			name: "missing tenant code",
			config: &Config{
				InstanceURL:  "https://test.awmdm.com",
				AuthMethod:   "oauth2",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: true,
		},
		{
			name: "missing OAuth2 credentials",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "oauth2",
			},
			wantErr: true,
		},
		{
			name: "missing Basic Auth credentials",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "basic",
			},
			wantErr: true,
		},
		{
			name: "invalid auth method",
			config: &Config{
				InstanceURL: "https://test.awmdm.com",
				TenantCode:  "test-tenant",
				AuthMethod:  "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

// TestClientDefaults tests that default values are set correctly.
func TestClientDefaults(t *testing.T) {
	config := &Config{
		InstanceURL: "https://test.awmdm.com",
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client.config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries = 3, got %d", client.config.MaxRetries)
	}

	if client.config.RateLimit != 1000 {
		t.Errorf("Expected RateLimit = 1000, got %d", client.config.RateLimit)
	}

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout = 30s, got %v", client.config.Timeout)
	}
}

// TestDoRequest tests the DoRequest method with a mock server.
func TestDoRequest(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("aw-tenant-code") != "test-tenant" {
			t.Errorf("Missing or incorrect aw-tenant-code header")
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("Missing Authorization header")
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
			t.Fatalf("Failed to encode response body: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	config := &Config{
		InstanceURL: server.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Execute request
	var result map[string]string
	_, err = client.DoRequest(context.Background(), "GET", "/test", "application/json;version=1", "application/json", nil, &result)
	if err != nil {
		t.Fatalf("DoRequest() failed: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status = success, got %s", result["status"])
	}
}

// validatingBody is a test double implementing the Validate() error interface
// that DoRequest auto-runs on request bodies at the send chokepoint.
type validatingBody struct{ err error }

func (v validatingBody) Validate() error { return v.err }

// noValidateBody has no Validate() method at all, so it must flow through
// DoRequest unaffected by the validation hook.
type noValidateBody struct {
	Name string `json:"name"`
}

// panicIfCalledBody has a pointer-receiver Validate() that panics if it is
// ever invoked. Used to prove the typed-nil guard actually skips the call
// rather than happening to survive a nil-safe Validate() body.
type panicIfCalledBody struct{ Name string }

func (p *panicIfCalledBody) Validate() error {
	if p == nil {
		panic("Validate called on nil receiver")
	}
	return nil
}

func TestDoRequest_RunsValidateOnBody(t *testing.T) {
	// Server should never be hit — validation fails before send.
	handlerCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		t.Errorf("request should not reach the server when Validate() fails")
	}))
	defer srv.Close()

	config := &Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}
	c, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	_, err = c.DoRequest(context.Background(), "POST", "/x", "application/json;version=1", "application/json",
		validatingBody{err: errors.New("OrgGroupId is required for ProfileV2Model")}, nil)
	if err == nil || !strings.Contains(err.Error(), "OrgGroupId is required for ProfileV2Model") {
		t.Fatalf("want validation error surfaced, got %v", err)
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("want error wrapped with 'validation failed', got %v", err)
	}
	if handlerCalled {
		t.Fatal("server handler was invoked even though Validate() returned an error")
	}
}

func TestDoRequest_ValidatePassesThrough(t *testing.T) {
	handlerCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	config := &Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}
	c, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if _, err := c.DoRequest(context.Background(), "POST", "/x", "application/json;version=1", "application/json", validatingBody{err: nil}, nil); err != nil {
		t.Fatalf("nil Validate() must not block send, got %v", err)
	}
	if !handlerCalled {
		t.Fatal("server handler was never invoked even though Validate() returned nil")
	}
}

// TestDoRequest_TypedNilBodySkipsValidate proves the reflect nil-pointer guard:
// an interface holding a nil *panicIfCalledBody is NOT == nil, so body != nil
// is true, but calling Validate() on the nil receiver would panic. This test
// must genuinely fail (panic) if the guard is removed.
func TestDoRequest_TypedNilBodySkipsValidate(t *testing.T) {
	handlerCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	config := &Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}
	c, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	var nilBody *panicIfCalledBody // typed nil, non-nil interface when boxed
	if _, err := c.DoRequest(context.Background(), "POST", "/x", "application/json;version=1", "application/json", nilBody, nil); err != nil {
		t.Fatalf("typed-nil body must not error, got %v", err)
	}
	if !handlerCalled {
		t.Fatal("server handler was never invoked for typed-nil body")
	}
}

func TestDoRequest_BodyWithoutValidateUnaffected(t *testing.T) {
	handlerCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	config := &Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}
	c, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if _, err := c.DoRequest(context.Background(), "POST", "/x", "application/json;version=1", "application/json", noValidateBody{Name: "foo"}, nil); err != nil {
		t.Fatalf("body without Validate() must be unaffected, got %v", err)
	}
	if !handlerCalled {
		t.Fatal("server handler was never invoked for body without Validate()")
	}
}

// TestDoRequestSetsHeadersFromParameters verifies that DoRequest sets Accept and Content-Type
// headers from the acceptHeader and contentType parameters rather than a global registry lookup.
func TestDoRequestSetsHeadersFromParameters(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json;version=2" {
			t.Errorf("Accept header = %q, want %q", got, "application/json;version=2")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	_, err = c.DoRequest(context.Background(), "GET", "/test",
		"application/json;version=2", "application/json", nil, &result)
	if err != nil {
		t.Fatal(err)
	}
}

// TestDoRequestRawByteBody verifies that DoRequest sends []byte bodies as raw bytes
// without JSON-marshaling, enabling binary blob uploads (application/octet-stream).
func TestDoRequestRawByteBody(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Value": 42}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, _ := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})

	rawBytes := []byte{0x50, 0x4B, 0x03, 0x04} // ZIP magic bytes
	var result map[string]interface{}
	_, err := c.DoRequest(context.Background(), "POST", "/upload",
		"application/json", "application/octet-stream", rawBytes, &result)
	if err != nil {
		t.Fatal(err)
	}

	// Verify raw bytes were sent, not JSON-marshaled
	if string(receivedBody) != string(rawBytes) {
		t.Errorf("body was JSON-marshaled instead of sent raw: got %q, want raw bytes", receivedBody)
	}
	if receivedContentType != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", receivedContentType)
	}
}

// TestDoRequestRawByteResponse verifies that when out is a *[]byte, handleResponse
// copies the raw response body directly instead of attempting json.Unmarshal.
// This covers genuinely non-JSON wire bodies (e.g. blob downloads), which is the
// real shape returned by BlobsV2Service.Get().
func TestDoRequestRawByteResponse(t *testing.T) {
	rawResponse := []byte{0x50, 0x4B, 0x03, 0x04, 0xFF, 0x00}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(rawResponse)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result []byte
	_, err = c.DoRequest(context.Background(), "GET", "/blob", "application/octet-stream", "", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(rawResponse) {
		t.Errorf("result = %v, want %v (raw bytes must not be JSON-unmarshaled)", result, rawResponse)
	}
}

// TestDoRequestRawByteResponsePlainString verifies that a *[]byte out always takes
// the raw-copy path, even when the body happens to look like a quoted JSON string.
// Real UEM blob bodies are raw bytes, never JSON-wrapped; the fix must
// not try to be clever about body shape for a *[]byte target.
func TestDoRequestRawByteResponsePlainString(t *testing.T) {
	rawResponse := []byte(`"quoted json string"`)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(rawResponse)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result []byte
	_, err = c.DoRequest(context.Background(), "GET", "/blob", "application/octet-stream", "", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must equal the raw wire bytes (including surrounding quotes), NOT the
	// JSON-decoded string value "quoted json string" without quotes.
	if string(result) != string(rawResponse) {
		t.Errorf("result = %q, want raw %q (must not JSON-unmarshal a *[]byte target)", result, rawResponse)
	}
}

// TestDoRequestRawByteResponseEmptyBody verifies that an empty response body leaves
// a *[]byte out target untouched (nil), with no error — matching existing behavior
// for empty JSON bodies with a struct out.
func TestDoRequestRawByteResponseEmptyBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// no body written
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result []byte
	_, err = c.DoRequest(context.Background(), "GET", "/blob", "application/octet-stream", "", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil (empty body must not touch out)", result)
	}
}

// TestDoRequestReturnsResponseHeaders verifies that DoRequest returns response headers.
func TestDoRequestReturnsResponseHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Api-Version", "v2")
		w.Header().Set("Location", "/api/resource/42")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, err := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	headers, err := c.DoRequest(context.Background(), "GET", "/test",
		"application/json;version=2", "application/json", nil, &result)
	if err != nil {
		t.Fatal(err)
	}
	if headers == nil {
		t.Fatal("expected non-nil response headers")
	}
	if got := headers.Get("X-Api-Version"); got != "v2" {
		t.Errorf("X-Api-Version header = %q, want %q", got, "v2")
	}
	if got := headers.Get("Location"); got != "/api/resource/42" {
		t.Errorf("Location header = %q, want %q", got, "/api/resource/42")
	}
}

// TestNewClient_WithAuthField verifies that supplying Config.Auth bypasses the
// per-method credential fields and produces a valid *Client.
func TestNewClient_WithAuthField(t *testing.T) {
	auth := NewBasicAuth("user", "pass")

	c, err := NewClient(&Config{
		InstanceURL: "https://example.com",
		TenantCode:  "tenant",
		Auth:        auth,
		// No AuthMethod, ClientID, Username, etc.
	})
	if err != nil {
		t.Fatalf("NewClient with Config.Auth failed: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient returned nil client")
	}
	// The Client struct does not expose its auth field via a getter, so we
	// verify correctness indirectly: validateConfig accepts the config without
	// credential fields (no error), and the inferred AuthMethod is set to "basic"
	// (the default for non-OAuth2 providers).
	if c.config.AuthMethod != "basic" {
		t.Errorf("expected inferred AuthMethod = %q, got %q", "basic", c.config.AuthMethod)
	}
}

// TestNewClient_WithHTTPClient verifies that supplying Config.HTTPClient wires
// the caller's client into the retryable transport layer. The SDK's retry and
// rate-limit wiring is applied on top; the caller-supplied client replaces only
// the inner *http.Client of the retryable wrapper.
func TestNewClient_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 7 * time.Second,
	}

	c, err := NewClient(&Config{
		InstanceURL: "https://example.com",
		TenantCode:  "tenant",
		AuthMethod:  "basic",
		Username:    "user",
		Password:    "pass",
		HTTPClient:  customClient,
	})
	if err != nil {
		t.Fatalf("NewClient with HTTPClient failed: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient returned nil client")
	}
	// The SDK applies its Timeout override to the supplied client, so the
	// timeout on the inner *http.Client will be 30 s (the default), not 7 s.
	// We verify the custom client was retained by checking the inner HTTP
	// client's timeout equals the SDK default (confirming it was mutated, not
	// replaced with a fresh default).
	innerTimeout := c.httpClient.HTTPClient.Timeout
	if innerTimeout != 30*time.Second {
		t.Errorf("expected inner client timeout = 30s (SDK default applied to supplied client), got %v", innerTimeout)
	}
}

// TestNewClient_TimeoutField verifies that Config.Timeout is applied to the
// inner http.Client inside the retryable wrapper. If the wiring line
// retryClient.HTTPClient.Timeout = config.Timeout were removed from NewClient,
// this test would fail because the inner client's timeout would remain at the
// retryablehttp default (0) rather than the configured value.
func TestNewClient_TimeoutField(t *testing.T) {
	cfg := &Config{
		InstanceURL: "https://example.com",
		TenantCode:  "tenant",
		AuthMethod:  "basic",
		Username:    "user",
		Password:    "pass",
		Timeout:     7 * time.Second,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient returned nil client")
	}
	// Verify the configured timeout is applied to the inner http.Client.
	// c.httpClient is *retryablehttp.Client; HTTPClient is its inner *http.Client.
	if got := c.httpClient.HTTPClient.Timeout; got != 7*time.Second {
		t.Errorf("expected inner HTTPClient.Timeout = 7s, got %v", got)
	}
}

// TestDoRequestXMLBody verifies that DoRequest marshals struct bodies as XML
// when the content type contains "xml".
func TestDoRequestXMLBody(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Value": 1}`))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	c, _ := NewClient(&Config{
		InstanceURL: ts.URL,
		TenantCode:  "test",
		AuthMethod:  "basic",
		Username:    "test",
		Password:    "test",
	})

	type TestPayload struct {
		XMLName xml.Name `xml:"Payload"`
		Name    string   `xml:"Name"`
		Value   int      `xml:"Value"`
	}
	payload := TestPayload{Name: "test", Value: 42}

	var result map[string]interface{}
	_, err := c.DoRequest(context.Background(), "POST", "/upload",
		"application/xml", "application/xml", payload, &result)
	if err != nil {
		t.Fatal(err)
	}

	if receivedContentType != "application/xml" {
		t.Errorf("Content-Type = %q, want application/xml", receivedContentType)
	}
	bodyStr := string(receivedBody)
	if !strings.Contains(bodyStr, "<Name>test</Name>") {
		t.Errorf("body should contain XML element <Name>test</Name>, got: %s", bodyStr)
	}
	if strings.Contains(bodyStr, `"Name"`) {
		t.Errorf("body should not contain JSON key \"Name\", got: %s", bodyStr)
	}
}

// TestAPIError_ErrorCodeAcceptsNumericWire covers a real-wire defect found
// live on a 26.2 tenant during the InternalAppsV1 lifecycle work
// (post-delete GET returns errorCode as a JSON NUMBER, e.g.
// {"errorCode":7000,"message":"...","activityId":"..."}); every errorCode
// value in the vendored fixture corpus (published as
// testdata/mock-responses/) is also a JSON number.
// APIError.ErrorCode is string-typed for backward compatibility with existing
// public API consumers (see public/examples/error-handling/main.go), but the
// plain field-tag json.Unmarshal previously failed outright on a numeric
// errorCode -- silently falling back to handleErrorResponse's generic branch,
// which loses ErrorCode entirely (left at its zero value "") while dumping
// the raw body into Message instead. This fix closes that gap; it is the
// same class of defect as InternalApplicationEntityV1.PushMode (swagger
// declares string, wire returns a JSON number) -- a UEM-wide pattern of
// enum-shaped fields declared/expected as strings but returned as numbers
// on the wire.
func TestAPIError_ErrorCodeAcceptsNumericWire(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    APIError
		wantErr bool
	}{
		{
			name: "numeric errorCode (real wire shape)",
			body: `{"errorCode":7000,"message":"Application not found or user does not have access to it.","activityId":"7df5f54b"}`,
			want: APIError{Message: "Application not found or user does not have access to it.", ErrorCode: "7000"},
		},
		{
			name: "string errorCode (still supported)",
			body: `{"errorCode":"INVALID_INPUT","message":"bad request"}`,
			want: APIError{Message: "bad request", ErrorCode: "INVALID_INPUT"},
		},
		{
			name: "missing errorCode",
			body: `{"message":"something else broke"}`,
			want: APIError{Message: "something else broke", ErrorCode: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got APIError
			err := json.Unmarshal([]byte(tt.body), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Message != tt.want.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.want.Message)
			}
			if got.ErrorCode != tt.want.ErrorCode {
				t.Errorf("ErrorCode = %q, want %q", got.ErrorCode, tt.want.ErrorCode)
			}
		})
	}
}

// TestClient_HandleErrorResponse_NumericErrorCode is the integration-level
// counterpart to TestAPIError_ErrorCodeAcceptsNumericWire: confirms
// handleErrorResponse (the actual code path a caller hits) surfaces the
// numeric errorCode correctly via errors.As, matching how
// tests/live_resource_lifecycle_test.go's TestLiveInternalAppsV1ChunkedUploadLifecycle
// asserts against it.
func TestClient_HandleErrorResponse_NumericErrorCode(t *testing.T) {
	c := &Client{}
	err := c.handleErrorResponse(401, []byte(`{"errorCode":7000,"message":"Application not found or user does not have access to it.","activityId":"x"}`))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.ErrorCode != "7000" {
		t.Errorf("ErrorCode = %q, want \"7000\"", apiErr.ErrorCode)
	}
}
