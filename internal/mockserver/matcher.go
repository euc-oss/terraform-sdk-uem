package mockserver

import (
	"net/http"
	"regexp"
	"strings"
)

// RequestMatcher matches incoming HTTP requests to mock responses.
type RequestMatcher struct {
	responses []*MockResponse
}

// NewRequestMatcher creates a new request matcher with the given responses.
func NewRequestMatcher(responses []*MockResponse) *RequestMatcher {
	return &RequestMatcher{responses: responses}
}

// Match finds the best matching response for an HTTP request
// Returns nil if no match is found.
func (rm *RequestMatcher) Match(r *http.Request) *MockResponse {
	var bestMatch *MockResponse
	var bestScore int

	for _, response := range rm.responses {
		score := scoreMatch(r, response)
		if score > bestScore {
			bestScore = score
			bestMatch = response
		}
	}

	return bestMatch
}

// scoreMatch calculates how well a request matches a response fixture.
// Higher score = better match. Returns 0 if the request cannot match.
//
// Scoring breakdown:
//   - Method match (required): +100
//   - Path match (required):   +100
//   - Query param match:       +10 each
//   - Platform match:          +50
//   - Version match:           +75
func scoreMatch(r *http.Request, response *MockResponse) int {
	score := 0

	// 1. HTTP method must match (required)
	if !strings.EqualFold(r.Method, response.Metadata.Method) {
		return 0
	}
	score += 100

	// 2. Path must match (required)
	if !matchPath(r.URL.Path, response.Metadata.Endpoint) {
		return 0
	}
	score += 100

	// 3. Query parameters (optional, but increase score if they match)
	if response.Request.QueryParams != nil {
		queryScore := matchQueryParams(r, response.Request.QueryParams)
		if queryScore == -1 {
			// Required query param missing
			return 0
		}
		score += queryScore
	}

	// 4. Platform query parameter (special case for profile search)
	if platform := r.URL.Query().Get("platform"); platform != "" && response.Metadata.Platform != "" {
		if strings.EqualFold(platform, response.Metadata.Platform) {
			score += 50
		}
	}

	// 5. Version-aware scoring: +75 if fixture version matches Accept header version
	if response.Metadata.Version != "" {
		acceptHeader := r.Header.Get("Accept")
		if reqVersion := extractVersionFromAccept(acceptHeader); reqVersion != "" {
			if reqVersion == response.Metadata.Version {
				score += 75
			}
		}
	}

	return score
}

// extractVersionFromAccept parses "application/json;version=2" → "2".
func extractVersionFromAccept(accept string) string {
	for _, part := range strings.Split(accept, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "version=") {
			return strings.TrimPrefix(part, "version=")
		}
	}
	return ""
}

// matchPath checks if a request path matches an endpoint pattern
// Supports path parameters like /api/mdm/profiles/{id}.
func matchPath(requestPath, pattern string) bool {
	// Exact match
	if requestPath == pattern {
		return true
	}

	// Convert pattern to regex
	// Replace {param} with regex pattern
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, `\{id\}`, `\d+`)
	regexPattern = strings.ReplaceAll(regexPattern, `\{uuid\}`, `[a-f0-9-]+`)
	// Replace any remaining {paramName} placeholders with a generic segment matcher
	catchAll := regexp.MustCompile(`\\\{[^}]+\\\}`)
	regexPattern = catchAll.ReplaceAllString(regexPattern, `[^/]+`)
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, requestPath)
	if err != nil {
		return false
	}

	return matched
}

// matchQueryParams checks if request query parameters match expected parameters
// Returns -1 if required param is missing, otherwise returns score based on matches.
func matchQueryParams(r *http.Request, expectedParams map[string]string) int {
	score := 0
	query := r.URL.Query()

	for key, expectedValue := range expectedParams {
		actualValue := query.Get(key)
		if actualValue == "" {
			// Required parameter missing
			return -1
		}
		if actualValue == expectedValue {
			score += 10
		}
	}

	return score
}
