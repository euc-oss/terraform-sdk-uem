package tests

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestMamV2KioskBookmarksFixture verifies AppBookmarksV2Service.GetBookmarkList
// against the genuine live capture for the ANDROID platform (empty list — no
// bookmarks configured on the capture tenant).
// Fixture: testdata/mock-responses/mam-apps/apps_get_kiosk-bookmarks_android.json.
func TestMamV2KioskBookmarksFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/apps_get_kiosk-bookmarks_android.json"))
	if err != nil {
		t.Fatalf("load apps_get_kiosk-bookmarks_android fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewAppBookmarksV2Service(c)
	ctx := context.Background()

	_, result, err := svc.GetBookmarkList(ctx, "android", nil)
	if err != nil {
		t.Fatalf("GetBookmarkList failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(*result) != 0 {
		t.Errorf("expected 0 bookmarks (empty capture), got %d", len(*result))
	}
}

// TestGeneratedAppRemovalProtectionLogsV2Lifecycle verifies
// AppRemovalProtectionLogsV2Service.AppRemovalDevices against the genuine
// live capture, which is itself a real 412 error response — the ARP Angular
// Migration Events feature is turned off on the capture tenant. The
// consuming behavior under test is that the SDK surfaces this as a
// client.APIError with the wire errorCode/message intact, not that the
// endpoint returns data.
// Fixture: testdata/mock-responses/mam-apps/apps_get_removal-protection-logs_devices.json.
func TestGeneratedAppRemovalProtectionLogsV2Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/apps_get_removal-protection-logs_devices.json"))
	if err != nil {
		t.Fatalf("load apps_get_removal-protection-logs_devices fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewAppRemovalProtectionLogsV2Service(c)
	ctx := context.Background()

	_, _, err = svc.AppRemovalDevices(ctx, &sdk.AppRemovalProtectionLogsV2AppRemovalDevicesOptions{
		OrganizationGroupUUID: "7d0aef64-735e-853e-1695-fdb07ba63f5c",
	})
	if err == nil {
		t.Fatal("expected an error surfacing the live 412 capture, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 412 {
		t.Errorf("StatusCode: got %d, want 412", apiErr.StatusCode)
	}
	if apiErr.ErrorCode != "7026" {
		t.Errorf("ErrorCode: got %q, want 7026", apiErr.ErrorCode)
	}

	// Control: a nil opts must fail client-side, before ever reaching this
	// fixture (organization_group_uuid is required).
	_, _, err = svc.AppRemovalDevices(ctx, nil)
	if err == nil {
		t.Error("expected an error when opts (containing required organization_group_uuid) is nil, got nil")
	}
}

// TestMamV2AppConfigTemplateFixture verifies AppsV2Service.GetAppConfigTemplateAsync
// against the genuine live capture (empty template list on the capture tenant).
// Fixture: testdata/mock-responses/mam-apps/apps_get_app-config-template.json.
func TestMamV2AppConfigTemplateFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/apps_get_app-config-template.json"))
	if err != nil {
		t.Fatalf("load apps_get_app-config-template fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewAppsV2Service(c)
	ctx := context.Background()

	_, result, err := svc.GetAppConfigTemplateAsync(ctx, "f3be5b88-fb76-176d-845b-0be80ea3f3ad", nil)
	if err != nil {
		t.Fatalf("GetAppConfigTemplateAsync failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(*result) != 0 {
		t.Errorf("expected 0 template entries (empty capture), got %d", len(*result))
	}
}

// TestMamV2AppsCategoriesFixture verifies AppsV2Service.GetCategoriesForApplication
// against the genuine live capture (categorytype=Application, empty category
// list on the capture tenant).
// Fixture: testdata/mock-responses/mam-apps/apps_get_categories.json.
func TestMamV2AppsCategoriesFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/apps_get_categories.json"))
	if err != nil {
		t.Fatalf("load apps_get_categories fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewAppsV2Service(c)
	ctx := context.Background()

	_, result, err := svc.GetCategoriesForApplication(ctx, &sdk.AppsV2GetCategoriesForApplicationOptions{
		Categorytype: "Application",
		Oguuid:       "7d0aef64-735e-853e-1695-fdb07ba63f5c",
	})
	if err != nil {
		t.Fatalf("GetCategoriesForApplication failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Categories) != 0 {
		t.Errorf("expected 0 categories (empty capture), got %d", len(result.Categories))
	}

	// Control: required query params (categorytype/oguuid) must be enforced
	// client-side when opts is nil.
	_, _, err = svc.GetCategoriesForApplication(ctx, nil)
	if err == nil {
		t.Error("expected an error when opts (containing required categorytype/oguuid) is nil, got nil")
	}
}

// TestMamV2BranchCacheFixture verifies
// InternalAppsV2Service.GetApplicationBranchCacheStatisticsAsync against the
// genuine live capture (all-zero BranchCache summary — no devices utilizing
// BranchCache on the capture tenant).
// Fixture: testdata/mock-responses/mam-apps/apps_get_internal_peerdistribution_branchcache.json.
func TestMamV2BranchCacheFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/apps_get_internal_peerdistribution_branchcache.json"))
	if err != nil {
		t.Fatalf("load apps_get_internal_peerdistribution_branchcache fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewInternalAppsV2Service(c)
	ctx := context.Background()

	_, result, err := svc.GetApplicationBranchCacheStatisticsAsync(ctx, "com.example.forklift", nil)
	if err != nil {
		t.Fatalf("GetApplicationBranchCacheStatisticsAsync failed: %v", err)
	}
	if result == nil || result.BranchcacheStatistics == nil {
		t.Fatal("expected non-nil BranchcacheStatistics")
	}
	if result.BranchcacheStatistics.TotalDeviceCount == nil || *result.BranchcacheStatistics.TotalDeviceCount != 0 {
		t.Errorf("expected TotalDeviceCount=0, got %+v", result.BranchcacheStatistics.TotalDeviceCount)
	}
}
