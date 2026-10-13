package mockserver

// MockResponse represents a complete API response configuration loaded from JSON files.
type MockResponse struct {
	Metadata ResponseMetadata `json:"metadata"`
	Request  RequestSpec      `json:"request"`
	Response ResponseSpec     `json:"response"`

	// SourceFile is the base name of the file the fixture was loaded from,
	// set by LoadResponseFromFile. It lets stateful routing pick one of
	// several fixtures that share a route (see preferredSmartGroupFixture).
	SourceFile string `json:"-"`
}

// ResponseMetadata contains information about the mock response.
type ResponseMetadata struct {
	Endpoint    string `json:"endpoint"`    // API endpoint pattern (e.g., "/api/mdm/profiles/search")
	Method      string `json:"method"`      // HTTP method (GET, POST, PUT, DELETE)
	Platform    string `json:"platform"`    // Platform filter (Android, iOS, etc.) - optional
	Description string `json:"description"` // Human-readable description
	Version     string `json:"version"`     // API version - optional
	// Provenance is set on a fixture carried over from an earlier release's
	// corpus rather than captured on this release (see FixtureProvenance).
	Provenance *FixtureProvenance `json:"provenance,omitempty"`
}

// PortedFrom262 is the provenance class of a fixture ported unchanged from
// the 26.2 corpus and not re-verified on this release.
const PortedFrom262 = "PORTED-FROM-26.2"

// FixtureProvenance records where a ported fixture came from.
type FixtureProvenance struct {
	Class         string  `json:"class"`
	SourceTenant  string  `json:"source_tenant"`
	SourceVersion string  `json:"source_version"`
	SourceCommit  string  `json:"source_commit"`
	ReverifiedOn  *string `json:"reverified_on"`
}

// IsPorted reports whether the fixture is a port rather than a genuine
// capture of this release.
func (m ResponseMetadata) IsPorted() bool {
	return m.Provenance != nil && m.Provenance.Class == PortedFrom262
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
