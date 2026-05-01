package mockserver

import (
	"net/http"
	"testing"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		pattern     string
		expected    bool
	}{
		{
			name:        "exact match",
			requestPath: "/api/mdm/profiles/search",
			pattern:     "/api/mdm/profiles/search",
			expected:    true,
		},
		{
			name:        "no match",
			requestPath: "/api/mdm/profiles/search",
			pattern:     "/api/mdm/profiles/list",
			expected:    false,
		},
		{
			name:        "path parameter - numeric ID",
			requestPath: "/api/mdm/profiles/12345",
			pattern:     "/api/mdm/profiles/{id}",
			expected:    true,
		},
		{
			name:        "path parameter - UUID",
			requestPath: "/api/v2/mdm/profile-payload-details/a3a8ee43-7933-3906-f4dd-b7724687e37d",
			pattern:     "/api/v2/mdm/profile-payload-details/{uuid}",
			expected:    true,
		},
		{
			name:        "path parameter - no match",
			requestPath: "/api/mdm/profiles/abc",
			pattern:     "/api/mdm/profiles/{id}",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPath(tt.requestPath, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPath(%q, %q) = %v, expected %v", tt.requestPath, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatchPathArbitraryPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		reqPath  string
		pattern  string
		expected bool
	}{
		{"exact match", "/api/foo", "/api/foo", true},
		{"id placeholder", "/api/profiles/123", "/api/profiles/{id}", true},
		{"uuid placeholder", "/api/sensors/abc-def-123", "/api/sensors/{uuid}", true},
		{"sensorUuid placeholder", "/api/devicesensors/abc-def-123", "/api/devicesensors/{sensorUuid}", true},
		{"organizationGroupUuid placeholder", "/api/devicesensors/list/abc-def-123", "/api/devicesensors/list/{organizationGroupUuid}", true},
		{"multiple placeholders", "/api/foo/123/bar/abc-def", "/api/foo/{id}/bar/{someUuid}", true},
		{"no match", "/api/wrong/path", "/api/other/{id}", false},
		{"trailing slash mismatch", "/api/foo/", "/api/foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPath(tt.reqPath, tt.pattern)
			if got != tt.expected {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.reqPath, tt.pattern, got, tt.expected)
			}
		})
	}
}

func TestRequestMatcher_Match(t *testing.T) {
	responses := []*MockResponse{
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/mdm/profiles/search",
				Method:   "GET",
				Platform: "Android",
			},
			Response: ResponseSpec{
				StatusCode: 200,
			},
		},
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/mdm/profiles/search",
				Method:   "GET",
			},
			Response: ResponseSpec{
				StatusCode: 200,
			},
		},
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/mdm/profiles/{id}",
				Method:   "GET",
			},
			Response: ResponseSpec{
				StatusCode: 200,
			},
		},
	}

	matcher := NewRequestMatcher(responses)

	tests := []struct {
		name             string
		method           string
		path             string
		queryParams      map[string]string
		expectedMatch    bool
		expectedPlatform string
	}{
		{
			name:          "match profile search",
			method:        "GET",
			path:          "/api/mdm/profiles/search",
			expectedMatch: true,
		},
		{
			name:   "match profile search with platform",
			method: "GET",
			path:   "/api/mdm/profiles/search",
			queryParams: map[string]string{
				"platform": "Android",
			},
			expectedMatch:    true,
			expectedPlatform: "Android",
		},
		{
			name:          "match profile by ID",
			method:        "GET",
			path:          "/api/mdm/profiles/12345",
			expectedMatch: true,
		},
		{
			name:          "no match - wrong method",
			method:        "POST",
			path:          "/api/mdm/profiles/search",
			expectedMatch: false,
		},
		{
			name:          "no match - wrong path",
			method:        "GET",
			path:          "/api/mdm/profiles/invalid",
			expectedMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Add query parameters
			if tt.queryParams != nil {
				q := req.URL.Query()
				for key, value := range tt.queryParams {
					q.Set(key, value)
				}
				req.URL.RawQuery = q.Encode()
			}

			match := matcher.Match(req)

			if tt.expectedMatch {
				if match == nil {
					t.Errorf("Expected match but got nil")
				} else if tt.expectedPlatform != "" && match.Metadata.Platform != tt.expectedPlatform {
					t.Errorf("Expected platform %q, got %q", tt.expectedPlatform, match.Metadata.Platform)
				}
			} else {
				if match != nil {
					t.Errorf("Expected no match but got: %+v", match.Metadata)
				}
			}
		})
	}
}
