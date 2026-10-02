package tests

import (
	"context"
	"testing"

	mdmv1 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// The four tests below exercise genuine live captures against a persistent
// hand-created baseline (test_baseline_sdk in the readonly_fixtures
// organization group): a pre-assignment empty GetBaselineAssignmentsAsync
// result, an empty GetBaselineDevicesAsync result (the tenant has no
// enrolled devices), a CloneBaselineAsync response proving the clone landed
// in a DIFFERENT (disposable) organization group than the source baseline's
// own OG, and an AssignBaselineAsync 204 against that clone. Each shares its
// endpoint/method/version with a pre-existing demoted synthetic-placeholder
// fixture of the same name family (see TestGeneratedBaselinesV1Lifecycle's
// doc comment), so each is loaded via its own isolated single-fixture mock
// server rather than the full directory scan -- same collision-avoidance
// pattern as TestGeneratedScriptsV1GetLiveCapture.

func TestGeneratedBaselinesV1AssignmentsGetLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/baselines/assignments_get_live.json"))
	if err != nil {
		t.Fatalf("load assignments_get_live fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := mdmv1.NewBaselinesV1Service(c)
	ctx := context.Background()

	_, assignments, err := svc.GetBaselineAssignmentsAsync(ctx, "any-og-uuid", "any-baseline-uuid")
	if err != nil {
		t.Fatalf("GetBaselineAssignmentsAsync failed: %v", err)
	}
	if assignments == nil {
		t.Fatal("expected non-nil assignments response")
	}
	if len(*assignments) != 0 {
		t.Errorf("expected 0 assignments (genuine pre-assignment capture), got %+v", assignments)
	}
}

func TestGeneratedBaselinesV1DevicesLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/baselines/devices_live.json"))
	if err != nil {
		t.Fatalf("load devices_live fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := mdmv1.NewBaselinesV1Service(c)
	ctx := context.Background()

	_, devices, err := svc.GetBaselineDevicesAsync(ctx, "any-og-uuid", "any-baseline-uuid")
	if err != nil {
		t.Fatalf("GetBaselineDevicesAsync failed: %v", err)
	}
	if devices == nil {
		t.Fatal("expected non-nil devices response")
	}
	if devices.Total == nil || *devices.Total != 0 {
		t.Errorf("expected Total 0 (genuine empty-tenant capture: no enrolled devices), got %+v", devices.Total)
	}
	if len(devices.Results) != 0 {
		t.Errorf("expected 0 device results, got %+v", devices.Results)
	}
}

func TestGeneratedBaselinesV1CloneLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/baselines/clone_response_live.json"))
	if err != nil {
		t.Fatalf("load clone_response_live fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := mdmv1.NewBaselinesV1Service(c)
	ctx := context.Background()

	req := &mdmv1.CreateBaselineRequestV1Model{Name: "clone-target-name-unused-by-mock", RootLocationGroupID: 99999}
	opts := &mdmv1.BaselinesV1CloneBaselineAsyncOptions{Action: "clone"}
	_, clone, err := svc.CloneBaselineAsync(ctx, "source-og-uuid", "source-baseline-uuid", req, opts)
	if err != nil {
		t.Fatalf("CloneBaselineAsync failed: %v", err)
	}
	if clone == nil {
		t.Fatal("expected non-nil clone response")
	}
	if clone.BaselineUUID == "" {
		t.Error("expected a non-empty cloned BaselineUUID")
	}
	if clone.RootLocationGroupID == nil || *clone.RootLocationGroupID != 12346 {
		t.Errorf("expected RootLocationGroupID 12346 (anonymized disposable-OG id, distinct from the source OG), got %+v", clone.RootLocationGroupID)
	}
	if clone.RootLocationGroupUUID == "" {
		t.Error("expected a non-empty RootLocationGroupUUID")
	}
}

func TestGeneratedBaselinesV1AssignLiveCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/baselines/assignments_post_live.json"))
	if err != nil {
		t.Fatalf("load assignments_post_live fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := mdmv1.NewBaselinesV1Service(c)
	ctx := context.Background()

	smartGroupID := 467674
	req := &mdmv1.BaselineAssignmentRequestV1Model{SmartGroups: []*int{&smartGroupID}}
	_, err = svc.AssignBaselineAsync(ctx, "clone-og-uuid", "clone-baseline-uuid", req)
	if err != nil {
		t.Fatalf("AssignBaselineAsync failed: %v", err)
	}
}
