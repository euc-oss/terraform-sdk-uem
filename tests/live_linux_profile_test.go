package tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/orgtest"
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// linuxProfileV4AcceptHeader mirrors internal/mdm/v4's AcceptHeader
// constant ("application/json;version=4", internal/mdm/v4/constants.go) --
// duplicated here as a literal rather than importing the internal mdm/v4
// package directly, since this file already reaches the same generated
// types/service through the public sdk alias (sdk.LinuxDeviceProfileEntity1V4,
// sdk.NewProfilesV4Service) per api-doctrine.md quirk 3 (Linux profiles use
// API v4 at a different path from every other platform).
const linuxProfileV4AcceptHeader = "application/json;version=4"

// linuxProfileV4CreateEndpoint is the live-validated-but-untracked Linux
// profile create route (testdata/request-templates/profiles/linux/wifi/
// wifi-create-v4.json's own api_validation.endpoint field). It has no
// generated Layer-1 method: ProfilesV4Service (internal/mdm/v4/profilesv4.go)
// only generates UpdateLinuxDeviceProfileAsync -- this SDK's generator
// configuration never declared a create operation for this path (confirmed
// by grep). This mirrors the "no SDK method exists for a live-validated
// route" situation api-doctrine.md quirk
// 16 documents for BlobsV1's delete -- the fix there was a raw HTTP call at
// the call site, which is what this test does too, via client.DoRequest
// directly rather than guessing at a generated method that doesn't exist.
const linuxProfileV4CreateEndpoint = "/api/mdm/profiles/platforms/linux/create"

// TestLive_LinuxWifiCreate_FullLiveWithOGContainment hand-writes the
// full-live-with-OG-containment tier for the Linux wifi create
// request-template (hand-written, not template-generated -- see
// .claude/rules/api-doctrine.md and testdata/live-policy.yaml's ProfilesV4
// create entry). Posts testdata/request-templates/profiles/linux/wifi/
// wifi-create-v4.json's body, unmarshaled into the real generated
// LinuxDeviceProfileEntity1V4 type, directly via client.DoRequest (no
// generated Create method exists for this route -- see
// linuxProfileV4CreateEndpoint's doc comment).
func TestLive_LinuxWifiCreate_FullLiveWithOGContainment(t *testing.T) {
	skipUnlessLiveMutate(t)
	c := getTestClient(t, "basic")
	ctx := context.Background()
	ogID := orgtest.NewDisposableOrgGroupOrFatal(t, c, liveOrgGroupID(t), nil)

	data, err := os.ReadFile("../testdata/request-templates/profiles/linux/wifi/wifi-create-v4.json")
	if err != nil {
		t.Fatalf("read request template: %v", err)
	}
	var req sdk.LinuxDeviceProfileEntity1V4
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal request template: %v", err)
	}
	if req.General == nil {
		t.Fatal("request template has no General payload")
	}
	req.General.ManagedLocationGroupID = intPtr(ogID) // override to the disposable OG

	var profileID int
	if _, err := c.DoRequest(ctx, "POST", linuxProfileV4CreateEndpoint, linuxProfileV4AcceptHeader, "application/json", &req, &profileID); err != nil {
		t.Fatalf("POST %s: %v", linuxProfileV4CreateEndpoint, err)
	}
	// Registered AFTER the OG create above, so LIFO fires this delete
	// BEFORE the OG's own delete cleanup -- do not depend on OG-delete
	// cascading to child profiles.
	profileSvc := resources.NewProfileService(c)
	t.Cleanup(func() {
		if err := profileSvc.Delete(context.Background(), profileID); err != nil {
			t.Logf("cleanup: ProfileService.Delete(%d) failed: %v", profileID, err)
		}
	})
}

// TestLive_LinuxWifiUpdate_FullLiveWithOGContainment hand-writes the
// full-live-with-OG-containment tier for the Linux wifi update
// request-template, using the real generated
// ProfilesV4Service.UpdateLinuxDeviceProfileAsync method. Creates its own
// profile first (via the same raw-DoRequest path as
// TestLive_LinuxWifiCreate_FullLiveWithOGContainment above, since Update
// needs a real profile ID to target) rather than depending on that other
// test's ordering or side effects.
func TestLive_LinuxWifiUpdate_FullLiveWithOGContainment(t *testing.T) {
	skipUnlessLiveMutate(t)
	c := getTestClient(t, "basic")
	ctx := context.Background()
	ogID := orgtest.NewDisposableOrgGroupOrFatal(t, c, liveOrgGroupID(t), nil)

	createData, err := os.ReadFile("../testdata/request-templates/profiles/linux/wifi/wifi-create-v4.json")
	if err != nil {
		t.Fatalf("read create request template: %v", err)
	}
	var createReq sdk.LinuxDeviceProfileEntity1V4
	if err := json.Unmarshal(createData, &createReq); err != nil {
		t.Fatalf("unmarshal create request template: %v", err)
	}
	if createReq.General == nil {
		t.Fatal("create request template has no General payload")
	}
	createReq.General.ManagedLocationGroupID = intPtr(ogID)

	var profileID int
	if _, err := c.DoRequest(ctx, "POST", linuxProfileV4CreateEndpoint, linuxProfileV4AcceptHeader, "application/json", &createReq, &profileID); err != nil {
		t.Fatalf("POST %s (setup create): %v", linuxProfileV4CreateEndpoint, err)
	}
	if profileID == 0 {
		t.Fatal("setup create returned zero profile id")
	}
	// Registered AFTER the OG create above, so LIFO fires this delete
	// BEFORE the OG's own delete cleanup -- do not depend on OG-delete
	// cascading to child profiles.
	profileSvc := resources.NewProfileService(c)
	t.Cleanup(func() {
		if err := profileSvc.Delete(context.Background(), profileID); err != nil {
			t.Logf("cleanup: ProfileService.Delete(%d) failed: %v", profileID, err)
		}
	})

	updateData, err := os.ReadFile("../testdata/request-templates/profiles/linux/wifi/wifi-update-v4.json")
	if err != nil {
		t.Fatalf("read update request template: %v", err)
	}
	var updateReq sdk.LinuxDeviceProfileEntity1V4
	if err := json.Unmarshal(updateData, &updateReq); err != nil {
		t.Fatalf("unmarshal update request template: %v", err)
	}
	if updateReq.General == nil {
		t.Fatal("update request template has no General payload")
	}
	updateReq.General.ManagedLocationGroupID = intPtr(ogID)
	updateReq.General.ProfileID = intPtr(profileID) // target the profile created above

	svc := sdk.NewProfilesV4Service(c)
	if _, err := svc.UpdateLinuxDeviceProfileAsync(ctx, &updateReq); err != nil {
		t.Fatalf("UpdateLinuxDeviceProfileAsync: %v", err)
	}
}
