package tests

import (
	"errors"
	"fmt"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
)

// TestIsAndroidEMMPreconditionError exercises isAndroidEMMPreconditionError
// (profile_crud_test.go) against synthetic errors without any live tenant
// call, so the skip-vs-fail classification used by
// TestIntegration_ProfileService_FullCRUD's Create_Profile step can be
// verified in isolation. Every non-matching case is a control: each one MUST
// return false, proving a genuinely different failure (wrong status, wrong
// error code, or a non-API error) is never silently masked as the known
// Android-EMM-registration carve-out.
func TestIsAndroidEMMPreconditionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "documented precondition: 412 + errorCode 5163",
			err:  &client.APIError{StatusCode: 412, ErrorCode: "5163", Message: "Android EMM Registration should be completed prior to profile setup."},
			want: true,
		},
		{
			name: "control: right status, wrong errorCode",
			err:  &client.APIError{StatusCode: 412, ErrorCode: "5164", Message: "some other precondition failure"},
			want: false,
		},
		{
			name: "control: right errorCode, wrong status",
			err:  &client.APIError{StatusCode: 400, ErrorCode: "5163", Message: "unrelated bad request reusing the code by coincidence"},
			want: false,
		},
		{
			name: "control: unrelated API error entirely",
			err:  &client.APIError{StatusCode: 401, ErrorCode: "7000", Message: "Application not found or user does not have access to it."},
			want: false,
		},
		{
			name: "control: non-APIError error",
			err:  fmt.Errorf("network timeout dialing host"),
			want: false,
		},
		{
			name: "control: wrapped non-matching APIError still fails via errors.As chain",
			err:  fmt.Errorf("create profile: %w", &client.APIError{StatusCode: 500, ErrorCode: "1000", Message: "Internal Server Error"}),
			want: false,
		},
		{
			name: "wrapped matching APIError still classifies as skip-worthy",
			err:  fmt.Errorf("create profile: %w", &client.APIError{StatusCode: 412, ErrorCode: "5163", Message: "Android EMM Registration should be completed prior to profile setup."}),
			want: true,
		},
		{
			name: "control: nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAndroidEMMPreconditionError(tt.err)
			if got != tt.want {
				t.Errorf("isAndroidEMMPreconditionError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestIsAndroidEMMPreconditionError_ErrorsAsSanity is a control proving
// errors.As itself behaves as isAndroidEMMPreconditionError assumes: it
// MUST return true for a directly-typed *client.APIError, and MUST return
// false for an error of a different type, so a change to client.APIError's
// shape (or a stdlib errors.As behavior change) would show up here first.
func TestIsAndroidEMMPreconditionError_ErrorsAsSanity(t *testing.T) {
	var apiErr *client.APIError
	if !errors.As(&client.APIError{StatusCode: 412, ErrorCode: "5163"}, &apiErr) {
		t.Fatal("errors.As failed to match a direct *client.APIError")
	}
	if errors.As(fmt.Errorf("plain error"), &apiErr) {
		t.Fatal("errors.As incorrectly matched a plain, non-APIError error")
	}
}
