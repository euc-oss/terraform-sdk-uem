package client

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"typed NotFoundError", &NotFoundError{Resource: "profiles", ID: "7"}, true},
		{"wrapped NotFoundError", fmt.Errorf("ctx: %w", &NotFoundError{Resource: "profiles", ID: "7"}), true},
		{"plain 404", &APIError{StatusCode: 404, Message: "Not Found"}, true},
		{"profile 400 Invalid Profile", &APIError{StatusCode: 400, ErrorCode: "400", Message: "Invalid Profile 68910."}, true},
		{"smart group 400 errorCode 6", &APIError{StatusCode: 400, ErrorCode: "6", Message: "Smart Group not found with Id: 12345 or User does not have access to it."}, true},
		{"internal app 401 errorCode 7000", &APIError{StatusCode: 401, ErrorCode: "7000", Message: "Application not found or user does not have access to it."}, true},
		{"profile DELETE 400", &APIError{StatusCode: 400, ErrorCode: "400", Message: "Profile not found or User does not have access to the Profile."}, true},
		{"smart group DELETE 400 errorCode 400", &APIError{StatusCode: 400, ErrorCode: "400", Message: "Smart Group not found with Id: 467814 or User does not have access to it."}, true},
		{"internal app DELETE 400", &APIError{StatusCode: 400, ErrorCode: "400", Message: "Application not found or user does not have access to it."}, true},
		{"wrapped internal app 401/7000", fmt.Errorf("get: %w", &APIError{StatusCode: 401, ErrorCode: "7000"}), true},

		// Negative controls.
		{"plain 401 (auth failure) is NOT not-found", &APIError{StatusCode: 401, Message: "Unauthorized"}, false},
		{"401 with a different errorCode is NOT not-found", &APIError{StatusCode: 401, ErrorCode: "1005", Message: "User credentials are invalid."}, false},
		{"401 errorCode 70000 is not 7000", &APIError{StatusCode: 401, ErrorCode: "70000"}, false},
		{"generic 400 validation error", &APIError{StatusCode: 400, ErrorCode: "400", Message: "Name is required."}, false},
		{"400 errorCode 6 without the smart-group message", &APIError{StatusCode: 400, ErrorCode: "6", Message: "Something else."}, false},
		{"smart-group message with an unrelated errorCode", &APIError{StatusCode: 400, ErrorCode: "1001", Message: "Smart Group not found with Id: 1"}, false},
		{"unrelated 400 mentioning an application", &APIError{StatusCode: 400, ErrorCode: "400", Message: "Application name is invalid."}, false},
		{"not-found message on a 500 is NOT not-found", &APIError{StatusCode: 500, ErrorCode: "400", Message: "Profile not found or User does not have access to the Profile."}, false},
		{"500", &APIError{StatusCode: 500, Message: "Internal Server Error"}, false},
		{"non-API error", errors.New("dial tcp: connection refused"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
