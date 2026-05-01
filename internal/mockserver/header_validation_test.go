package mockserver

import (
	"net/http"
	"testing"
)

func TestHeaderValidationRejectsWrongAcceptVersion(t *testing.T) {
	responses := []*MockResponse{
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/mdm/devicesensors/{sensorUuid}",
				Method:   "GET",
				Version:  "1",
			},
			Request: RequestSpec{
				Headers: map[string]string{
					"Accept":       "application/json;version=1",
					"Content-Type": "application/json",
				},
			},
			Response: ResponseSpec{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       map[string]interface{}{"name": "test"},
			},
		},
	}

	ms := NewMockServer(responses)
	defer ms.Close()

	// Send request with WRONG Accept version
	req, _ := http.NewRequest("GET", ms.URL()+"/api/mdm/devicesensors/abc-123", nil)
	req.Header.Set("Accept", "application/json;version=2") // Wrong!
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode == 200 {
		t.Error("Expected non-200 status for wrong Accept header, got 200")
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for wrong Accept header, got %d", resp.StatusCode)
	}
}

func TestHeaderValidationAcceptsCorrectHeaders(t *testing.T) {
	responses := []*MockResponse{
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/mdm/devicesensors/{sensorUuid}",
				Method:   "GET",
				Version:  "1",
			},
			Request: RequestSpec{
				Headers: map[string]string{
					"Accept":       "application/json;version=1",
					"Content-Type": "application/json",
				},
			},
			Response: ResponseSpec{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       map[string]interface{}{"name": "test"},
			},
		},
	}

	ms := NewMockServer(responses)
	defer ms.Close()

	req, _ := http.NewRequest("GET", ms.URL()+"/api/mdm/devicesensors/abc-123", nil)
	req.Header.Set("Accept", "application/json;version=1")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 for correct headers, got %d", resp.StatusCode)
	}
}

func TestHeaderValidationSkippedWhenNoExpectedHeaders(t *testing.T) {
	responses := []*MockResponse{
		{
			Metadata: ResponseMetadata{
				Endpoint: "/api/mdm/profiles/search",
				Method:   "GET",
			},
			// No Request.Headers — validation skipped
			Response: ResponseSpec{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       map[string]interface{}{},
			},
		},
	}

	ms := NewMockServer(responses)
	defer ms.Close()

	req, _ := http.NewRequest("GET", ms.URL()+"/api/mdm/profiles/search", nil)
	req.Header.Set("Accept", "anything-goes")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 when no expected headers defined, got %d", resp.StatusCode)
	}
}
