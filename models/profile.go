package models

import (
	"encoding/json"
	"time"
)

// IDValue represents the nested ID structure in API responses.
type IDValue struct {
	Value int `json:"Value"`
}

// Profile represents a Workspace ONE UEM configuration profile
// Note: Some fields may be returned as strings or ints depending on the API version
// The profile structure varies by endpoint:
// - /api/mdm/profiles/search: Returns search-specific fields (ProfileName, Platform, etc.)
// - /api/mdm/profiles/{id}: Returns General + payload sections as top-level keys (Passcode, EmailList, etc.)
// - /api/v2/mdm/profile-payload-details/{uuid}: Returns Payloads as an array.
type Profile struct {
	// Search response fields
	ID                      *IDValue  `json:"Id,omitempty"`        // Nested ID structure
	ProfileID               int       `json:"ProfileId,omitempty"` // Direct ID (used in some responses)
	ProfileName             string    `json:"ProfileName,omitempty"`
	Platform                string    `json:"Platform,omitempty"`
	ProfileType             string    `json:"ProfileType,omitempty"`
	ProfileStatus           string    `json:"ProfileStatus,omitempty"` // Status in search responses
	ProfileScope            string    `json:"ProfileScope,omitempty"`
	OrganizationGroupID     string    `json:"OrganizationGroupId,omitempty"`    // Can be string or int
	ManagedLocationGroupID  string    `json:"ManagedLocationGroupId,omitempty"` // Can be string or int
	Status                  string    `json:"Status,omitempty"`
	CreatedOn               time.Time `json:"CreatedOn,omitempty"`
	ModifiedOn              time.Time `json:"ModifiedOn,omitempty"`
	Description             string    `json:"Description,omitempty"`
	AssignedDeviceCount     int       `json:"AssignedDeviceCount,omitempty"`
	InstalledDeviceCount    int       `json:"InstalledDeviceCount,omitempty"`
	FailedDeviceCount       int       `json:"FailedDeviceCount,omitempty"`
	NotInstalledDeviceCount int       `json:"NotInstalledDeviceCount,omitempty"`
	ManagedBy               string    `json:"ManagedBy,omitempty"`
	Type                    string    `json:"Type,omitempty"` // Assignment type in search responses

	// Get response fields - General section
	General interface{} `json:"General,omitempty"`

	// Get response fields - Payload sections (dynamic keys like Passcode, EmailList, WifiList, etc.)
	// These are captured in RawPayloads map since they vary by platform and profile type
	RawPayloads map[string]interface{} `json:"-"` // Not directly unmarshaled

	// GetDetails response field - Payloads as array
	Payloads interface{} `json:"Payloads,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to capture dynamic payload fields.
func (p *Profile) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Define known non-payload fields
	knownFields := map[string]bool{
		"Id": true, "ProfileId": true, "ProfileName": true, "Platform": true,
		"ProfileType": true, "ProfileStatus": true, "ProfileScope": true,
		"OrganizationGroupId": true, "ManagedLocationGroupId": true, "Status": true,
		"CreatedOn": true, "ModifiedOn": true, "Description": true,
		"AssignedDeviceCount": true, "InstalledDeviceCount": true,
		"FailedDeviceCount": true, "NotInstalledDeviceCount": true,
		"ManagedBy": true, "Type": true, "General": true, "Payloads": true,
	}

	// Extract payload fields (anything not in knownFields)
	p.RawPayloads = make(map[string]interface{})
	for key, value := range raw {
		if !knownFields[key] {
			p.RawPayloads[key] = value
		}
	}

	// Now unmarshal into the struct normally using a type alias to avoid recursion
	type Alias Profile
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	return json.Unmarshal(data, aux)
}

// GetProfileID returns the profile ID, checking multiple possible locations.
func (p *Profile) GetProfileID() int {
	// Check nested ID structure (used in search responses)
	if p.ID != nil {
		return p.ID.Value
	}
	// Check direct ProfileID field
	if p.ProfileID != 0 {
		return p.ProfileID
	}
	// Check General.ProfileId (used in Get responses)
	if generalMap, ok := p.General.(map[string]interface{}); ok {
		if profileID, ok := generalMap["ProfileId"].(float64); ok {
			return int(profileID)
		}
	}
	return 0
}

// GetName returns the profile name from various possible locations.
func (p *Profile) GetName() string {
	if p.ProfileName != "" {
		return p.ProfileName
	}
	// Check General.Name (used in GetByID responses)
	if generalMap, ok := p.General.(map[string]interface{}); ok {
		if name, ok := generalMap["Name"].(string); ok {
			return name
		}
	}
	return ""
}

// GetPlatform returns the profile platform from various possible locations.
func (p *Profile) GetPlatform() string {
	if p.Platform != "" {
		return p.Platform
	}
	// Check General.Platform (used in GetByID responses)
	if generalMap, ok := p.General.(map[string]interface{}); ok {
		if platform, ok := generalMap["Platform"].(string); ok {
			return platform
		}
	}
	return ""
}

// GetStatus returns the profile status from various possible locations.
func (p *Profile) GetStatus() string {
	if p.ProfileStatus != "" {
		return p.ProfileStatus
	}
	if p.Status != "" {
		return p.Status
	}
	// Check General.IsActive (used in GetByID responses)
	if generalMap, ok := p.General.(map[string]interface{}); ok {
		if isActive, ok := generalMap["IsActive"].(bool); ok {
			if isActive {
				return "Active"
			}
			return "Inactive"
		}
	}
	return ""
}

// ProfileSearchResponse represents the response from profile search endpoint.
type ProfileSearchResponse struct {
	Profiles []Profile `json:"Profiles,omitempty"`
	Page     int       `json:"Page,omitempty"`
	PageSize int       `json:"PageSize,omitempty"`
	Total    int       `json:"Total,omitempty"`
}

// ProfileListItem represents a profile in a list response.
type ProfileListItem struct {
	ProfileID           int    `json:"ProfileId,omitempty"`
	ProfileName         string `json:"ProfileName,omitempty"`
	Platform            string `json:"Platform,omitempty"`
	ProfileType         string `json:"ProfileType,omitempty"`
	OrganizationGroupID string `json:"OrganizationGroupId,omitempty"` // Can be string or int
	Status              string `json:"Status,omitempty"`
	AssignedDeviceCount int    `json:"AssignedDeviceCount,omitempty"`
}

// ProfileCreateRequest represents a request to create a profile.
// The structure is flat - General and payload sections (Passcode, Restrictions, etc.) are at the same level.
// Example:
//
//	{
//	  "General": {...},
//	  "Passcode": {...},
//	  "Restrictions": {...}
//	}
type ProfileCreateRequest map[string]interface{}

// ProfileGeneral contains general profile information.
type ProfileGeneral struct {
	Name                   string `json:"Name"`
	Description            string `json:"Description,omitempty"`
	AssignmentType         string `json:"AssignmentType,omitempty"`         // "Auto", "Optional", etc.
	ProfileScope           string `json:"ProfileScope,omitempty"`           // "Production", "Test", etc.
	ManagedLocationGroupID string `json:"ManagedLocationGroupId,omitempty"` // Can be string or int
	IsActive               bool   `json:"IsActive,omitempty"`
}

// ProfileUpdateRequest represents a request to update a profile.
// The structure is flat - General and payload sections (Passcode, Restrictions, etc.) are at the same level.
type ProfileUpdateRequest map[string]interface{}

// ProfileDeleteResponse represents the response from deleting a profile.
type ProfileDeleteResponse struct {
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
}

// Platform constants.
const (
	PlatformAndroid       = "Android"
	PlatformAppleIOS      = "Apple iOS"
	PlatformWindows10     = "Windows 10"
	PlatformAppleOSX      = "AppleOsX"
	PlatformWindowsRugged = "Windows_Rugged"
	PlatformLinux         = "To do" // Linux is represented as "To do" in Workspace ONE
)

// ProfileType constants.
const (
	ProfileTypeDevice      = "Device"
	ProfileTypeUser        = "User"
	ProfileTypeCertificate = "Certificate"
	ProfileTypeApplication = "Application"
	ProfileTypeCompliance  = "Compliance"
)

// Status constants.
const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
	StatusDraft    = "Draft"
)

// AssignmentType constants.
const (
	AssignmentTypeAuto       = "Auto"
	AssignmentTypeOptional   = "Optional"
	AssignmentTypeCompliance = "Compliance"
)
