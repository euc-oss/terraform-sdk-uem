package client

import (
	"testing"
)

func TestParseLocationID(t *testing.T) {
	tests := []struct {
		name      string
		location  string
		wantID    string
		wantError bool
	}{
		{
			name:     "normal path with leading slash",
			location: "/api/mdm/devicesensors/uuid-here",
			wantID:   "uuid-here",
		},
		{
			name:     "normal path without leading slash",
			location: "api/mam/apps/internal/12345",
			wantID:   "12345",
		},
		{
			name:     "trailing slash",
			location: "/api/resource/42/",
			wantID:   "42",
		},
		{
			name:      "empty string",
			location:  "",
			wantError: true,
		},
		{
			name:     "single segment",
			location: "12345",
			wantID:   "12345",
		},
		{
			name:      "just slashes",
			location:  "///",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLocationID(tt.location)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseLocationID(%q) expected error, got nil (id=%q)", tt.location, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseLocationID(%q) unexpected error: %v", tt.location, err)
				return
			}
			if got != tt.wantID {
				t.Errorf("ParseLocationID(%q) = %q, want %q", tt.location, got, tt.wantID)
			}
		})
	}
}
