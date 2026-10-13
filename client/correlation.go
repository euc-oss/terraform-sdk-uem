package client

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
)

// CorrelationIDHeader is the header UEM uses to correlate a request with its
// server-side log entries. The server reuses the request's value when it is a
// valid GUID and generates one otherwise, and it sets the header on every
// response.
//
// Correlation rule: the SDK sends one GUID per logical call (one DoRequest) and
// reuses it for every retry of that call, so all attempts correlate in the
// server's logs. The value the server echoes in its response is authoritative
// for APIError.CorrelationID and the debug line; when the response has no such
// header (a gateway error page) or there is no response at all (timeout, reset),
// the id the SDK sent is used. The sent id is kept too (APIError.SentCorrelationID)
// so the two can be compared when a gateway answers with a different one.
//
// Every error DoRequest returns carries the call's id, with debug logging off: an
// *APIError has it in its CorrelationID field, every other error (timeouts, context
// deadline or cancellation, connection, TLS and DNS failures, retries exhausted
// without a response, token request failures, client-side validation) is wrapped in
// a *CorrelatedError. The text of both ends in "(correlation id: <guid>)", and
// CorrelationID(err) extracts the id from any of them, through further wrapping.
const CorrelationIDHeader = "X-Correlation-ID"

// maxCorrelationIDLen is the longest correlation id taken from a response header.
const maxCorrelationIDLen = 128

// newCorrelationID returns a random (version 4) GUID in canonical lowercase form.
func newCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail on supported platforms; without an id the
		// request simply goes out without one and the server generates its own.
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// cleanCorrelationID returns a response header value that is safe to store and
// print, or "" when it is not one: after trimming surrounding white space the value
// must be 1 to maxCorrelationIDLen characters of printable ASCII without spaces.
// A value that fails is treated as absent rather than repaired, because a repaired
// id would match nothing in the server's logs; the caller then reports the id the
// SDK sent.
func cleanCorrelationID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > maxCorrelationIDLen {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return ""
		}
	}
	return s
}

// resolveCorrelationID picks the id to report for a call: the server's echoed
// value when present, otherwise the id the SDK sent.
func resolveCorrelationID(respHeader string, sent string) string {
	if id := cleanCorrelationID(respHeader); id != "" {
		return id
	}
	return sent
}

// CorrelatedError is an error returned by a call, together with the call's
// X-Correlation-ID. Error() is the wrapped error's text followed by
// " (correlation id: <id>)". It unwraps to the original error, so errors.Is and
// errors.As keep working (for example errors.Is(err, context.DeadlineExceeded) for
// a timeout, or errors.As for *url.Error).
//
// An API error response is an *APIError instead, which carries the id in its
// CorrelationID field; DoRequest never wraps an *APIError in a CorrelatedError.
// For a call that fails before a request is sent (client-side validation, the
// rate limiter, building the request) the id is the one the call would have sent;
// the server has no record of it.
type CorrelatedError struct {
	ID  string
	Err error
}

// Error returns the wrapped error's text with the correlation id appended.
func (e *CorrelatedError) Error() string {
	return e.Err.Error() + " (correlation id: " + e.ID + ")"
}

// Unwrap returns the original error.
func (e *CorrelatedError) Unwrap() error { return e.Err }

// CorrelationID returns the correlation id of the call that produced err: the
// CorrelationID of an *APIError, or the ID of a *CorrelatedError, found anywhere
// in err's chain. It returns "" for an error that carries none.
func CorrelationID(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.CorrelationID != "" {
		return apiErr.CorrelationID
	}
	var ce *CorrelatedError
	if errors.As(err, &ce) {
		return ce.ID
	}
	return ""
}

// withCorrelation attaches id to err unless err already carries one.
func withCorrelation(err error, id string) error {
	if err == nil || id == "" || CorrelationID(err) != "" {
		return err
	}
	return &CorrelatedError{ID: id, Err: err}
}
