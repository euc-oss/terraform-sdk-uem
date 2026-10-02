package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestGeneratedSensorsV2Lifecycle verifies the V2 sensor create/get/update
// lifecycle (ops 1-3 of the sensors V2 verification batch), live-captured
// against UEM 26.2.0.0.
//
// Real-wire findings baked into these fixtures:
//   - GET always returns "timeout": 0 regardless of what was sent on
//     create/update -- not an anonymization artifact (verified against
//     tools/mock-capture/anonymize.go, which does not touch this field).
//   - (Retracted) an earlier working note claimed the create response's
//     "uuid" does not identify the created resource. That was a methodology
//     error: it compared the fixture's own anonymized uuid (anonymize.go
//     unconditionally fakes any "*uuid"-suffixed key) against a real value,
//     which can never match. A follow-up raw, non-fixture create+list check
//     confirmed the real create-response uuid is in fact reliable.
//
// Fixtures: testdata/mock-responses/sensors/{create,get,update}_sensor_v2.json.
func TestGeneratedSensorsV2Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV2Service(c)
	ctx := context.Background()

	// 1. Create sensor — POST /api/mdm/devicesensors → 201
	createReq := &sdk.DeviceSensorRequestV2Model{
		Name:                  "sdk_test_sensor_v2_lifecycle",
		OrganizationGroupUUID: "804334cf-f067-4e0c-f8fa-1f95ac47b237",
		Platform:              "WIN_RT",
		QueryType:             "POWERSHELL",
		QueryResponseType:     "STRING",
		ExecutionContext:      "SYSTEM",
		ExecutionArchitecture: "EITHER64OR32BIT",
		ScriptData:            "V3JpdGUtT3V0cHV0ICJoZWxsbyI=",
	}
	_, created, err := svc.CreateDeviceSensorAsync(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateDeviceSensorAsync failed: %v", err)
	}
	if created == nil {
		t.Fatal("expected non-nil create response")
	}

	// 2. Get sensor — GET /api/mdm/devicesensors/{sensorUuid} → 200
	sensorUUID := "0e44d914-4225-2276-b7cc-5cecb1ab3c4b"
	_, sensor, err := svc.GetDeviceSensorAsync(ctx, sensorUUID)
	if err != nil {
		t.Fatalf("GetDeviceSensorAsync failed: %v", err)
	}
	if sensor == nil {
		t.Fatal("expected non-nil sensor response")
	}
	if sensor.Platform != "WIN_RT" {
		t.Errorf("sensor platform: got %q, want %q", sensor.Platform, "WIN_RT")
	}

	// 3. Update sensor — PUT /api/mdm/devicesensors/{sensorUuid} → 204
	updateReq := &sdk.DeviceSensorUpdateV2Model{
		Description:           "Live capture test sensor, safe to delete",
		Platform:              "WIN_RT",
		QueryType:             "POWERSHELL",
		ExecutionContext:      "SYSTEM",
		ExecutionArchitecture: "EITHER64OR32BIT",
		ScriptData:            "V3JpdGUtT3V0cHV0ICJoZWxsbyI=",
		Timeout:               intPtr(180),
	}
	_, err = svc.UpdateDeviceSensorAsync(ctx, sensorUUID, updateReq)
	if err != nil {
		t.Fatalf("UpdateDeviceSensorAsync failed: %v", err)
	}
}

// TestGeneratedSensorsV2AssignmentLifecycle verifies the V2 assignment
// create/list/rank/get/update/delete lifecycle (ops 4-9 of the sensors V2
// verification batch), live-captured against UEM 26.2.0.0.
//
// Real-wire findings baked into these fixtures:
//   - "smart_group_uuids" is not marked required in swagger's
//     DeviceSensorAssignmentRequestV1Model but IS required on the wire (422
//     if omitted) -- an existing built-in smart group was reused.
//   - (Retracted) the create-assignment fixture's "uuid" appears identical to
//     the sensor-create fixture's "uuid"; that is a coincidence of the
//     anonymizer's deterministic PRNG reseed for same-shape {"uuid": ...}
//     bodies, not evidence of a real defect (see CreateDeviceSensorAsync
//     entry in tests/verified-endpoints.json for the full retraction).
//
// Fixtures: testdata/mock-responses/sensors/assignment_*_v2.json.
func TestGeneratedSensorsV2AssignmentLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewDeviceSensorsV2Service(c)
	ctx := context.Background()

	sensorUUID := "0e44d914-4225-2276-b7cc-5cecb1ab3c4b"
	assignmentUUID := "8fe98ed4-4ad2-ac03-26a2-60eb985e3f18"

	// 4. Create assignment — POST /api/mdm/devicesensors/{sensorUuid}/assignment → 201
	createReq := &sdk.DeviceSensorAssignmentRequestV1ModelV2{
		Name:            "Batch3 Test Assignment",
		TriggerType:     "SCHEDULE",
		SmartGroupUUIDs: []string{"bd3a7003-ee58-663f-6f18-1183b03a8627"},
	}
	_, created, err := svc.AddDeviceSensorAssignmentAsync(ctx, sensorUUID, createReq)
	if err != nil {
		t.Fatalf("AddDeviceSensorAssignmentAsync failed: %v", err)
	}
	if created == nil {
		t.Fatal("expected non-nil create-assignment response")
	}

	// 5. List assignments — GET /api/mdm/devicesensors/{sensorUuid}/assignments → 200
	_, list, err := svc.GetDeviceSensorAssignmentsAsync(ctx, sensorUUID)
	if err != nil {
		t.Fatalf("GetDeviceSensorAssignmentsAsync failed: %v", err)
	}
	if list == nil || len(*list) != 1 {
		t.Fatalf("expected exactly 1 assignment, got %v", list)
	}
	if (*list)[0].Ranking == nil || *(*list)[0].Ranking != 1 {
		t.Errorf("ranking: got %v, want 1", (*list)[0].Ranking)
	}

	// 6. Bulk update rankings — POST .../assignments?action=update-ranking → 204
	rankingReq := []sdk.DeviceSensorAssignmentRankingV1ModelV2{
		{UUID: assignmentUUID, Ranking: intPtr(1)},
	}
	_, err = svc.BulkUpdateDeviceSensorAssignmentRankingsAsync(ctx, sensorUUID, &rankingReq,
		&sdk.DeviceSensorsV2BulkUpdateDeviceSensorAssignmentRankingsAsyncOptions{Action: "update-ranking"})
	if err != nil {
		t.Fatalf("BulkUpdateDeviceSensorAssignmentRankingsAsync failed: %v", err)
	}

	// 7. Get assignment by id — GET /api/mdm/devicesensors/assignments/{assignmentUuid} → 200
	_, assignment, err := svc.GetDeviceSensorAssignmentAsync(ctx, assignmentUUID)
	if err != nil {
		t.Fatalf("GetDeviceSensorAssignmentAsync failed: %v", err)
	}
	if assignment == nil {
		t.Fatal("expected non-nil assignment response")
	}

	// 8. Update assignment — PUT /api/mdm/devicesensors/assignments/{assignmentUuid} → 204
	updateReq := &sdk.DeviceSensorAssignmentRequestV1ModelV2{
		Name:            "Batch3 Test Assignment Updated",
		TriggerType:     "SCHEDULE",
		SmartGroupUUIDs: []string{"bd3a7003-ee58-663f-6f18-1183b03a8627"},
	}
	_, _, err = svc.UpdateDeviceSensorAssignmentAsync(ctx, assignmentUUID, updateReq)
	if err != nil {
		t.Fatalf("UpdateDeviceSensorAssignmentAsync failed: %v", err)
	}

	// 9. Delete assignment — DELETE /api/mdm/devicesensors/assignments/{assignmentUuid} → 204
	_, err = svc.DeleteDeviceSensorAssignmentAsync(ctx, assignmentUUID)
	if err != nil {
		t.Fatalf("DeleteDeviceSensorAssignmentAsync failed: %v", err)
	}
}
