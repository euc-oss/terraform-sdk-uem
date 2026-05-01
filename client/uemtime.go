package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// uemTimeFormats lists every datetime layout we have seen the Workspace ONE
// UEM API return or accept. Order matters — most-specific first so a string
// with fractional seconds and a timezone is not truncated to a date.
var uemTimeFormats = []string{
	time.RFC3339Nano,                      // 2006-01-02T15:04:05.999999999Z07:00
	time.RFC3339,                          // 2006-01-02T15:04:05Z07:00
	"2006-01-02T15:04:05.999999999Z07:00", // explicit nano with TZ
	"2006-01-02T15:04:05.000Z07:00",       // millis with TZ
	"2006-01-02T15:04:05.999999999",       // nano, no TZ — UEM common
	"2006-01-02T15:04:05.000",             // millis, no TZ — UEM common (this is the bug repro format)
	"2006-01-02T15:04:05",                 // seconds, no TZ
	"2006-01-02 15:04:05.999999999",       // space separator, nano
	"2006-01-02 15:04:05",                 // space separator
	"2006-01-02",                          // date only
	"01/02/2006 15:04:05",                 // US-style with time
	"01/02/2006",                          // US-style date only
}

// canonicalUEMTimeFormat is what we emit when serializing back to JSON or a
// query string. RFC3339 with millisecond precision matches UEM's preferred
// inbound format while keeping an explicit UTC marker so any consumer can
// parse it unambiguously.
const canonicalUEMTimeFormat = "2006-01-02T15:04:05.000Z07:00"

// UEMTime is a [time.Time] wrapper that tolerates the various datetime
// formats the Workspace ONE UEM API returns — including timestamps with no
// timezone like "2026-04-27T16:00:00.000" — and emits a single canonical
// format on the wire.
//
// Use UEMTime anywhere a model field maps to a `format: date-time` schema.
// The zero value is a zero time and serializes as JSON null.
type UEMTime struct {
	time.Time
}

// NewUEMTime wraps a [time.Time] value as a UEMTime.
func NewUEMTime(t time.Time) UEMTime {
	return UEMTime{Time: t}
}

// ParseUEMTime parses s using every known UEM datetime layout and returns
// the first successful result. Empty input parses to a zero UEMTime.
//
// On failure the error includes a hint when s looks like a near-miss for one
// of the known layouts (e.g. "2026-04-27T16:00" → suggest seconds, or
// "2026-04-27T16-00-00" → suggest colon separators).
func ParseUEMTime(s string) (UEMTime, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return UEMTime{}, nil
	}
	for _, layout := range uemTimeFormats {
		if parsed, err := time.Parse(layout, s); err == nil {
			return UEMTime{Time: parsed}, nil
		}
	}
	return UEMTime{}, fmt.Errorf("UEMTime: cannot parse %q: %s", s, suggestFix(s))
}

// suggestFix inspects s and returns a short, user-actionable message when the
// input looks like a near-miss for a real datetime. It is best-effort and
// always returns a non-empty string ("expected a datetime like 2006-01-02..."
// when no specific suggestion fires) so callers can append it directly to an
// error message.
func suggestFix(s string) string {
	// Common corrupt forms first — these are catchable typos.
	switch {
	case strings.Contains(s, "T") && strings.Count(s, ":") == 0 && strings.Count(s, "-") >= 4:
		// e.g. "2026-04-27T16-00-00" — used dashes instead of colons in the time part
		fixed := repairTimeSeparators(s)
		return fmt.Sprintf("did you mean %q? (time portion uses ':' between hours/minutes/seconds)", fixed)
	case strings.Contains(s, "/") && !strings.ContainsAny(s, "T :"):
		// "27/04/2026" — looks DD/MM/YYYY, suggest the supported MM/DD/YYYY
		return "MM/DD/YYYY is supported (e.g. \"04/27/2026\"); DD/MM/YYYY is not"
	case strings.Contains(s, "T") && strings.Count(s, ":") == 1:
		// "2026-04-27T16:00" — missing seconds
		return fmt.Sprintf("did you mean %q? (seconds are required, e.g. T16:00:00)", s+":00")
	case strings.Contains(s, " ") && strings.HasSuffix(strings.ToUpper(s), " UTC"):
		// "2026-04-27 16:00:00 UTC" — Go time layout doesn't accept "UTC" literal.
		// Trim by length so any case (UTC, utc, Utc, …) is stripped cleanly.
		return fmt.Sprintf("strip the trailing \"UTC\" — try %q (or use \"Z\")", s[:len(s)-4])
	case strings.Contains(s, "Z") && strings.HasSuffix(s, "Z") && strings.Contains(s, "+"):
		// "2026-04-27T16:00:00+00:00Z" — both offset and Z
		return "remove either the offset or the trailing 'Z'; use one or the other"
	case onlyDigits(s) && len(s) == 8:
		// "20260427" — compact date, suggest hyphens
		return fmt.Sprintf("did you mean %q? (insert hyphens: YYYY-MM-DD)", s[:4]+"-"+s[4:6]+"-"+s[6:])
	}
	return "expected formats include 2006-01-02T15:04:05Z, 2006-01-02T15:04:05.000, or 2006-01-02"
}

func repairTimeSeparators(s string) string {
	tIdx := strings.Index(s, "T")
	if tIdx < 0 || tIdx == len(s)-1 {
		return s
	}
	return s[:tIdx+1] + strings.ReplaceAll(s[tIdx+1:], "-", ":")
}

func onlyDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// UnmarshalJSON implements [json.Unmarshaler]. It accepts:
//   - JSON null or empty string → zero time
//   - any string layout listed in uemTimeFormats
//   - a JSON number interpreted as Unix seconds (covers a few legacy endpoints)
func (t *UEMTime) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		t.Time = time.Time{}
		return nil
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		t.Time = time.Time{}
		return nil
	}

	// Numeric epoch fallback (rare, but some UEM diagnostics endpoints use it).
	// Use strconv.ParseInt for strict integer parsing — fmt.Sscanf("%d") would
	// silently accept "1e3" or "1700.5" and truncate.
	if raw[0] != '"' {
		secs, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("UEMTime: expected JSON string or integer epoch, got %q", raw)
		}
		t.Time = time.Unix(secs, 0).UTC()
		return nil
	}

	// JSON-decode the string so any escapes are honored before parsing the layout.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("UEMTime: %w", err)
	}
	parsed, err := ParseUEMTime(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// MarshalJSON implements [json.Marshaler]. A zero UEMTime marshals to null
// so it can be omitted with `omitempty` semantics on struct tags.
func (t UEMTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + t.UTC().Format(canonicalUEMTimeFormat) + `"`), nil
}

// String returns the canonical UEM datetime representation. The zero value
// returns an empty string so query-parameter formatters can omit it cleanly.
// This overrides the embedded [time.Time.String] method whose default layout
// is not parseable by the UEM API.
func (t UEMTime) String() string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(canonicalUEMTimeFormat)
}
