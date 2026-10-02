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
	return rm.MatchPreferring(r, "")
}

// preferredFixtureBonus is added to a matching fixture whose SourceFile is
// the preferred one. It is below the 100 points an exact literal path earns
// over a template, so a concrete-id fixture still wins.
const preferredFixtureBonus = 99

// MatchPreferring is Match, except that a matching fixture whose SourceFile
// equals preferred gets preferredFixtureBonus. An empty preferred is Match.
func (rm *RequestMatcher) MatchPreferring(r *http.Request, preferred string) *MockResponse {
	var bestMatch *MockResponse
	var bestScore int

	for _, response := range rm.responses {
		score := scoreMatch(r, response)
		if score > 0 && preferred != "" && response.SourceFile == preferred {
			score += preferredFixtureBonus
		}
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

	// 2. Path must match (required). Exact literal match scores higher than
	// template (regex) match so a fixture with endpoint "/api/mdm/profiles/12346"
	// wins over a fixture with "/api/mdm/profiles/{id}" when the request is
	// exactly /api/mdm/profiles/12346.
	pathScore := pathMatchScore(r.URL.Path, response.Metadata.Endpoint)
	if pathScore == 0 {
		return 0
	}
	score += pathScore

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
	return pathMatchScore(requestPath, pattern) > 0
}

// exactPathScore is pathMatchScore's result for an exact literal match.
const exactPathScore = 200

// pathMatchScore returns 200 when requestPath exactly equals pattern,
// 100 when pattern is a template (contains {placeholder}) that matches,
// and 0 when there is no match. Exact matches always beat template matches
// so that a fixture whose endpoint bakes in a specific anonymized id wins
// over a fixture with a generic {id} placeholder.
func pathMatchScore(requestPath, pattern string) int {
	if requestPath == pattern {
		return exactPathScore
	}

	// Convert pattern to regex.
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, `\{id\}`, `\d+`)
	regexPattern = strings.ReplaceAll(regexPattern, `\{uuid\}`, `[a-f0-9-]+`)
	catchAll := regexp.MustCompile(`\\\{[^}]+\\\}`)
	regexPattern = catchAll.ReplaceAllString(regexPattern, `[^/]+`)
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, requestPath)
	if err != nil || !matched {
		return 0
	}
	return 100
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
