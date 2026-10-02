package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestGeneratedSensorsV2List verifies GET /api/mdm/devicesensors/list/{organizationGroupUuid}
// (V2, GetDeviceSensorsAsync) against the genuine empty-list response captured live
// (UEM 26.2.0.0 -- org group with zero sensors configured).
// Fixture: testdata/mock-responses/sensors/list_sensors_v2_empty.json.
func TestGeneratedSensorsV2List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV2Service(c)
	ctx := context.Background()

	_, list, err := svc.GetDeviceSensorsAsync(ctx, "804334cf-f067-4e0c-f8fa-1f95ac47b237", nil)
	if err != nil {
		t.Fatalf("GetDeviceSensorsAsync failed: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil list response")
	}
	if len(list.ResultSet) != 0 {
		t.Errorf("ResultSet: got %d entries, want 0 (empty-list fixture)", len(list.ResultSet))
	}
	if list.TotalResults == nil || *list.TotalResults != 0 {
		t.Errorf("TotalResults: got %v, want 0", list.TotalResults)
	}
}
