package tests

import (
	"context"
	"testing"

	mamv1 "github.com/euc-oss/terraform-sdk-uem/v26/internal/mam/v1"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestInternalAppsV1ZipGetDeleteBypassSurvivesTestdataAbsence proves
// get_internal_app_zip.json / delete_internal_app_zip.json's own bypass load
// paths survive testdata/ removal on their own. TestGeneratedInternalAppsV1Lifecycle_ZIP
// cannot prove this itself: its create step (step 2, before get/delete) fails
// for an unrelated, pre-existing reason even with testdata/ present (the
// lifecycle tests deliberately never set FileName on create -- see that
// test's own doc comment -- so create always hits the shared DEFAULT
// chunk-handler fixture, which is out of this fixture family's scope), so
// the lifecycle test's t.Fatalf at create means it never reaches get/delete
// to exercise their bypass loading at all. This test calls
// GetInternalAppByIdAsync/DeleteInternalAppAsync directly, skipping
// upload/create entirely, to prove those two fixtures' own bypass path
// independent of that unrelated blocker.
func TestInternalAppsV1ZipGetDeleteBypassSurvivesTestdataAbsence(t *testing.T) {
	ms := newMockServerFromBypassFixturesOnly(t, []string{
		"../internal/mockserver/testdata/synthetic-placeholders/internal-apps-v1/zip/get_internal_app_zip.json",
		"../internal/mockserver/testdata/synthetic-placeholders/internal-apps-v1/zip/delete_internal_app_zip.json",
	})
	defer ms.Close()

	c := mockserver.NewMockClient(t, ms)
	svc := mamv1.NewInternalAppsV1Service(c)
	ctx := context.Background()

	_, app, err := svc.GetInternalAppByIdAsync(ctx, 67007)
	if err != nil {
		t.Fatalf("GetInternalAppByIdAsync failed: %v", err)
	}
	if app == nil {
		t.Fatal("nil app")
	}

	_, err = svc.DeleteInternalAppAsync(ctx, 67007)
	if err != nil {
		t.Fatalf("DeleteInternalAppAsync failed: %v", err)
	}
}
