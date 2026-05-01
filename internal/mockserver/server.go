package mockserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
)

// MockServer is an HTTP server that returns mock responses based on configuration files.
type MockServer struct {
	server  *httptest.Server
	matcher *RequestMatcher
	state   *ServerState
}

// NewMockServer creates a new mock server with the given responses.
func NewMockServer(responses []*MockResponse) *MockServer {
	ms := &MockServer{
		matcher: NewRequestMatcher(responses),
		state:   NewServerState(),
	}

	ms.server = httptest.NewServer(http.HandlerFunc(ms.handleRequest))
	return ms
}

// handleRequest processes incoming HTTP requests and returns mock responses.
func (ms *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check for stateful operations first
	if ms.handleStatefulRequest(w, r) {
		return
	}

	// Find matching response
	response := ms.matcher.Match(r)
	if response == nil {
		// No match found - return 404
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("No mock response found for %s %s", r.Method, r.URL.Path),
		}); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Error encoding response body: %v\n", err)
		}
		return
	}

	// Validate request headers against fixture expectations
	if len(response.Request.Headers) > 0 {
		for key, expectedValue := range response.Request.Headers {
			actual := r.Header.Get(key)
			if actual != expectedValue {
				http.Error(w, fmt.Sprintf("Mock server header validation failed: header %q: got %q, want %q (fixture: %s %s)",
					key, actual, expectedValue, response.Metadata.Method, response.Metadata.Endpoint), http.StatusBadRequest)
				return
			}
		}
	}

	// Set response headers
	for key, value := range response.Response.Headers {
		w.Header().Set(key, value)
	}

	// Set status code
	w.WriteHeader(response.Response.StatusCode)

	// Write response body
	if response.Response.Body != nil {
		if err := json.NewEncoder(w).Encode(response.Response.Body); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Error encoding response body: %v\n", err)
		}
	}
}

// handleStatefulRequest handles stateful CRUD operations.
// Returns true if the request was handled, false otherwise.
func (ms *MockServer) handleStatefulRequest(w http.ResponseWriter, r *http.Request) bool {
	// Only handle profile-related endpoints
	if !strings.HasPrefix(r.URL.Path, "/api/mdm/profiles") {
		return false
	}

	// Handle profile GET by ID - check if profile exists in state
	if r.Method == "GET" {
		// Extract profile ID from path
		re := regexp.MustCompile(`^/api/mdm/profiles/(\d+)$`)
		matches := re.FindStringSubmatch(r.URL.Path)
		if len(matches) == 2 {
			profileID, _ := strconv.Atoi(matches[1])
			return ms.handleProfileGet(w, r, profileID)
		}
	}

	// Handle profile CREATE
	if r.Method == "POST" && strings.Contains(r.URL.Path, "/platforms/") && strings.HasSuffix(r.URL.Path, "/create") {
		return ms.handleProfileCreate(w, r)
	}

	// Handle profile UPDATE (POST for most platforms, PUT for Windows)
	if (r.Method == "POST" || r.Method == "PUT") && strings.Contains(r.URL.Path, "/platforms/") && strings.HasSuffix(r.URL.Path, "/update") {
		return ms.handleProfileUpdate(w, r)
	}

	// Handle profile DELETE
	if r.Method == "DELETE" {
		re := regexp.MustCompile(`^/api/mdm/profiles/(\d+)$`)
		matches := re.FindStringSubmatch(r.URL.Path)
		if len(matches) == 2 {
			profileID, _ := strconv.Atoi(matches[1])
			return ms.handleProfileDelete(w, r, profileID)
		}
	}

	return false
}

// handleProfileGet handles GET /api/mdm/profiles/{id} with state awareness.
func (ms *MockServer) handleProfileGet(w http.ResponseWriter, _ *http.Request, profileID int) bool {
	// Check if this profile was created through the mock server
	ms.state.mu.RLock()
	wasCreated := ms.state.profiles[profileID] != nil || ms.state.deletedProfiles[profileID]
	ms.state.mu.RUnlock()

	// If profile was never created through mock server, fall back to static responses
	if !wasCreated {
		return false
	}

	// Profile was created through mock server - check if it still exists
	profile := ms.state.GetProfile(profileID)
	if profile == nil {
		// Profile was deleted - return 404
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Profile %d not found", profileID),
		}); err != nil {
			fmt.Printf("Error encoding response: %v\n", err)
		}
		return true
	}

	// Build response matching the actual API structure
	response := map[string]interface{}{
		"ProfileId":              profile.ID,
		"ProfileName":            profile.Name,
		"Description":            profile.Description,
		"Platform":               profile.Platform,
		"Status":                 profile.Status,
		"ProfileType":            "Device",
		"ProfileScope":           "Production",
		"ManagedLocationGroupId": "14165", // String to match Profile model
		"General":                profile.General,
	}

	// Add payloads
	for key, value := range profile.Payloads {
		response[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
	}
	return true
}

// handleProfileCreate handles POST /api/mdm/profiles/platforms/{platform}/create.
func (ms *MockServer) handleProfileCreate(w http.ResponseWriter, r *http.Request) bool {
	// Parse request body
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"}); encErr != nil {
			fmt.Printf("Error encoding error response: %v\n", encErr)
		}
		return true
	}

	// Extract profile data
	general, _ := requestBody["General"].(map[string]interface{})
	name, _ := general["Name"].(string)
	description, _ := general["Description"].(string)

	// Extract platform from URL path segment:
	// /api/mdm/profiles/platforms/{segment}/create
	platform := platformFromURL(r.URL.Path)

	// Generate profile ID (use 99999 for consistency with mock responses)
	profileID := 99999

	// Extract payloads (everything except General)
	payloads := make(map[string]interface{})
	for key, value := range requestBody {
		if key != "General" {
			payloads[key] = value
		}
	}

	// Store in state
	ms.state.CreateProfile(profileID, name, description, platform, general, payloads)

	// Return profile ID (matching actual API behavior)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(profileID); err != nil {
		fmt.Printf("Error encoding profile ID: %v\n", err)
	}
	return true
}

// handleProfileUpdate handles POST /api/mdm/profiles/platforms/{platform}/update.
func (ms *MockServer) handleProfileUpdate(w http.ResponseWriter, r *http.Request) bool {
	// Parse request body
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"}); encErr != nil {
			fmt.Printf("Error encoding error response: %v\n", encErr)
		}
		return true
	}

	// Extract profile ID from General section
	general, _ := requestBody["General"].(map[string]interface{})
	profileIDFloat, _ := general["ProfileId"].(float64)
	profileID := int(profileIDFloat)

	// Check if profile exists
	if !ms.state.ProfileExists(profileID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Profile %d not found", profileID),
		}); err != nil {
			fmt.Printf("Error encoding error response: %v\n", err)
		}
		return true
	}

	// Extract updated data
	name, _ := general["Name"].(string)
	description, _ := general["Description"].(string)

	// Extract payloads
	payloads := make(map[string]interface{})
	for key, value := range requestBody {
		if key != "General" {
			payloads[key] = value
		}
	}

	// Update in state
	if err := ms.state.UpdateProfile(profileID, name, description, general, payloads); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to update profile: %v", err),
		}); encErr != nil {
			fmt.Printf("Error encoding error response: %v\n", encErr)
		}
		return true
	}

	// Return 204 No Content (matching actual API behavior)
	w.WriteHeader(http.StatusNoContent)
	return true
}

// handleProfileDelete handles DELETE /api/mdm/profiles/{id}.
func (ms *MockServer) handleProfileDelete(w http.ResponseWriter, _ *http.Request, profileID int) bool {
	// Delete from state (no error if doesn't exist)
	ms.state.DeleteProfile(profileID)

	// Return 204 No Content (matching actual API behavior)
	w.WriteHeader(http.StatusNoContent)
	return true
}

// URL returns the base URL of the mock server.
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Close shuts down the mock server.
func (ms *MockServer) Close() {
	ms.server.Close()
}

// platformFromURL extracts the platform api_value from a profile URL.
// URL format: /api/mdm/profiles/platforms/{segment}/create or .../update.
func platformFromURL(urlPath string) string {
	// Map URL path segments to api_value strings.
	segmentToAPIValue := map[string]string{
		"android":  "Android",
		"apple":    "Apple iOS",
		"appleosx": "AppleOsX",
		"winrt":    "Windows 10",
		"qnx":      "Windows_Rugged",
		"linux":    "To do",
	}

	parts := strings.Split(urlPath, "/")
	// Expected: ["", "api", "mdm", "profiles", "platforms", "{segment}", "create"|"update"]
	if len(parts) >= 7 {
		if val, ok := segmentToAPIValue[parts[5]]; ok {
			return val
		}
	}
	return "Unknown"
}
