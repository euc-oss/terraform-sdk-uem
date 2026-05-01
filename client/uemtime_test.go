package client

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestUEMTime_UnmarshalJSON_AcceptsAllUEMFormats(t *testing.T) {
	cases := []struct {
		name string
		json string
		want time.Time
	}{
		{"RFC3339 with Z", `"2026-04-27T16:00:00Z"`, time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)},
		{"RFC3339 nano with Z", `"2026-04-27T16:00:00.123456789Z"`, time.Date(2026, 4, 27, 16, 0, 0, 123456789, time.UTC)},
		{"RFC3339 with offset", `"2026-04-27T16:00:00-07:00"`, time.Date(2026, 4, 27, 16, 0, 0, 0, time.FixedZone("", -7*3600))},
		{"millis no TZ (bug repro)", `"2026-04-27T16:00:00.000"`, time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)},
		{"seconds no TZ", `"2026-04-27T16:00:00"`, time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)},
		{"date only", `"2026-04-27"`, time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)},
		{"space separator", `"2026-04-27 16:00:00"`, time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)},
		{"US slashes", `"04/27/2026"`, time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)},
		{"empty string", `""`, time.Time{}},
		{"null", `null`, time.Time{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got UEMTime
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("Unmarshal(%s): got %v want %v", tc.json, got.Time, tc.want)
			}
		})
	}
}

func TestUEMTime_UnmarshalJSON_RejectsGarbage(t *testing.T) {
	var got UEMTime
	if err := json.Unmarshal([]byte(`"not a date"`), &got); err == nil {
		t.Fatal("expected error for unparseable string")
	}
}

func TestUEMTime_UnmarshalJSON_AcceptsEpochNumber(t *testing.T) {
	var got UEMTime
	if err := json.Unmarshal([]byte(`1700000000`), &got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Unix(1700000000, 0).UTC()
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got.Time, want)
	}
}

// TestUEMTime_UnmarshalJSON_RejectsNonIntegerEpoch pins strict integer parsing:
// fmt.Sscanf("%d") would silently accept "1e3" or "1700.5" and truncate to a
// wrong value; strconv.ParseInt rejects them outright.
func TestUEMTime_UnmarshalJSON_RejectsNonIntegerEpoch(t *testing.T) {
	for _, raw := range []string{`1e3`, `1700.5`, `0x10`} {
		var got UEMTime
		if err := json.Unmarshal([]byte(raw), &got); err == nil {
			t.Errorf("expected error for non-integer epoch %q, got time=%v", raw, got.Time)
		}
	}
}

func TestUEMTime_MarshalJSON_RoundTrip(t *testing.T) {
	original := UEMTime{Time: time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"2026-04-27T16:00:00.000Z"` {
		t.Errorf("Marshal: got %s", data)
	}

	var roundTrip UEMTime
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if !roundTrip.Equal(original.Time) {
		t.Errorf("round-trip mismatch: got %v want %v", roundTrip.Time, original.Time)
	}
}

func TestUEMTime_MarshalJSON_ZeroIsNull(t *testing.T) {
	data, err := json.Marshal(UEMTime{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("zero UEMTime: got %s want null", data)
	}
}

func TestUEMTime_String_CanonicalFormat(t *testing.T) {
	v := UEMTime{Time: time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)}
	if v.String() != "2026-04-27T16:00:00.000Z" {
		t.Errorf("String: got %q", v.String())
	}
	if (UEMTime{}).String() != "" {
		t.Errorf("zero String: got %q want empty", (UEMTime{}).String())
	}
}

func TestParseUEMTime_AllLayouts(t *testing.T) {
	if _, err := ParseUEMTime("2026-04-27T16:00:00.000"); err != nil {
		t.Fatalf("ParseUEMTime: %v", err)
	}
	if got, _ := ParseUEMTime(""); !got.IsZero() {
		t.Errorf("empty input should yield zero")
	}
	if _, err := ParseUEMTime("garbage"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUEMTime_TypoSuggestions(t *testing.T) {
	cases := []struct {
		input        string
		wantContains string
	}{
		{"2026-04-27T16-00-00", `"2026-04-27T16:00:00"`},     // dashes in time
		{"2026-04-27T16:00", "T16:00:00"},                    // missing seconds
		{"27/04/2026", "MM/DD/YYYY"},                         // DD/MM ordering
		{"2026-04-27 16:00:00 UTC", `"2026-04-27 16:00:00"`}, // trailing UTC literal
		{"20260427", `"2026-04-27"`},                         // compact date
		{"hello world", "expected formats"},                  // generic fallback
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			_, err := ParseUEMTime(tc.input)
			if err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantContains)
			}
		})
	}
}

// TestUEMTime_RealWorldUEMResponse verifies the bug repro from the original
// report: InternalAppsV1_GetInternalAppByIdAsync returned RenewalDate as
// "2026-04-27T16:00:00.000" and the default time.Time unmarshaler failed
// with: parsing time ... cannot parse "" as "Z07:00".
func TestUEMTime_RealWorldUEMResponse(t *testing.T) {
	body := []byte(`{"RenewalDate":"2026-04-27T16:00:00.000"}`)
	var payload struct {
		RenewalDate UEMTime `json:"RenewalDate"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("real-world payload should unmarshal cleanly, got: %v", err)
	}
	if payload.RenewalDate.IsZero() {
		t.Fatal("expected non-zero parsed time")
	}
}
