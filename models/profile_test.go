package models

import (
	"encoding/json"
	"testing"
)

func TestIDValue(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected int
	}{
		{
			name:     "valid ID value",
			json:     `{"Value": 12345}`,
			expected: 12345,
		},
		{
			name:     "zero value",
			json:     `{"Value": 0}`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id IDValue
			if err := json.Unmarshal([]byte(tt.json), &id); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if id.Value != tt.expected {
				t.Errorf("expected Value=%d, got %d", tt.expected, id.Value)
			}
		})
	}
}

func TestProfile_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name            string
		json            string
		expectedName    string
		expectedID      int
		expectedPayload bool
	}{
		{
			name: "search response format",
			json: `{
				"Id": {"Value": 12345},
				"ProfileName": "Test Profile",
				"Platform": "Android",
				"ProfileStatus": "Active"
			}`,
			expectedName: "Test Profile",
			expectedID:   12345,
		},
		{
			name: "with custom payload fields",
			json: `{
				"ProfileId": 67890,
				"ProfileName": "Passcode Profile",
				"Passcode": {"MinLength": 6},
				"Restrictions": {"AllowCamera": false}
			}`,
			expectedName:    "Passcode Profile",
			expectedID:      67890,
			expectedPayload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Profile
			if err := json.Unmarshal([]byte(tt.json), &p); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if p.GetName() != tt.expectedName {
				t.Errorf("expected name=%q, got %q", tt.expectedName, p.GetName())
			}
			if p.GetProfileID() != tt.expectedID {
				t.Errorf("expected ID=%d, got %d", tt.expectedID, p.GetProfileID())
			}
			if tt.expectedPayload && len(p.RawPayloads) == 0 {
				t.Error("expected RawPayloads to be populated")
			}
		})
	}
}

func TestProfile_GetProfileID(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected int
	}{
		{
			name:     "from nested ID",
			profile:  Profile{ID: &IDValue{Value: 111}},
			expected: 111,
		},
		{
			name:     "from ProfileID field",
			profile:  Profile{ProfileID: 222},
			expected: 222,
		},
		{
			name: "from General map",
			profile: Profile{
				General: map[string]interface{}{"ProfileId": float64(333)},
			},
			expected: 333,
		},
		{
			name:     "nested ID takes precedence",
			profile:  Profile{ID: &IDValue{Value: 111}, ProfileID: 222},
			expected: 111,
		},
		{
			name:     "zero when not set",
			profile:  Profile{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.profile.GetProfileID(); got != tt.expected {
				t.Errorf("GetProfileID() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestProfile_GetName(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected string
	}{
		{
			name:     "from ProfileName field",
			profile:  Profile{ProfileName: "Direct Name"},
			expected: "Direct Name",
		},
		{
			name: "from General map",
			profile: Profile{
				General: map[string]interface{}{"Name": "General Name"},
			},
			expected: "General Name",
		},
		{
			name:     "ProfileName takes precedence",
			profile:  Profile{ProfileName: "Direct", General: map[string]interface{}{"Name": "General"}},
			expected: "Direct",
		},
		{
			name:     "empty when not set",
			profile:  Profile{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.profile.GetName(); got != tt.expected {
				t.Errorf("GetName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProfile_GetPlatform(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected string
	}{
		{
			name:     "from Platform field",
			profile:  Profile{Platform: "Android"},
			expected: "Android",
		},
		{
			name: "from General map",
			profile: Profile{
				General: map[string]interface{}{"Platform": "Apple iOS"},
			},
			expected: "Apple iOS",
		},
		{
			name:     "Platform field takes precedence",
			profile:  Profile{Platform: "Android", General: map[string]interface{}{"Platform": "iOS"}},
			expected: "Android",
		},
		{
			name:     "empty when not set",
			profile:  Profile{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.profile.GetPlatform(); got != tt.expected {
				t.Errorf("GetPlatform() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProfile_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected string
	}{
		{
			name:     "from ProfileStatus field",
			profile:  Profile{ProfileStatus: "Active"},
			expected: "Active",
		},
		{
			name:     "from Status field",
			profile:  Profile{Status: "Inactive"},
			expected: "Inactive",
		},
		{
			name: "from General IsActive true",
			profile: Profile{
				General: map[string]interface{}{"IsActive": true},
			},
			expected: "Active",
		},
		{
			name: "from General IsActive false",
			profile: Profile{
				General: map[string]interface{}{"IsActive": false},
			},
			expected: "Inactive",
		},
		{
			name:     "ProfileStatus takes precedence",
			profile:  Profile{ProfileStatus: "Active", Status: "Inactive"},
			expected: "Active",
		},
		{
			name:     "empty when not set",
			profile:  Profile{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.profile.GetStatus(); got != tt.expected {
				t.Errorf("GetStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProfileSearchResponse(t *testing.T) {
	jsonData := `{
		"Profiles": [
			{"ProfileId": 1, "ProfileName": "Profile 1"},
			{"ProfileId": 2, "ProfileName": "Profile 2"}
		],
		"Page": 0,
		"PageSize": 10,
		"Total": 2
	}`

	var resp ProfileSearchResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(resp.Profiles))
	}
	if resp.Page != 0 {
		t.Errorf("expected Page=0, got %d", resp.Page)
	}
	if resp.Total != 2 {
		t.Errorf("expected Total=2, got %d", resp.Total)
	}
}

func TestPlatformConstants(t *testing.T) {
	// Verify platform constants are defined correctly
	if PlatformAndroid != "Android" {
		t.Errorf("PlatformAndroid = %q, want %q", PlatformAndroid, "Android")
	}
	if PlatformAppleIOS != "Apple iOS" {
		t.Errorf("PlatformAppleIOS = %q, want %q", PlatformAppleIOS, "Apple iOS")
	}
	if PlatformWindows10 != "Windows 10" {
		t.Errorf("PlatformWindows10 = %q, want %q", PlatformWindows10, "Windows 10")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusActive != "Active" {
		t.Errorf("StatusActive = %q, want %q", StatusActive, "Active")
	}
	if StatusInactive != "Inactive" {
		t.Errorf("StatusInactive = %q, want %q", StatusInactive, "Inactive")
	}
}
