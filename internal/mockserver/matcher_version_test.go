package mockserver

import (
	"net/http"
	"testing"
)

func TestVersionAwareScoringPrefersMatchingVersion(t *testing.T) {
	v1Fixture := &MockResponse{
		Metadata: ResponseMetadata{
			Endpoint: "/api/mdm/devicesensors/{sensorUuid}",
			Method:   "GET",
			Version:  "1",
		},
		Response: ResponseSpec{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       map[string]interface{}{"version": "v1"},
		},
	}
	v2Fixture := &MockResponse{
		Metadata: ResponseMetadata{
			Endpoint: "/api/mdm/devicesensors/{sensorUuid}",
			Method:   "GET",
			Version:  "2",
		},
		Response: ResponseSpec{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       map[string]interface{}{"version": "v2"},
		},
	}

	// Request with Accept: version=1 should prefer v1 fixture
	reqV1, _ := http.NewRequest("GET", "/api/mdm/devicesensors/abc-123", nil)
	reqV1.Header.Set("Accept", "application/json;version=1")

	scoreV1onV1 := scoreMatch(reqV1, v1Fixture)
	scoreV1onV2 := scoreMatch(reqV1, v2Fixture)

	if scoreV1onV1 <= scoreV1onV2 {
		t.Errorf("V1 request should score higher on V1 fixture: v1=%d, v2=%d", scoreV1onV1, scoreV1onV2)
	}

	// Request with Accept: version=2 should prefer v2 fixture
	reqV2, _ := http.NewRequest("GET", "/api/mdm/devicesensors/abc-123", nil)
	reqV2.Header.Set("Accept", "application/json;version=2")

	scoreV2onV1 := scoreMatch(reqV2, v1Fixture)
	scoreV2onV2 := scoreMatch(reqV2, v2Fixture)

	if scoreV2onV2 <= scoreV2onV1 {
		t.Errorf("V2 request should score higher on V2 fixture: v1=%d, v2=%d", scoreV2onV1, scoreV2onV2)
	}
}

func TestVersionAwareScoringNoVersionFieldNoBonus(t *testing.T) {
	fixture := &MockResponse{
		Metadata: ResponseMetadata{
			Endpoint: "/api/mdm/profiles/search",
			Method:   "GET",
			// No Version field — should still match, just no bonus
		},
		Response: ResponseSpec{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       map[string]interface{}{},
		},
	}

	req, _ := http.NewRequest("GET", "/api/mdm/profiles/search", nil)
	req.Header.Set("Accept", "application/json;version=1")

	score := scoreMatch(req, fixture)

	// Should still match (method + path = 200), just no version bonus
	if score < 200 {
		t.Errorf("Fixture without version should still match: score=%d", score)
	}
	if score != 200 {
		t.Errorf("Fixture without version should score exactly 200 (method+path only), got %d", score)
	}
}

func TestVersionAwareScoringMismatchNoPenalty(t *testing.T) {
	fixture := &MockResponse{
		Metadata: ResponseMetadata{
			Endpoint: "/api/mdm/devicesensors/{sensorUuid}",
			Method:   "GET",
			Version:  "2",
		},
		Response: ResponseSpec{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       map[string]interface{}{},
		},
	}

	req, _ := http.NewRequest("GET", "/api/mdm/devicesensors/abc-123", nil)
	req.Header.Set("Accept", "application/json;version=1") // Mismatch!

	score := scoreMatch(req, fixture)

	// Should still match on method+path (200), but no version bonus
	if score != 200 {
		t.Errorf("Version mismatch should score exactly 200 (method+path), got %d", score)
	}
}
