package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedSensorLifecycleV1 verifies the full CRUD lifecycle for
// device sensors using V1 generated endpoints.
// Fixtures: testdata/mock-responses/sensors/.
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
		// Platform, QueryType, TriggerType are string enums, not the ints old
		// swagger implied (verified against DeviceSensorRequestV1Model in
		// internal-source/mdmv1.json). All three are required, so the
		// generated Validate() rejects an empty-string zero value; use a real
		// member of each enum instead: "WIN_RT" (platform), "POWERSHELL"
		// (query_type), "EVENT" (trigger_type).
		Platform:              "WIN_RT",
		QueryType:             "POWERSHELL",
		TriggerType:           "EVENT",
		OrganizationGroupUUID: "804334cf-f067-4e0c-f8fa-1f95ac47b237",
		ScriptData:            "V3JpdGUtT3V0cHV0ICdIZWxsbyBXb3JsZCc=",
	}
	_, err := svc.CreateDeviceSensor(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateDeviceSensor failed: %v", err)
	}

	// 2. Get sensor — GET /api/mdm/devicesensors/{sensorUuid} → 200
	// Live-captured against the real test tenant:
	// create->get lifecycle against a genuine sensor, replacing a fabricated
	// fixture. Name/description are anonymizer-faked (seed 12345); UUID is
	// the fixture's own anonymized value, not the real tenant's sensor UUID.
	sensorUUID := "a447ee15-78a8-4992-cb7a-7ce115d8c83d"
	_, sensor, err := svc.GetDeviceSensor(ctx, sensorUUID)
	if err != nil {
		t.Fatalf("GetDeviceSensor failed: %v", err)
	}
	if sensor == nil {
		t.Fatal("expected non-nil sensor response")
	}
	var wantFixture struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	}
	loadFixtureResponseBodyForTest(t, "sensors/get_sensor.json", &wantFixture)
	if sensor.Name != wantFixture.Name {
		t.Errorf("sensor name: got %q, want %q", sensor.Name, wantFixture.Name)
	}
	if sensor.UUID != wantFixture.UUID {
		t.Errorf("sensor UUID: got %q, want %q", sensor.UUID, wantFixture.UUID)
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

	// 4. List sensors — GET /api/mdm/devicesensors/list/{orgGroupUuid} → 204
	// Live-captured 2026-08-05 against the real sdktest tenant, which has zero
	// device sensors (204 No Content) — replacing a fabricated fixture that
	// claimed 2. DoRequest leaves a zero-value response for an empty body, so
	// TotalResults is an unset (nil) pointer and ResultSet is empty, not populated-zero.
	orgGroupUUID := "804334cf-f067-4e0c-f8fa-1f95ac47b237"
	_, listResp, err := svc.GetDeviceSensors(ctx, orgGroupUUID, nil)
	if err != nil {
		t.Fatalf("GetDeviceSensors failed: %v", err)
	}
	if listResp == nil {
		t.Fatal("expected non-nil list response")
	}
	if listResp.TotalResults != nil {
		t.Errorf("total results: got %v, want nil (empty tenant, 204 response)", *listResp.TotalResults)
	}
	if len(listResp.ResultSet) != 0 {
		t.Errorf("result set length: got %d, want 0", len(listResp.ResultSet))
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

// TestGeneratedAssignDeviceSensorV1 verifies the V1 smart-group assignment
// endpoint using the generated DeviceSensorsV1Service.
// Fixture: testdata/mock-responses/sensors/assign_sensors.json.
func TestGeneratedAssignDeviceSensorV1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV1Service(c)
	ctx := context.Background()

	// Assign sensors to smart groups — POST /api/mdm/devicesensors/assign → 200
	assignReq := &sdk.DeviceSensorSmartGroupAssignmentV1Model{
		OrganizationGroupUUID: "804334cf-f067-4e0c-f8fa-1f95ac47b237",
		DeviceSensors: []string{
			"a447ee15-78a8-4992-cb7a-7ce115d8c83d",
			"e998053f-ed8a-dcb1-0978-e4771130e8f1",
		},
		SmartGroups: []string{"8df22927-ea01-3372-df01-a14006567365"},
	}
	_, resp, err := svc.AssignDeviceSensor(ctx, assignReq)
	if err != nil {
		t.Fatalf("AssignDeviceSensor failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil assignment response")
	}
	// Live-captured against the real test tenant:
	// assigned a genuine sensor to a real smart group, all-accepted (0
	// failures) — replacing a fabricated fixture that claimed a mixed
	// accepted/failed outcome. On this all-success shape the server omits
	// FailedSensors entirely rather than sending an empty array.
	if resp.TotalItems == nil || *resp.TotalItems != 1 {
		t.Errorf("total items: got %v, want 1", resp.TotalItems)
	}
	if resp.AcceptedItems == nil || *resp.AcceptedItems != 1 {
		t.Errorf("accepted items: got %v, want 1", resp.AcceptedItems)
	}
	if resp.FailedItems == nil || *resp.FailedItems != 0 {
		t.Errorf("failed items: got %v, want 0", resp.FailedItems)
	}
	if len(resp.FailedSensors) != 0 {
		t.Fatalf("failed sensors length: got %d, want 0", len(resp.FailedSensors))
	}
}
