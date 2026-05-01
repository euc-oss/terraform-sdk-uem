package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/models"
)

// createTestClient creates a client pointing to a test server.
func createTestClient(t *testing.T, handler http.HandlerFunc) (*client.Client, *httptest.Server) {
	server := httptest.NewServer(handler)

	cfg := &client.Config{
		InstanceURL: server.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return c, server
}

func TestNewProfileService(t *testing.T) {
	c, server := createTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	svc := NewProfileService(c)
	if svc == nil {
		t.Fatal("NewProfileService returned nil")
	}
	if svc.client != c {
		t.Error("ProfileService client not set correctly")
	}
}

func TestProfileService_Search(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/mdm/profiles/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Check query params
		if r.URL.Query().Get("pagesize") == "" {
			t.Error("expected pagesize param")
		}

		resp := models.ProfileSearchResponse{
			Profiles: []models.Profile{
				{ProfileID: 1, ProfileName: "Test Profile"},
			},
			Page:     0,
			PageSize: 500,
			Total:    1,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	result, err := svc.Search(context.Background(), nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(result.Profiles))
	}
	if result.Profiles[0].ProfileName != "Test Profile" {
		t.Errorf("unexpected profile name: %s", result.Profiles[0].ProfileName)
	}
}

func TestProfileService_Search_WithOptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("platform") != "Android" {
			t.Errorf("expected platform=Android, got %s", q.Get("platform"))
		}
		if q.Get("pagesize") != "10" {
			t.Errorf("expected pagesize=10, got %s", q.Get("pagesize"))
		}
		if q.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", q.Get("page"))
		}

		resp := models.ProfileSearchResponse{Profiles: []models.Profile{}, Total: 0}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	opts := &SearchOptions{
		Platform: "Android",
		PageSize: 10,
		Page:     2,
	}
	_, err := svc.Search(context.Background(), opts)
	if err != nil {
		t.Fatalf("Search with options failed: %v", err)
	}
}

func TestProfileService_Get(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/mdm/profiles/12345" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := models.Profile{ProfileID: 12345, ProfileName: "Test Profile"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	profile, err := svc.Get(context.Background(), 12345, models.PlatformAndroid)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if profile.ProfileID != 12345 {
		t.Errorf("expected ProfileID=12345, got %d", profile.ProfileID)
	}
}

func TestProfileService_Get_Linux(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Linux profiles use a different endpoint
		if r.URL.Path != "/api/mdm/profiles/linux/12345" {
			t.Errorf("unexpected path for Linux: %s", r.URL.Path)
		}

		resp := models.Profile{ProfileID: 12345, Platform: models.PlatformLinux}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	_, err := svc.Get(context.Background(), 12345, models.PlatformLinux)
	if err != nil {
		t.Fatalf("Get Linux profile failed: %v", err)
	}
}

func TestProfileService_GetDetails(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/mdm/profile-payload-details/test-uuid-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := models.Profile{ProfileID: 12345, Payloads: []interface{}{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	_, err := svc.GetDetails(context.Background(), "test-uuid-123")
	if err != nil {
		t.Fatalf("GetDetails failed: %v", err)
	}
}

func TestProfileService_Create(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/mdm/profiles/platforms/android/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// API returns just the profile ID
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(99999)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	req := &models.ProfileCreateRequest{}
	profile, err := svc.Create(context.Background(), models.PlatformAndroid, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if profile.ProfileID != 99999 {
		t.Errorf("expected ProfileID=99999, got %d", profile.ProfileID)
	}
}

func TestProfileService_Delete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/mdm/profiles/12345" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}

	c, server := createTestClient(t, handler)
	defer server.Close()

	svc := NewProfileService(c)
	err := svc.Delete(context.Background(), 12345)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestBuildProfileEndpoint(t *testing.T) {
	tests := []struct {
		platform  string
		operation string
		expected  string
	}{
		{models.PlatformAndroid, "create", "/api/mdm/profiles/platforms/android/create"},
		{models.PlatformAppleIOS, "create", "/api/mdm/profiles/platforms/apple/create"},
		{models.PlatformWindows10, "update", "/api/mdm/profiles/platforms/winrt/update"},
		{models.PlatformWindowsRugged, "create", "/api/mdm/profiles/platforms/qnx/create"},
		{models.PlatformAppleOSX, "create", "/api/mdm/profiles/platforms/appleosx/create"},
		{models.PlatformLinux, "create", "/api/mdm/profiles/platforms/linux/create"},
		{"Unknown", "create", "/api/mdm/profiles"},
	}

	for _, tt := range tests {
		t.Run(tt.platform+"_"+tt.operation, func(t *testing.T) {
			result := buildProfileEndpoint(tt.platform, tt.operation)
			if result != tt.expected {
				t.Errorf("buildProfileEndpoint(%q, %q) = %q, want %q",
					tt.platform, tt.operation, result, tt.expected)
			}
		})
	}
}
