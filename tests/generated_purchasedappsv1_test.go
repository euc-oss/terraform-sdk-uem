package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedPurchasedAppsV1SearchEmpty exercises the generated
// PurchasedAppsV1Service.VppAppSearchAsync against
// testdata/mock-responses/mam-apps/apps_get_purchased_search.json.
//
// The fixture is a genuine 204 No Content capture (no VPP token on the
// capturing tenant), so this test asserts on the empty-result path rather
// than a populated search response -- that is the real, fixture-backed
// behavior of this operation on the captured tenant.
func TestGeneratedPurchasedAppsV1SearchEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewPurchasedAppsV1Service(c)

	_, resp, err := svc.VppAppSearchAsync(context.Background(), nil)
	if err != nil {
		t.Fatalf("VppAppSearchAsync failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected a non-nil (zero-value) response struct for 204 No Content, got nil")
	}
	if len(resp.Application) != 0 {
		t.Fatalf("expected empty Application slice for 204 No Content fixture, got %+v", resp.Application)
	}
}
