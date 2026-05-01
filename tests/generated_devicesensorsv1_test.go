package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedSensorLifecycleV1 verifies the full CRUD lifecycle for
// device sensors using V1 generated endpoints.
// Fixtures: testdata/mock-responses/sensors/
func TestGeneratedSensorLifecycleV1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV1Service(c)
	ctx := context.Background()

	// 1. Create sensor — POST /api/mdm/devicesensors → 201 (no body)
	createReq := &sdk.DeviceSensorRequestV1Model{
		Name: "Test Sensor",
		// Platform, QueryType, TriggerType are string enums (e.g. "WIN_RT", "POWERSHELL", "EVENT").
		// Use empty string as zero value; the mock server does not validate enum values.
		Platform:              "",
		QueryType:             "",
		TriggerType:           "",
		OrganizationGroupUUID: "804334cf-f067-4e0c-f8fa-1f95ac47b237",
		ScriptData:            "V3JpdGUtT3V0cHV0ICdIZWxsbyBXb3JsZCc=",
	}
	_, err := svc.CreateDeviceSensor(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateDeviceSensor failed: %v", err)
	}

	// 2. Get sensor — GET /api/mdm/devicesensors/{sensorUuid} → 200
	sensorUUID := "a447ee15-78a8-4992-cb7a-7ce115d8c83d"
	_, sensor, err := svc.GetDeviceSensor(ctx, sensorUUID)
	if err != nil {
		t.Fatalf("GetDeviceSensor failed: %v", err)
	}
	if sensor == nil {
		t.Fatal("expected non-nil sensor response")
	}
	if sensor.Name != "Test Sensor Alpha" {
		t.Errorf("sensor name: got %q, want %q", sensor.Name, "Test Sensor Alpha")
	}
	if sensor.UUID != sensorUUID {
		t.Errorf("sensor UUID: got %q, want %q", sensor.UUID, sensorUUID)
	}
	// NOTE: EventTrigger will be nil because the V1 API returns string enums
	// but the generated model uses []*int (QUIRK-11). All other enum fields
	// (Platform, QueryType, etc.) are correctly string-typed via mdmv1.overlay.yaml.

	// 3. Update sensor — PUT /api/mdm/devicesensors/{sensorUuid} → 204 (no body)
	updateReq := &sdk.DeviceSensorUpdateV1Model{
		Description: "Updated description",
		ScriptData:  "V3JpdGUtT3V0cHV0ICdVcGRhdGVkJw==",
	}
	_, err = svc.UpdateDeviceSensor(ctx, sensorUUID, updateReq)
	if err != nil {
		t.Fatalf("UpdateDeviceSensor failed: %v", err)
	}

	// 4. List sensors — GET /api/mdm/devicesensors/list/{orgGroupUuid} → 200
	orgGroupUUID := "804334cf-f067-4e0c-f8fa-1f95ac47b237"
	_, listResp, err := svc.GetDeviceSensors(ctx, orgGroupUUID, nil)
	if err != nil {
		t.Fatalf("GetDeviceSensors failed: %v", err)
	}
	if listResp == nil {
		t.Fatal("expected non-nil list response")
	}
	if listResp.TotalResults == nil || *listResp.TotalResults != 2 {
		t.Errorf("total results: got %v, want 2", listResp.TotalResults)
	}
	if len(listResp.ResultSet) != 2 {
		t.Errorf("result set length: got %d, want 2", len(listResp.ResultSet))
	}

	// 5. Bulk delete — POST /api/mdm/devicesensors/bulkdelete → 204 (no body)
	deleteReq := &sdk.DeviceSensorsBulkDeleteRequestV1Model{
		OrganizationGroupUUID: orgGroupUUID,
		SensorUUIDs:           []string{sensorUUID},
	}
	_, err = svc.BulkDeleteDeviceSensors(ctx, deleteReq)
	if err != nil {
		t.Fatalf("BulkDeleteDeviceSensors failed: %v", err)
	}
}
