package tests

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// Test constants for the UpdatesV1 read-only endpoints. Values are arbitrary
// (the mockserver matches by path template, not by exact ID value) except
// where noted.
const (
	testUpdatesOrgGroupUUID = "666ff6cc-aa5b-3c07-feaa-3a95d3a4bd2c"
	testUpdatesUpdateUUID   = "08e56864-baa2-84e7-71b7-54711284a42e"
	// The testUpdatesDeploymentUUID constant is the deployment UUID shared by the create,
	// get, update, and delete mutating fixtures (updates_deployment_*.json)
	// — captured as a single deployment's lifecycle in one live capture, so
	// the same UUID recurs across all four fixture bodies.
	testUpdatesDeploymentUUID = "1bc333e5-b6ed-b450-37c4-0007f627bd62"
)

// TestGeneratedUpdatesV1ReadOnly exercises the 7 read-only UpdatesV1
// endpoints against real captured (and anonymized) fixtures from
// testdata/mock-responses/updates/.
//
// NOTE on scope: only 6 of the 7 endpoints have fixtures here.
// GetDeviceUpdateDeploymentDetails (GET /api/mdm/updates/deployments/{uuid})
// is NOT covered in THIS test — the read-only-pass capture org group
// (sdktest) had zero device update deployments at the time, so no real
// deployment existed to fetch details for. A later mutating-capture pass
// created a real deployment and captured this endpoint's response; it is
// now exercised by TestGeneratedUpdatesV1DeploymentLifecycle below.
func TestGeneratedUpdatesV1ReadOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewUpdatesV1Service(c)
	ctx := context.Background()

	// 1. SEARCH — GET /api/mdm/updates?organization_group_uuid={orgGroupUuid} → 200
	// Fixture: updates/updates_search.json
	_, searchResp, err := svc.GetDeviceUpdatesBySearchParameters(ctx, &sdk.UpdatesV1GetDeviceUpdatesBySearchParametersOptions{
		OrganizationGroupUUID: testUpdatesOrgGroupUUID,
	})
	if err != nil {
		t.Fatalf("GetDeviceUpdatesBySearchParameters failed: %v", err)
	}
	if searchResp == nil {
		t.Fatal("expected non-nil search response")
	}
	if searchResp.Total == nil || *searchResp.Total != 173 {
		t.Errorf("total: got %v, want 173", searchResp.Total)
	}
	if len(searchResp.UpdateList) != 50 {
		t.Fatalf("update_list length: got %d, want 50", len(searchResp.UpdateList))
	}
	var wantSearchFirst struct {
		UpdateList []struct {
			Name       string `json:"name"`
			UUID       string `json:"uuid"`
			UpdateType string `json:"update_type"`
		} `json:"update_list"`
	}
	loadFixtureResponseBodyForTest(t, "updates/updates_search.json", &wantSearchFirst)
	if len(wantSearchFirst.UpdateList) == 0 {
		t.Fatal("updates/updates_search.json fixture has an empty update_list -- cannot derive expected values")
	}
	wantFirst := wantSearchFirst.UpdateList[0]

	first := searchResp.UpdateList[0]
	if first.Name != wantFirst.Name {
		t.Errorf("first update name: got %q, want %q", first.Name, wantFirst.Name)
	}
	if first.UUID != wantFirst.UUID {
		t.Errorf("first update uuid: got %q, want %q", first.UUID, wantFirst.UUID)
	}
	if first.UpdateType != wantFirst.UpdateType {
		t.Errorf("first update type: got %q, want %q", first.UpdateType, wantFirst.UpdateType)
	}

	// 2. GET SPECIFIC UPDATE DETAILS — GET /api/mdm/updates/{uuid} → 200
	// Fixture: updates/updates_get_details.json
	_, details, err := svc.GetSpecificUpdateDetails(ctx, testUpdatesUpdateUUID)
	if err != nil {
		t.Fatalf("GetSpecificUpdateDetails failed: %v", err)
	}
	if details == nil {
		t.Fatal("expected non-nil details response")
	}
	if details.Name != "Test Rabbitthrow" {
		t.Errorf("details name: got %q, want %q", details.Name, "Test Rabbitthrow")
	}
	if details.Version != "17.3.1" {
		t.Errorf("details version: got %q, want %q", details.Version, "17.3.1")
	}
	if len(details.SupportedDevices) != 49 {
		t.Errorf("supported_devices length: got %d, want 49", len(details.SupportedDevices))
	}
	// Real-wire finding: the live server omits "description" and
	// "external_key" entirely when they are empty (JsonNetFormatter's
	// NullValueHandling.Ignore) rather than sending "". Confirm the
	// generated model still zero-values cleanly when the keys are absent.
	if details.Description != "" {
		t.Errorf("description: got %q, want empty (key absent on wire)", details.Description)
	}

	// 3. GET DEPLOYMENTS BY DEVICE UPDATE — GET /api/mdm/updates/{uuid}/deployments → 204
	// Fixture: updates/updates_deployments_empty.json
	// Real-wire finding: an update with zero deployments returns HTTP 204 No
	// Content, not HTTP 200 with an empty JSON array. Confirm the generated
	// service handles this without error and yields a nil/empty slice.
	_, deployments, err := svc.GetDeploymentsByDeviceUpdate(ctx, testUpdatesUpdateUUID, &sdk.UpdatesV1GetDeploymentsByDeviceUpdateOptions{
		OrganizationGroupUUID: testUpdatesOrgGroupUUID,
	})
	if err != nil {
		t.Fatalf("GetDeploymentsByDeviceUpdate failed: %v", err)
	}
	if deployments != nil && len(*deployments) != 0 {
		t.Errorf("expected no deployments on 204, got %+v", *deployments)
	}

	// 5. GET DEVICE READINESS — GET /api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}/device-readiness → 200
	// Fixture: updates/updates_device_readiness.json
	_, readiness, err := svc.GetDeviceReadiness(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID)
	if err != nil {
		t.Fatalf("GetDeviceReadiness failed: %v", err)
	}
	if readiness == nil {
		t.Fatal("expected non-nil readiness response")
	}
	if readiness.Eligible == nil || *readiness.Eligible != 0 {
		t.Errorf("eligible: got %v, want 0", readiness.Eligible)
	}
	if readiness.NotEligible == nil || *readiness.NotEligible != 0 {
		t.Errorf("not_eligible: got %v, want 0", readiness.NotEligible)
	}
	if readiness.AlreadyOnThisVersion == nil || *readiness.AlreadyOnThisVersion != 0 {
		t.Errorf("already_on_this_version: got %v, want 0", readiness.AlreadyOnThisVersion)
	}
	if readiness.OnHigherVersion == nil || *readiness.OnHigherVersion != 0 {
		t.Errorf("on_higher_version: got %v, want 0", readiness.OnHigherVersion)
	}

	// 6. GET COUNT OF DEVICE STATUS — GET /api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}/device-status → 404
	// Fixture: updates/updates_device_status_count_no_deployment.json
	// This endpoint structurally requires an existing deployment; since the
	// capture org group has none, the live server returns a 404 error for
	// every update UUID tried. That 404 is real, unmutated data — captured
	// here as the genuine error-path response rather than a fabricated
	// success payload.
	_, countResp, err := svc.GetCountOfDeviceStatus(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID)
	if err == nil {
		t.Fatal("expected error for GetCountOfDeviceStatus (no deployment exists), got nil")
	}
	if countResp != nil {
		t.Errorf("expected nil response on error, got %+v", countResp)
	}

	// 7. GET DEVICE UPDATE STATUS BY SEARCH PARAMETERS — GET /api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}/update-status → 200
	// Fixture: updates/updates_update_status.json
	_, statusResp, err := svc.GetDeviceUpdateStatusBySearchParameters(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID, nil)
	if err != nil {
		t.Fatalf("GetDeviceUpdateStatusBySearchParameters failed: %v", err)
	}
	if statusResp == nil {
		t.Fatal("expected non-nil status response")
	}
	if statusResp.Total == nil || *statusResp.Total != 0 {
		t.Errorf("total: got %v, want 0", statusResp.Total)
	}
	if len(statusResp.DeviceList) != 0 {
		t.Errorf("device_list length: got %d, want 0 (no deployments in this org group)", len(statusResp.DeviceList))
	}
}

// testUpdatesDeploymentStartTime builds the client.UEMTime shared by the
// create/update request bodies below, matching the wire value
// "2024-01-15T00:00:00Z" baked into the captured get fixture.
func testUpdatesDeploymentStartTime(t *testing.T) client.UEMTime {
	t.Helper()
	ts, err := client.ParseUEMTime("2024-01-15T00:00:00Z")
	if err != nil {
		t.Fatalf("parse deployment_start_time: %v", err)
	}
	return ts
}

// TestGeneratedUpdatesV1DeploymentLifecycle exercises the mutating
// create/get/update/delete UpdatesV1 endpoints against the live-captured
// (and anonymized) fixtures in testdata/mock-responses/updates/: create
// (201 + BaseModelV1), get (200, immediately-after-create shape), update
// (204 empty, confirmed to persist on the wire), delete (204 empty,
// confirmed by re-GET to actually remove the deployment on the wire).
//
// NOTE on "confirms persisted" / "confirms gone": the metadata.description
// on updates_deployment_update.json and updates_deployment_delete.json both
// record that the live capture re-GET'd the deployment afterward and saw
// the mutation land for real (renamed fields on update; a 404 on delete) —
// not a silent no-op. Those follow-up re-GET responses were NOT captured as
// their own fixture files, and this task's hard rule #1 forbids fabricating
// or re-capturing fixtures. The mock server also has no stateful tracking
// for the updates/deployments resource (unlike /api/mdm/profiles — see
// internal/mockserver/state.go), so a second GET against this mock would
// just replay the same static "immediately after create" body regardless
// of the update/delete calls in between. This test therefore only asserts
// that update and delete each complete without error against their real
// captured 204 contracts; the "did it actually persist/vanish on the real
// server" claim rests on the prose already recorded in the fixture
// metadata from the live capture, not on a mock re-GET.
func TestGeneratedUpdatesV1DeploymentLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewUpdatesV1Service(c)
	ctx := context.Background()

	startTime := testUpdatesDeploymentStartTime(t)

	// CREATE — POST /api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid}/deployment → 201 + BaseModelV1{id:0, uuid}.
	// Fixture: updates/updates_deployment_create.json
	createReq := &sdk.DeviceUpdateDeploymentBaseV1Model{
		Name:                "Test Deployment",
		DeploymentType:      "DOWNLOAD_ONLY",
		DeploymentStartTime: startTime,
		SmartGroupUUIDs:     []string{"c2e73a8e-0b57-a56f-d3ae-45c0a3f9c220"},
	}
	_, created, err := svc.CreateUpdateDeployment(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID, createReq)
	if err != nil {
		t.Fatalf("CreateUpdateDeployment failed: %v", err)
	}
	var wantCreated struct {
		UUID string `json:"uuid"`
	}
	loadFixtureResponseBodyForTest(t, "updates/updates_deployment_create.json", &wantCreated)
	if created == nil || created.UUID != wantCreated.UUID {
		t.Fatalf("expected created deployment UUID %q, got %+v", wantCreated.UUID, created)
	}
	if created.ID == nil || *created.ID != 0 {
		t.Errorf("expected id=0 (deployments are UUID-identified, not integer-ID-identified), got %v", created.ID)
	}

	// GET — GET /api/mdm/updates/deployments/{uuid} → 200, immediately-after-create shape.
	// Fixture: updates/updates_deployment_get.json
	// Also verifies GetDeviceUpdateDeploymentDetails, unverified after the
	// read-only pass for lack of any real deployment to fetch.
	_, got, err := svc.GetDeviceUpdateDeploymentDetails(ctx, testUpdatesDeploymentUUID)
	if err != nil {
		t.Fatalf("GetDeviceUpdateDeploymentDetails failed: %v", err)
	}
	var wantGot struct {
		UUID string `json:"uuid"`
	}
	loadFixtureResponseBodyForTest(t, "updates/updates_deployment_get.json", &wantGot)
	if got == nil || got.UUID != wantGot.UUID {
		t.Fatalf("expected fetched deployment UUID %q, got %+v", wantGot.UUID, got)
	}
	if got.Name != "Test Snowlung" {
		t.Errorf("name: got %q, want %q", got.Name, "Test Snowlung")
	}
	if got.Ranking == nil || *got.Ranking != 1 {
		t.Errorf("ranking: got %v, want 1", got.Ranking)
	}
	// Real-wire finding (see fixture description): allow_user_deferral,
	// max_number_of_deferrals, and priority are all omitted on a fresh V1
	// create rather than emitted as null/zero (NullValueHandling.Ignore).
	if got.AllowUserDeferral != nil {
		t.Errorf("allow_user_deferral: expected omitted (nil) on a fresh V1 create, got %v", *got.AllowUserDeferral)
	}
	if got.MaxNumberOfDeferrals != nil {
		t.Errorf("max_number_of_deferrals: expected omitted (nil) on a fresh V1 create, got %v", *got.MaxNumberOfDeferrals)
	}
	if got.Priority != "" {
		t.Errorf("priority: expected omitted (empty) on a fresh V1 create, got %q", got.Priority)
	}

	// UPDATE — PUT /api/mdm/updates/deployments/{uuid} → 204 empty, confirmed to persist on the wire.
	// Fixture: updates/updates_deployment_update.json
	updateReq := &sdk.DeviceUpdateDeploymentUpdateV1Model{
		Name:                "Test Deployment (renamed)",
		DeploymentType:      "DOWNLOAD_ONLY",
		DeploymentStartTime: startTime,
		SmartGroupUUIDs:     []string{"c2e73a8e-0b57-a56f-d3ae-45c0a3f9c220"},
	}
	if _, err := svc.UpdateDeviceUpdateDeployment(ctx, testUpdatesDeploymentUUID, updateReq); err != nil {
		t.Fatalf("UpdateDeviceUpdateDeployment failed: %v", err)
	}

	// DELETE — DELETE /api/mdm/updates/deployments/{uuid} → 204 empty, confirmed removed by re-GET (404) on the wire.
	// Fixture: updates/updates_deployment_delete.json
	if _, err := svc.DeleteDeviceUpdateDeployment(ctx, testUpdatesDeploymentUUID); err != nil {
		t.Fatalf("DeleteDeviceUpdateDeployment failed: %v", err)
	}
}

// TestGeneratedUpdatesV1CreateConflict exercises the 409 DeploymentExists
// conflict path for CreateUpdateDeployment (double-create against the same
// update+org group).
//
// This fixture (updates_deployment_create_conflict.json) shares its exact
// endpoint, method, and version with the success fixture
// (updates_deployment_create.json) — both are a bare POST .../deployment
// with no distinguishing query params — so loading the full
// testdata/mock-responses corpus together always resolves the tie in favor
// of the alphabetically-first-loaded create.json fixture (filepath.Walk
// visits create.json before create_conflict.json, and
// RequestMatcher.Match's `score > bestScore` check keeps the first response
// on an exact tie — see internal/mockserver/matcher.go). This is the same
// same-path/no-query-param collision the ScriptsV1 bulkdelete-partial test
// works around (TestGeneratedScriptsV1BulkDeletePartial in
// generated_scriptsv1_test.go): the only way to exercise the shadowed
// fixture is an isolated mock server loading just that one file directly
// via mockserver.LoadResponseFromFile, as done here. No mockserver code
// changes were needed — this is the established pattern for this exact
// class of collision.
func TestGeneratedUpdatesV1CreateConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/updates/updates_deployment_create_conflict.json"))
	if err != nil {
		t.Fatalf("load conflict fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewUpdatesV1Service(c)
	ctx := context.Background()

	createReq := &sdk.DeviceUpdateDeploymentBaseV1Model{
		Name:                "Test Deployment",
		DeploymentType:      "DOWNLOAD_ONLY",
		DeploymentStartTime: testUpdatesDeploymentStartTime(t),
		SmartGroupUUIDs:     []string{"c2e73a8e-0b57-a56f-d3ae-45c0a3f9c220"},
	}
	_, _, err = svc.CreateUpdateDeployment(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID, createReq)
	if err == nil {
		t.Fatal("expected error for duplicate deployment (409 DeploymentExists), got nil")
	}
	if !strings.Contains(err.Error(), "409") {
		t.Errorf("expected error to mention status 409, got: %v", err)
	}
	if !strings.Contains(err.Error(), "5206") {
		t.Errorf("expected error to mention errorCode 5206 (DeploymentExists), got: %v", err)
	}
}

// TestGeneratedUpdatesV1GetCountOfDeviceStatusSuccess exercises the 200
// success path for GetCountOfDeviceStatus (a live deployment exists),
// live-captured against a deployment created in a disposable sdktestrun*
// org group.
//
// This fixture (updates_device_status_count_success.json) shares its exact
// endpoint, method, and version with the pre-existing no-deployment 404
// fixture (updates_device_status_count_no_deployment.json) — both are a
// bare GET .../device-status with no distinguishing query params — so
// loading the full testdata/mock-responses corpus together resolves the tie
// arbitrarily by load order. Same collision class as
// TestGeneratedUpdatesV1CreateConflict above: exercised via an isolated
// mock server loading just this one file, so TestGeneratedUpdatesV1ReadOnly
// keeps exercising the genuine no-deployment 404 case undisturbed.
func TestGeneratedUpdatesV1GetCountOfDeviceStatusSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/updates/updates_device_status_count_success.json"))
	if err != nil {
		t.Fatalf("load device status count fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewUpdatesV1Service(c)
	ctx := context.Background()

	_, countResp, err := svc.GetCountOfDeviceStatus(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID)
	if err != nil {
		t.Fatalf("GetCountOfDeviceStatus failed: %v", err)
	}
	if countResp == nil {
		t.Fatal("expected non-nil count response")
	}
	if countResp.Idle == nil || *countResp.Idle != 0 {
		t.Errorf("idle: got %v, want 0", countResp.Idle)
	}
	if countResp.NotStarted == nil || *countResp.NotStarted != 0 {
		t.Errorf("not_started: got %v, want 0", countResp.NotStarted)
	}
}

// TestGeneratedUpdatesV1BulkRanking exercises
// BulkUpdateDeviceUpdateDeployment (POST
// .../deployments?action=update-ranking) against
// updates_deployments_bulk_rank.json. The plural .../deployments path is
// distinct from the singular .../deployment create path, so it does not
// collide with the create/create-conflict fixtures above.
//
// PRECONDITION — DO NOT "simplify" this to a single smart group: bulk
// ranking only works when the sibling deployments being ranked live on
// DIFFERENT smart groups under the same update+org group. Two deployments
// targeting the SAME smart group trip the 409 DeploymentExists conflict at
// creation time instead (errorCode 5206 — see
// updates_deployment_create_conflict.json and
// TestGeneratedUpdatesV1CreateConflict above). This was discovered
// empirically during live capture and is recorded in this fixture's
// metadata.description. That's why the two ranking entries below reference
// two distinct deployment UUIDs (implicitly on two distinct smart groups) —
// collapsing them to one smart group would break the precondition this test
// exists to exercise, in a confusing way (the mock would still return 204,
// but the live server would not).
//
// NOTE on "ranking values actually change": the live capture confirmed via
// a per-deployment re-GET after this call that ranking actually swapped
// (1<->2), i.e. this is not a silent no-op — see the fixture's
// metadata.description. That readback was not captured as its own fixture
// (no stateful tracking for this resource in the mock server, and hard
// rule #1 for this task forbids fabricating/re-capturing fixtures), so
// this test can only confirm the bulk-rank call itself succeeds against
// the real captured 204 contract; it cannot independently re-verify the
// mutation against the static mock.
func TestGeneratedUpdatesV1BulkRanking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewUpdatesV1Service(c)
	ctx := context.Background()

	rankFirst := 2
	rankSecond := 1
	rankings := []sdk.DeviceUpdateDeploymentRankingV1Model{
		// Deployment on smart group A, promoted to ranking 2.
		{UUID: "ec4a352b-6409-c5a7-84b0-6c192258c3ba", Ranking: &rankFirst},
		// Sibling deployment on smart group B (DIFFERENT smart group — see
		// precondition above), demoted to ranking 1.
		{UUID: "a0806726-141a-3a35-4288-74bfc7844f1b", Ranking: &rankSecond},
	}
	opts := &sdk.UpdatesV1BulkUpdateDeviceUpdateDeploymentOptions{Action: "update-ranking"}
	headers, err := svc.BulkUpdateDeviceUpdateDeployment(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID, &rankings, opts)
	if err != nil {
		t.Fatalf("BulkUpdateDeviceUpdateDeployment failed: %v", err)
	}
	if headers == nil {
		t.Error("expected non-nil response headers on the 204 response")
	}
}

// TestGeneratedUpdatesV1DevicesUpdateAction exercises Start, Stop, and an
// invalid action value — all three hit the exact same path template
// (POST /api/mdm/updates/{updateUuid}/groups/{organizationGroupUuid})
// against the SAME auto-loaded fixture corpus in one mock server.
//
// All three fixtures (updates_group_action_start.json,
// updates_group_action_stop.json, updates_group_action_invalid.json) share
// an identical endpoint/method/version; only the `action` query parameter
// differs. This exercises the mockserver's existing query-param scoring
// (RequestMatcher.scoreMatch → matchQueryParams in
// internal/mockserver/matcher.go, +10 per matching key/value pair): a
// request with action=Start scores 10 higher against the Start fixture than
// against Stop/invalid (which score 0 extra since "Start" != their expected
// value), so Start always wins for that request, and likewise for Stop and
// the invalid value. No mockserver code changes were needed for this —
// the existing scoring already disambiguates correctly.
func TestGeneratedUpdatesV1DevicesUpdateAction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewUpdatesV1Service(c)
	ctx := context.Background()

	// START — action=Start → 202 Accepted, empty body.
	// Fixture: updates/updates_group_action_start.json
	startHeaders, err := svc.DevicesUpdateAction(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID,
		&sdk.UpdatesV1DevicesUpdateActionOptions{Action: "Start"})
	if err != nil {
		t.Fatalf("DevicesUpdateAction(Start) failed: %v", err)
	}
	if cl := startHeaders.Get("Content-Length"); cl != "" && cl != "0" {
		t.Errorf("expected empty body on Start 202 (Content-Length \"\" or \"0\"), got %q", cl)
	}

	// STOP — action=Stop → 202 Accepted, empty body.
	// Fixture: updates/updates_group_action_stop.json
	stopHeaders, err := svc.DevicesUpdateAction(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID,
		&sdk.UpdatesV1DevicesUpdateActionOptions{Action: "Stop"})
	if err != nil {
		t.Fatalf("DevicesUpdateAction(Stop) failed: %v", err)
	}
	if cl := stopHeaders.Get("Content-Length"); cl != "" && cl != "0" {
		t.Errorf("expected empty body on Stop 202 (Content-Length \"\" or \"0\"), got %q", cl)
	}

	// INVALID — action=Pause → 422, errorCode 1012 (generic model-validation
	// shape; NOT the 400/errorCode 5222 canonical docs describe — see
	// updates_group_action_invalid.json's metadata.description for the
	// wire-vs-docs disagreement).
	// Fixture: updates/updates_group_action_invalid.json
	_, err = svc.DevicesUpdateAction(ctx, testUpdatesUpdateUUID, testUpdatesOrgGroupUUID,
		&sdk.UpdatesV1DevicesUpdateActionOptions{Action: "Pause"})
	if err == nil {
		t.Fatal("expected error for invalid action value, got nil")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("expected error to mention status 422, got: %v", err)
	}
	if !strings.Contains(err.Error(), "1012") {
		t.Errorf("expected error to mention errorCode 1012, got: %v", err)
	}
}
