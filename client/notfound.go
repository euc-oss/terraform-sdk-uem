package client

import (
	"errors"
	"fmt"
	"strings"
)

// NotFoundError reports that a resource does not exist. It is returned
// directly by SDK code that can determine absence without a server
// response (e.g. a registry miss after discovery), and IsNotFound also
// recognizes the server's own not-found signatures on an *APIError.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
}

// IsNotFound reports whether err means "this resource does not exist".
//
// UEM has no uniform not-found response, and the same resource can answer
// a GET and a repeat DELETE differently. Each signature below was captured
// live on 26.2 (2026-09-24) against an object deleted moments earlier:
//   - HTTP 404 (generic).
//   - Profiles, GET: HTTP 400, message "Invalid Profile <id>.".
//   - Profiles, DELETE: HTTP 400, message "Profile not found or User does
//     not have access to the Profile.".
//   - Smart groups, GET: HTTP 400, errorCode 6, message "Smart Group not
//     found with Id: <id> or User does not have access to it.".
//   - Smart groups, DELETE: HTTP 400, errorCode 400, the same message.
//   - Internal apps (incl. macOS), GET: HTTP 401 with errorCode exactly
//     7000. A 401 WITHOUT errorCode 7000 is an authentication failure and
//     is deliberately NOT treated as not-found.
//   - Internal apps, DELETE: HTTP 400, message "Application not found or
//     user does not have access to it.".
//
// Where errorCode is only the generic 400, the message prefix is the one
// distinguishing field, so it is matched as a message prefix. The server's own wording
// folds "no access" into "not found"; callers get the server's verdict.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return true
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.StatusCode {
	case 404:
		return true
	case 401:
		return apiErr.ErrorCode == "7000"
	case 400:
		msg := apiErr.Message
		switch {
		case strings.HasPrefix(msg, "Invalid Profile "),
			strings.HasPrefix(msg, "Profile not found or User does not have access to the Profile"),
			strings.HasPrefix(msg, "Application not found or user does not have access to it"):
			return true
		case strings.HasPrefix(msg, "Smart Group not found with Id:"):
			return apiErr.ErrorCode == "6" || apiErr.ErrorCode == "400"
		}
	}
	return false
}

// UnsupportedPlatformError reports that a resource's platform string (as
// reported by a search endpoint) has no proven mapping to the platform key
// the SDK's detail/update/delete calls switch on. RawPlatform is the exact
// string the server returned; an empty RawPlatform is reported distinctly,
// since it indicates a row from a code path that never sets the platform
// (e.g. canonically, Per-App VPN rows under profiletype=perappvpnprofile).
type UnsupportedPlatformError struct {
	Resource    string
	ID          string
	RawPlatform string
}

func (e *UnsupportedPlatformError) Error() string {
	if e.RawPlatform == "" {
		return fmt.Sprintf("%s %s: the search endpoint reported an EMPTY platform, which has no mapping to a detail-GET platform", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s %s: unsupported platform %q (no proven mapping from the search endpoint's platform string to a detail-GET platform)", e.Resource, e.ID, e.RawPlatform)
}

// UnsupportedOperationError reports that an operation does not exist on the
// server for a given platform, so the SDK refuses it before sending any
// request (for example, there is no profile create endpoint for Windows
// Rugged). Reason says why, with the doctrine reference.
type UnsupportedOperationError struct {
	Resource  string
	Operation string
	Platform  string
	Reason    string
}

func (e *UnsupportedOperationError) Error() string {
	return fmt.Sprintf("%s %s is not supported for platform %q: %s", e.Resource, e.Operation, e.Platform, e.Reason)
}
