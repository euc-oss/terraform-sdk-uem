package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// The two tests below exercise genuine live captures of the device sensors
// list endpoint against a persistent hand-populated organization group
// holding 2 real macOS/BASH sensors -- the pre-existing sensors/list_sensors*
// fixtures are genuine empty-list captures (0 sensors at capture time) that
// never decode a real sensor item, so neither language's suite has ever
// round-tripped a populated DeviceSensorResponseV1Model or
// DeviceSensorResponseLiteV2Model. Each shares its endpoint/method/version
// with its empty sibling, so each is loaded via its own isolated
// single-fixture mock server rather than the full directory scan -- same
// collision-avoidance pattern used throughout this branch (see
// TestGeneratedScriptsV1GetLiveCapture).

// TestGeneratedSensorsV1ListPopulatedLiveCapture also proves
// DeviceSensorResponseV1Model.QueryType's real wire shape: this V1 sensor
// (macOS/BASH) returns query_type as a JSON NUMBER, decoded via
// client.IntOrString.
func TestGeneratedSensorsV1ListPopulatedLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/sensors/list_sensors_populated.json"))
	if err != nil {
		t.Fatalf("load list_sensors_populated fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV1Service(c)
	ctx := context.Background()

	_, list, err := svc.GetDeviceSensors(ctx, "any-og-uuid", nil)
	if err != nil {
		t.Fatalf("GetDeviceSensors failed: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil list response")
	}
	if len(list.ResultSet) != 2 {
		t.Fatalf("ResultSet: got %d entries, want 2 (populated fixture)", len(list.ResultSet))
	}
	if list.TotalResults == nil || *list.TotalResults != 2 {
		t.Errorf("TotalResults: got %v, want 2", list.TotalResults)
	}
	for i, item := range list.ResultSet {
		if item.QueryType == nil {
			t.Fatalf("ResultSet[%d].QueryType: got nil, want a decoded IntOrString", i)
		}
		if item.QueryType.IsStr {
			t.Errorf("ResultSet[%d].QueryType: got string %q, want a JSON-number value (this fixture's real wire shape)", i, item.QueryType.StrVal)
		}
		if item.QueryType.IntVal == 0 {
			t.Errorf("ResultSet[%d].QueryType.IntVal: got 0, want the real captured non-zero value", i)
		}
	}
}

// TestGeneratedSensorsV2ListPopulatedLiveCapture confirms the V2 response's
// query_type is genuinely a plain string on the wire for the same sensors
// (no flex-enum needed on the V2 side).
func TestGeneratedSensorsV2ListPopulatedLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/sensors/list_sensors_v2_populated.json"))
	if err != nil {
		t.Fatalf("load list_sensors_v2_populated fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV2Service(c)
	ctx := context.Background()

	_, list, err := svc.GetDeviceSensorsAsync(ctx, "any-og-uuid", nil)
	if err != nil {
		t.Fatalf("GetDeviceSensorsAsync failed: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil list response")
	}
	if len(list.ResultSet) != 2 {
		t.Fatalf("ResultSet: got %d entries, want 2 (populated fixture)", len(list.ResultSet))
	}
	for i, item := range list.ResultSet {
		if item.QueryType != "BASH" {
			t.Errorf("ResultSet[%d].QueryType: got %q, want \"BASH\" (real wire value, plain string)", i, item.QueryType)
		}
	}
}
