package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedGetDeviceSensorsByDeviceV1 verifies GET /api/mdm/devices/{deviceUuid}/sensors
// using the generated DeviceSensorsService (mdmv1 — device-scoped sensor list).
//
// The fixture (testdata/mock-responses/sensors/get_device_sensors.json) was
// flagged SUSPICIOUS by the 2026-08-05 ledger audit (hand-typed sequential-hex
// placeholder uuids, suspiciously round timestamps) -- no live-tenant capture
// exists for this endpoint. It is loaded here via the bypass mechanism
// (newMockServerWithBypassFixtures, see tests/bypass_fixture_helper_test.go)
// from its Phase-3-surviving synthetic-placeholder location
// (internal/mockserver/testdata/synthetic-placeholders/sensors/get_device_sensors.json),
// and this test's assertions were weakened 2026-08-06 to check shape
// (non-empty results, UUID-shaped uuid fields, non-empty name/value) rather
// than the specific fabricated values the old fixture claimed.
func TestGeneratedGetDeviceSensorsByDeviceV1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := newMockServerWithBypassFixtures(t, "../testdata/mock-responses", []string{
		"../internal/mockserver/testdata/synthetic-placeholders/sensors/get_device_sensors.json",
	})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsService(c)
	ctx := context.Background()

	deviceUUID := "a447ee15-78a8-4992-cb7a-7ce115d8c83d"

	_, resp, err := svc.GetDeviceSensorsByDeviceUuidAsync(ctx, deviceUUID, nil)
	if err != nil {
		t.Fatalf("GetDeviceSensorsByDeviceUuidAsync failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.TotalResults != len(resp.Results) {
		t.Errorf("total_results: got %v, want it to match len(results)=%d", resp.TotalResults, len(resp.Results))
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected at least one sensor result")
	}
	for i, r := range resp.Results {
		if !looksLikeUUID(r.UUID) {
			t.Errorf("results[%d].UUID is not UUID-shaped: %q", i, r.UUID)
		}
		if r.Name == "" {
			t.Errorf("results[%d].Name is empty", i)
		}
		if r.Value == "" {
			t.Errorf("results[%d].Value is empty", i)
		}
	}
}
