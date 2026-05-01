package mockserver

// MockResponse represents a complete API response configuration loaded from JSON files.
type MockResponse struct {
	Metadata ResponseMetadata `json:"metadata"`
	Request  RequestSpec      `json:"request"`
	Response ResponseSpec     `json:"response"`
}

// ResponseMetadata contains information about the mock response.
type ResponseMetadata struct {
	Endpoint    string `json:"endpoint"`    // API endpoint pattern (e.g., "/api/mdm/profiles/search")
	Method      string `json:"method"`      // HTTP method (GET, POST, PUT, DELETE)
	Platform    string `json:"platform"`    // Platform filter (Android, iOS, etc.) - optional
	Description string `json:"description"` // Human-readable description
	Version     string `json:"version"`     // API version - optional
}

// RequestSpec defines the expected request characteristics.
type RequestSpec struct {
	Headers     map[string]string `json:"headers"`      // Expected request headers
	QueryParams map[string]string `json:"query_params"` // Expected query parameters
}

// ResponseSpec defines the response to return.
type ResponseSpec struct {
	StatusCode int               `json:"status_code"` // HTTP status code
	Headers    map[string]string `json:"headers"`     // Response headers
	Body       interface{}       `json:"body"`        // Response body (any JSON structure)
}

// ProfileState represents a profile stored in the mock server's state.
type ProfileState struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Platform    string                 `json:"platform"`
	Status      string                 `json:"status"`
	General     map[string]interface{} `json:"general"`
	Payloads    map[string]interface{} `json:"payloads"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}
