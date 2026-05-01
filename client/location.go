package client

import (
	"fmt"
	"strings"
)

// ParseLocationID extracts the last non-empty path segment from an HTTP
// Location header value. The UEM API uses this pattern for returning
// newly created resource IDs and UUIDs.
//
// Examples:
//
//	"api/mam/apps/internal/12345"                                  → "12345", nil
//	"/api/mdm/devicesensors/a1b2c3d4-e5f6-7890-abcd-ef1234567890" → "a1b2c3d4-...", nil
//	"/some/path/"                                                   → last non-empty segment
//	""                                                              → "", error
func ParseLocationID(location string) (string, error) {
	trimmed := strings.Trim(location, "/")
	if trimmed == "" {
		return "", fmt.Errorf("location header is empty")
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1], nil
}
